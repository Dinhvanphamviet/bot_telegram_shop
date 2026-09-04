package service

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"telegram-shop/internal/model"
	"telegram-shop/internal/repository"
	"telegram-shop/internal/sepay"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentService handles SePay webhook processing and payment lifecycle.
type PaymentService struct {
	db              *pgxpool.Pool
	paymentRepo     *repository.PaymentRepo
	orderRepo       *repository.OrderRepo
	userRepo        *repository.UserRepo
	productLinkRepo *repository.ProductLinkRepo
	walletRepo      *repository.WalletRepo
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(
	db *pgxpool.Pool,
	paymentRepo *repository.PaymentRepo,
	orderRepo *repository.OrderRepo,
	userRepo *repository.UserRepo,
	productLinkRepo *repository.ProductLinkRepo,
	walletRepo *repository.WalletRepo,
) *PaymentService {
	return &PaymentService{
		db:              db,
		paymentRepo:     paymentRepo,
		orderRepo:       orderRepo,
		userRepo:        userRepo,
		productLinkRepo: productLinkRepo,
		walletRepo:      walletRepo,
	}
}

// PaymentResult contains the result of processing a webhook for notification purposes.
type PaymentResult struct {
	Payment     *model.Payment
	Order       *model.Order
	Link        *model.ProductLink
	Links       []model.ProductLink
	UserID      uuid.UUID
	Success     bool
	NeedsRefund bool
	Message     string
}

// HandleSepayWebhook processes an incoming SePay webhook payment notification.
// This is the core payment processing logic with full idempotency.
func (s *PaymentService) HandleSepayWebhook(ctx context.Context, payload *sepay.WebhookPayload) (*PaymentResult, error) {
	// Only process incoming transfers
	if !strings.EqualFold(strings.TrimSpace(payload.TransferType), "in") {
		log.Printf("[SePay] Ignoring non-incoming transfer (type=%s): %s", payload.TransferType, payload.Content)
		return nil, nil
	}

	// Extract transfer content — search across Content, Description, and Code
	content := extractTransferContent(payload.Content, payload.Description, payload.Code)
	if content == "" {
		log.Printf("[SePay] No matching transfer content found in content=%q, desc=%q, code=%q",
			payload.Content, payload.Description, payload.Code)
		return nil, nil
	}

	// Find payment by transfer content
	payment, err := s.paymentRepo.FindByTransferContent(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("find payment: %w", err)
	}
	if payment == nil {
		log.Printf("[SePay] No payment found for transfer content: %s (content=%q, desc=%q, code=%q)",
			content, payload.Content, payload.Description, payload.Code)
		return nil, nil
	}

	// IDEMPOTENCY: if already processed, skip
	if payment.Status == model.PaymentStatusSuccess {
		log.Printf("Payment already processed (idempotent): %s", payment.ID)
		return &PaymentResult{
			Payment: payment,
			UserID:  payment.UserID,
			Success: true,
			Message: "Đã xử lý trước đó",
		}, nil
	}

	// Check payment is still pending
	if payment.Status != model.PaymentStatusPending {
		log.Printf("Payment not pending (status=%s): %s", payment.Status, payment.ID)
		return nil, nil
	}

	// Verify amount
	if payload.TransferAmount < payment.Amount {
		log.Printf("Amount mismatch: expected=%d, got=%d, payment=%s",
			payment.Amount, payload.TransferAmount, payment.ID)
		return nil, fmt.Errorf("số tiền không khớp: cần %d, nhận %d", payment.Amount, payload.TransferAmount)
	}

	// Process based on payment type
	switch payment.PaymentType {
	case model.PaymentTypeOrder:
		return s.processOrderPayment(ctx, payment, payload.ReferenceCode)
	case model.PaymentTypeDeposit:
		return s.processDepositPayment(ctx, payment, payload.ReferenceCode)
	default:
		return nil, fmt.Errorf("unknown payment type: %s", payment.PaymentType)
	}
}

// processOrderPayment handles a successful payment for a product order.
func (s *PaymentService) processOrderPayment(ctx context.Context, payment *model.Payment, providerTxID string) (*PaymentResult, error) {
	if payment.OrderID == nil {
		return nil, fmt.Errorf("order payment has no order_id: %s", payment.ID)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get the order
	order, err := s.orderRepo.FindByID(ctx, *payment.OrderID)
	if err != nil || order == nil {
		return nil, fmt.Errorf("find order: %w", err)
	}

	// Check order not already paid
	if order.Status == model.OrderStatusPaid {
		return &PaymentResult{Payment: payment, Order: order, UserID: payment.UserID, Success: true, Message: "Đã xử lý"}, nil
	}

	if err := s.paymentRepo.UpdateStatusTx(ctx, tx, payment.ID, model.PaymentStatusSuccess, providerTxID); err != nil {
		return nil, fmt.Errorf("update payment: %w", err)
	}

	qty := order.Quantity
	if qty <= 0 {
		qty = 1
	}

	links, err := s.productLinkRepo.ClaimLinks(ctx, tx, order.ItemID, qty)
	if err != nil {
		return nil, fmt.Errorf("claim links: %w", err)
	}

	if len(links) < qty {
		// Out of stock refund to wallet
		log.Printf("Out of stock for item %s (requested %d, claimed %d), refunding order %s", order.ItemID, qty, len(links), order.ID)

		if err := s.orderRepo.UpdateStatus(ctx, tx, order.ID, model.OrderStatusCancelled); err != nil {
			return nil, fmt.Errorf("cancel order: %w", err)
		}

		if _, err := s.userRepo.UpdateBalance(ctx, tx, order.UserID, order.Amount); err != nil {
			return nil, fmt.Errorf("refund balance: %w", err)
		}

		wt := &model.WalletTransaction{
			UserID:      order.UserID,
			Amount:      order.Amount,
			Type:        model.WalletTypeRefund,
			Status:      "SUCCESS",
			Description: fmt.Sprintf("Hoàn tiền đơn %s (hết hàng)", order.ID.String()[:8]),
			ReferenceID: order.ID.String(),
		}
		if err := s.walletRepo.Create(ctx, tx, wt); err != nil {
			return nil, fmt.Errorf("create refund tx: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit refund: %w", err)
		}

		return &PaymentResult{
			Payment:     payment,
			Order:       order,
			UserID:      payment.UserID,
			Success:     false,
			NeedsRefund: true,
			Message:     fmt.Sprintf("Sản phẩm không đủ hàng trong kho. Đã hoàn %s vào ví của bạn.", formatMoney(order.Amount)),
		}, nil
	}

	linkIDs := make([]uuid.UUID, len(links))
	for i, l := range links {
		linkIDs[i] = l.ID
	}

	if err := s.productLinkRepo.AssignLinks(ctx, tx, linkIDs, order.UserID, order.ID); err != nil {
		return nil, fmt.Errorf("assign links: %w", err)
	}

	if err := s.orderRepo.SetProductLink(ctx, tx, order.ID, links[0].ID); err != nil {
		return nil, fmt.Errorf("set link on order: %w", err)
	}
	if err := s.orderRepo.UpdateStatus(ctx, tx, order.ID, model.OrderStatusPaid); err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	log.Printf("Order payment completed: payment=%s, order=%s, quantity=%d", payment.ID, order.ID, qty)
	return &PaymentResult{
		Payment: payment,
		Order:   order,
		Link:    &links[0],
		Links:   links,
		UserID:  payment.UserID,
		Success: true,
		Message: "Thanh toán thành công!",
	}, nil
}

// processDepositPayment handles a successful wallet deposit.
func (s *PaymentService) processDepositPayment(ctx context.Context, payment *model.Payment, providerTxID string) (*PaymentResult, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.paymentRepo.UpdateStatusTx(ctx, tx, payment.ID, model.PaymentStatusSuccess, providerTxID); err != nil {
		return nil, fmt.Errorf("update payment: %w", err)
	}

	newBalance, err := s.userRepo.UpdateBalance(ctx, tx, payment.UserID, payment.Amount)
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	wt := &model.WalletTransaction{
		UserID:      payment.UserID,
		Amount:      payment.Amount,
		Type:        model.WalletTypeDeposit,
		Status:      "SUCCESS",
		Description: fmt.Sprintf("Nạp tiền qua QR (%s)", payment.TransferContent),
		ReferenceID: payment.ID.String(),
	}
	if err := s.walletRepo.Create(ctx, tx, wt); err != nil {
		return nil, fmt.Errorf("create wallet tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	log.Printf("Deposit completed: payment=%s, user=%s, amount=%d, new_balance=%d",
		payment.ID, payment.UserID, payment.Amount, newBalance)
	return &PaymentResult{
		Payment: payment,
		UserID:  payment.UserID,
		Success: true,
		Message: fmt.Sprintf("Nạp tiền thành công! Số dư: %s", formatMoney(newBalance)),
	}, nil
}

// ExpiredPaymentNotification holds information to notify a user about an expired payment.
type ExpiredPaymentNotification struct {
	PaymentID       uuid.UUID
	TelegramID      int64
	ChatID          int64
	MessageID       int64
	TransferContent string
	Amount          int64
	PaymentType     string
}

// SetMessageInfo records Telegram chat_id and message_id for the QR message.
func (s *PaymentService) SetMessageInfo(ctx context.Context, paymentID uuid.UUID, chatID, messageID int64) error {
	return s.paymentRepo.SetMessageInfo(ctx, paymentID, chatID, messageID)
}

// CancelPayment cancels a pending payment and its associated order (if any).
func (s *PaymentService) CancelPayment(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.paymentRepo.FindByID(ctx, paymentID)
	if err != nil || payment == nil {
		return fmt.Errorf("payment not found")
	}
	if payment.Status != model.PaymentStatusPending {
		return fmt.Errorf("payment is not pending")
	}

	if err := s.paymentRepo.CancelPayment(ctx, paymentID); err != nil {
		return fmt.Errorf("cancel payment: %w", err)
	}

	if payment.OrderID != nil && payment.PaymentType == model.PaymentTypeOrder {
		tx, err := s.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer tx.Rollback(ctx)

		if err := s.orderRepo.UpdateStatus(ctx, tx, *payment.OrderID, model.OrderStatusCancelled); err != nil {
			return fmt.Errorf("cancel order: %w", err)
		}
		return tx.Commit(ctx)
	}
	return nil
}

// ExpirePendingPayments expires all pending payments past their deadline.
func (s *PaymentService) ExpirePendingPayments(ctx context.Context) ([]ExpiredPaymentNotification, error) {
	expired, err := s.paymentRepo.FindPendingExpired(ctx)
	if err != nil {
		return nil, fmt.Errorf("find expired: %w", err)
	}

	var notifications []ExpiredPaymentNotification

	for _, p := range expired {
		if err := s.paymentRepo.ExpirePayment(ctx, p.ID); err != nil {
			log.Printf("Error expiring payment %s: %v", p.ID, err)
			continue
		}

		if p.OrderID != nil && p.PaymentType == model.PaymentTypeOrder {
			tx, err := s.db.Begin(ctx)
			if err != nil {
				log.Printf("Error beginning tx for order expiry: %v", err)
				continue
			}
			if err := s.orderRepo.UpdateStatus(ctx, tx, *p.OrderID, model.OrderStatusExpired); err != nil {
				tx.Rollback(ctx)
				log.Printf("Error expiring order %s: %v", p.OrderID, err)
				continue
			}
			tx.Commit(ctx)
		}

		var cID, mID int64
		if p.ChatID != nil {
			cID = *p.ChatID
		}
		if p.MessageID != nil {
			mID = *p.MessageID
		}

		user, err := s.userRepo.FindByID(ctx, p.UserID)
		if err == nil && user != nil && user.TelegramID > 0 {
			notifications = append(notifications, ExpiredPaymentNotification{
				PaymentID:       p.ID,
				TelegramID:      user.TelegramID,
				ChatID:          cID,
				MessageID:       mID,
				TransferContent: p.TransferContent,
				Amount:          p.Amount,
				PaymentType:     p.PaymentType,
			})
		}

		log.Printf("Expired payment: %s (%s)", p.ID, p.TransferContent)
	}

	if len(expired) > 0 {
		log.Printf("Expired %d pending payments", len(expired))
	}
	return notifications, nil
}

var transferContentRegex = regexp.MustCompile(`(?i)SHOP[\s\-_:]*([A-Za-z0-9]{8})`)

// extractTransferContent extracts our transfer content code from the bank transfer description.
// SePay sends the full bank description which may contain extra text.
// We look for our "SHOP" prefix pattern across content, description, or code.
func extractTransferContent(texts ...string) string {
	for _, text := range texts {
		if text == "" {
			continue
		}
		matches := transferContentRegex.FindStringSubmatch(text)
		if len(matches) > 1 {
			return "SHOP" + strings.ToUpper(matches[1])
		}
	}
	return ""
}

// formatMoney formats amount in VND (e.g. 50000 → "50.000đ").
func formatMoney(amount int64) string {
	if amount < 0 {
		return "-" + formatMoney(-amount)
	}
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return s + "đ"
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result) + "đ"
}
