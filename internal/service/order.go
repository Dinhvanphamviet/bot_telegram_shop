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
func (s *OrderService) PurchaseWithBalance(ctx context.Context, userID, itemID uuid.UUID) (*model.Order, *model.ProductLink, error) {
	// Get item info
	item, err := s.itemRepo.FindByID(ctx, itemID)
	if err != nil {
		return nil, nil, fmt.Errorf("find item: %w", err)
	}
	if item == nil || !item.IsActive {
		return nil, nil, errors.New("sản phẩm không tồn tại hoặc đã ngừng bán")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = s.userRepo.UpdateBalance(ctx, tx, userID, -item.Price)
	if err != nil {
		return nil, nil, fmt.Errorf("số dư không đủ để thanh toán")
	}

	order := &model.Order{
		UserID:        userID,
		ItemID:        itemID,
		Amount:        item.Price,
		Status:        model.OrderStatusPaid,
		PaymentMethod: model.PaymentMethodWallet,
	}
	if err := s.orderRepo.Create(ctx, tx, order); err != nil {
		return nil, nil, fmt.Errorf("create order: %w", err)
	}

	// Lock and claim single available link
	link, err := s.productLinkRepo.ClaimLink(ctx, tx, itemID)
	if err != nil {
		return nil, nil, fmt.Errorf("claim link: %w", err)
	}
	if link == nil {
		// Out of stock rollback
		_, _ = s.userRepo.UpdateBalance(ctx, tx, userID, item.Price)
		return nil, nil, errors.New("sản phẩm đã hết hàng, số tiền đã được hoàn lại")
	}

	if err := s.productLinkRepo.AssignLink(ctx, tx, link.ID, userID, order.ID); err != nil {
		return nil, nil, fmt.Errorf("assign link: %w", err)
	}

	if err := s.orderRepo.SetProductLink(ctx, tx, order.ID, link.ID); err != nil {
		return nil, nil, fmt.Errorf("set product link: %w", err)
	}

	wt := &model.WalletTransaction{
		UserID:      userID,
		Amount:      -item.Price,
		Type:        model.WalletTypePurchase,
		Status:      "SUCCESS",
		Description: fmt.Sprintf("Mua %s", item.Name),
		ReferenceID: order.ID.String(),
	}
	if err := s.walletRepo.Create(ctx, tx, wt); err != nil {
		return nil, nil, fmt.Errorf("create wallet tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit: %w", err)
	}

	log.Printf("Purchase completed: order=%s, item=%s, link=%s", order.ID, itemID, link.ID)
	return order, link, nil
}

// PurchaseWithQR creates a pending order and returns a QR payment URL.
func (s *OrderService) PurchaseWithQR(ctx context.Context, userID, itemID uuid.UUID) (*model.Order, *model.Payment, string, error) {
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
	if count == 0 {
		return nil, nil, "", errors.New("sản phẩm đã hết hàng")
	}

	order := &model.Order{
		UserID:        userID,
		ItemID:        itemID,
		Amount:        item.Price,
		Status:        model.OrderStatusPending,
		PaymentMethod: model.PaymentMethodQR,
	}
	if err := s.orderRepo.CreateNoTx(ctx, order); err != nil {
		return nil, nil, "", fmt.Errorf("create order: %w", err)
	}

	shortID := strings.ToUpper(order.ID.String()[:8])
	transferContent := sepay.GenerateTransferContent(shortID)
	qrURL := sepay.GenerateQRURL(s.cfg.SepayBankCode, s.cfg.SepayAccountNumber, item.Price, transferContent)

	payment := &model.Payment{
		OrderID:         &order.ID,
		UserID:          userID,
		Provider:        "SEPAY",
		Amount:          item.Price,
		Status:          model.PaymentStatusPending,
		QRURL:           qrURL,
		TransferContent: transferContent,
		PaymentType:     model.PaymentTypeOrder,
		ExpiredAt:       time.Now().Add(15 * time.Minute),
	}
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, nil, "", fmt.Errorf("create payment: %w", err)
	}

	log.Printf("QR payment created: order=%s, payment=%s, content=%s", order.ID, payment.ID, transferContent)
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
