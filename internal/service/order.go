package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"telegram-shop/internal/config"
	"telegram-shop/internal/model"
	"telegram-shop/internal/repository"
	"telegram-shop/internal/sepay"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderService handles order and purchase business logic.
type OrderService struct {
	db              *pgxpool.Pool
	cfg             *config.Config
	orderRepo       *repository.OrderRepo
	userRepo        *repository.UserRepo
	itemRepo        *repository.ItemRepo
	productLinkRepo *repository.ProductLinkRepo
	paymentRepo     *repository.PaymentRepo
	walletRepo      *repository.WalletRepo
}

// NewOrderService creates a new OrderService.
func NewOrderService(
	db *pgxpool.Pool,
	cfg *config.Config,
	orderRepo *repository.OrderRepo,
	userRepo *repository.UserRepo,
	itemRepo *repository.ItemRepo,
	productLinkRepo *repository.ProductLinkRepo,
	paymentRepo *repository.PaymentRepo,
	walletRepo *repository.WalletRepo,
) *OrderService {
	return &OrderService{
		db:              db,
		cfg:             cfg,
		orderRepo:       orderRepo,
		userRepo:        userRepo,
		itemRepo:        itemRepo,
		productLinkRepo: productLinkRepo,
		paymentRepo:     paymentRepo,
		walletRepo:      walletRepo,
	}
}

// PurchaseWithBalance processes a purchase using wallet balance.
// Everything runs in a single transaction for atomicity.
func (s *OrderService) PurchaseWithBalance(ctx context.Context, userID, itemID uuid.UUID, quantity int) (*model.Order, []model.ProductLink, error) {
	if quantity <= 0 {
		quantity = 1
	}

	// Get item info
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, nil, fmt.Errorf("find item: %w", err)
	}
	if item == nil || !item.IsActive {
		return nil, nil, errors.New("sản phẩm không tồn tại hoặc đã ngừng bán")
	}

	count, err := s.itemRepo.CountAvailableLinks(ctx, itemID)
	if err != nil {
		return nil, nil, fmt.Errorf("count links: %w", err)
	}
	if count < quantity {
		return nil, nil, fmt.Errorf("sản phẩm hiện chỉ còn %d trong kho, không đủ số lượng yêu cầu (%d)", count, quantity)
	}

	totalAmount := item.Price * int64(quantity)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = s.userRepo.UpdateBalance(ctx, tx, userID, -totalAmount)
	if err != nil {
		return nil, nil, fmt.Errorf("số dư không đủ để thanh toán")
	}

	order := &model.Order{
		UserID:        userID,
		ItemID:        itemID,
		Amount:        totalAmount,
		Quantity:      quantity,
		Status:        model.OrderStatusPaid,
		PaymentMethod: model.PaymentMethodWallet,
	}
	if err := s.orderRepo.Create(ctx, tx, order); err != nil {
		return nil, nil, fmt.Errorf("create order: %w", err)
	}

	// Lock and claim requested number of links
	links, err := s.productLinkRepo.ClaimLinks(ctx, tx, itemID, quantity)
	if err != nil {
		return nil, nil, fmt.Errorf("claim links: %w", err)
	}
	if len(links) < quantity {
		// Out of stock rollback
		_, _ = s.userRepo.UpdateBalance(ctx, tx, userID, totalAmount)
		return nil, nil, errors.New("sản phẩm đã hết hoặc không đủ hàng, số tiền đã được hoàn lại")
	}

	linkIDs := make([]uuid.UUID, len(links))
	for i, l := range links {
		linkIDs[i] = l.ID
	}

	if err := s.productLinkRepo.AssignLinks(ctx, tx, linkIDs, userID, order.ID); err != nil {
		return nil, nil, fmt.Errorf("assign links: %w", err)
	}

	if err := s.orderRepo.SetProductLink(ctx, tx, order.ID, links[0].ID); err != nil {
		return nil, nil, fmt.Errorf("set product link: %w", err)
	}

	wt := &model.WalletTransaction{
		UserID:      userID,
		Amount:      -totalAmount,
		Type:        model.WalletTypePurchase,
		Status:      "SUCCESS",
		Description: fmt.Sprintf("Mua %s (x%d)", item.Name, quantity),
		ReferenceID: order.ID.String(),
	}
	if err := s.walletRepo.Create(ctx, tx, wt); err != nil {
		return nil, nil, fmt.Errorf("create wallet tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit: %w", err)
	}

	log.Printf("Purchase completed: order=%s, item=%s, quantity=%d, amount=%d", order.ID, itemID, quantity, totalAmount)
	return order, links, nil
}

// PurchaseWithQR creates a pending order and returns a QR payment URL.
func (s *OrderService) PurchaseWithQR(ctx context.Context, userID, itemID uuid.UUID, quantity int) (*model.Order, *model.Payment, string, error) {
	if quantity <= 0 {
		quantity = 1
	}

	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("find item: %w", err)
	}
	if item == nil || !item.IsActive {
		return nil, nil, "", errors.New("sản phẩm không tồn tại hoặc đã ngừng bán")
	}

	count, err := s.itemRepo.CountAvailableLinks(ctx, itemID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("count links: %w", err)
	}
	if count < quantity {
		return nil, nil, "", fmt.Errorf("sản phẩm chỉ còn %d trong kho, không đủ số lượng yêu cầu (%d)", count, quantity)
	}

	totalAmount := item.Price * int64(quantity)

	order := &model.Order{
		UserID:        userID,
		ItemID:        itemID,
		Amount:        totalAmount,
		Quantity:      quantity,
		Status:        model.OrderStatusPending,
		PaymentMethod: model.PaymentMethodQR,
	}
	if err := s.orderRepo.CreateNoTx(ctx, order); err != nil {
		return nil, nil, "", fmt.Errorf("create order: %w", err)
	}

	shortID := strings.ToUpper(order.ID.String()[:8])
	transferContent := sepay.GenerateTransferContent(shortID)
	qrURL := sepay.GenerateQRURL(s.cfg.SepayBankCode, s.cfg.SepayAccountNumber, totalAmount, transferContent)

	payment := &model.Payment{
		OrderID:         &order.ID,
		UserID:          userID,
		Provider:        "SEPAY",
		Amount:          totalAmount,
		Status:          model.PaymentStatusPending,
		QRURL:           qrURL,
		TransferContent: transferContent,
		PaymentType:     model.PaymentTypeOrder,
		ExpiredAt:       time.Now().Add(15 * time.Minute),
	}
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, nil, "", fmt.Errorf("create payment: %w", err)
	}

	log.Printf("QR payment created: order=%s, payment=%s, quantity=%d, content=%s", order.ID, payment.ID, quantity, transferContent)
	return order, payment, qrURL, nil
}

// GetUserOrders returns orders for a user.
func (s *OrderService) GetUserOrders(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.OrderDetail, error) {
	return s.orderRepo.FindByUserID(ctx, userID, limit, offset)
}

// GetAllOrders returns all orders (admin).
func (s *OrderService) GetAllOrders(ctx context.Context, limit, offset int) ([]model.OrderDetail, error) {
	return s.orderRepo.FindAll(ctx, limit, offset)
}
