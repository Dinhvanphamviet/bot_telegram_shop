package service

import (
	"context"
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

// WalletService handles wallet operations.
type WalletService struct {
	db          *pgxpool.Pool
	cfg         *config.Config
	userRepo    *repository.UserRepo
	walletRepo  *repository.WalletRepo
	paymentRepo *repository.PaymentRepo
}

// NewWalletService creates a new WalletService.
func NewWalletService(
	db *pgxpool.Pool,
	cfg *config.Config,
	userRepo *repository.UserRepo,
	walletRepo *repository.WalletRepo,
	paymentRepo *repository.PaymentRepo,
) *WalletService {
	return &WalletService{
		db:          db,
		cfg:         cfg,
		userRepo:    userRepo,
		walletRepo:  walletRepo,
		paymentRepo: paymentRepo,
	}
}

// GetBalance returns the current balance for a user.
func (s *WalletService) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.userRepo.GetBalance(ctx, userID)
}

// GetTransactions returns recent wallet transactions for a user.
func (s *WalletService) GetTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.WalletTransaction, error) {
	return s.walletRepo.FindByUserID(ctx, userID, limit, offset)
}

// DepositWithQR creates a QR payment for wallet deposit.
func (s *WalletService) DepositWithQR(ctx context.Context, userID uuid.UUID, amount int64) (*model.Payment, string, error) {
	if amount < 10000 {
		return nil, "", fmt.Errorf("số tiền nạp tối thiểu là 10.000đ")
	}

	// Generate unique transfer content for deposit
	depositID := uuid.New()
	shortID := strings.ToUpper(depositID.String()[:8])
	transferContent := sepay.GenerateTransferContent(shortID)

	// Generate QR URL
	qrURL := sepay.GenerateQRURL(s.cfg.SepayBankCode, s.cfg.SepayAccountNumber, amount, transferContent)

	// Create payment (no order for deposits)
	payment := &model.Payment{
		UserID:          userID,
		Provider:        "SEPAY",
		Amount:          amount,
		Status:          model.PaymentStatusPending,
		QRURL:           qrURL,
		TransferContent: transferContent,
		PaymentType:     model.PaymentTypeDeposit,
		ExpiredAt:       time.Now().Add(10 * time.Minute),
	}
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, "", fmt.Errorf("create payment: %w", err)
	}

	log.Printf("Deposit QR created: payment=%s, user=%s, amount=%d", payment.ID, userID, amount)
	return payment, qrURL, nil
}

// AdminTopup adds balance to a user's wallet (admin operation).
func (s *WalletService) AdminTopup(ctx context.Context, userID uuid.UUID, amount int64, adminNote string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	newBalance, err := s.userRepo.UpdateBalance(ctx, tx, userID, amount)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	wt := &model.WalletTransaction{
		UserID:      userID,
		Amount:      amount,
		Type:        model.WalletTypeDeposit,
		Status:      "SUCCESS",
		Description: fmt.Sprintf("Admin nạp tiền: %s", adminNote),
		ReferenceID: "admin",
	}
	if err := s.walletRepo.Create(ctx, tx, wt); err != nil {
		return fmt.Errorf("create wallet tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("Admin topup: user=%s, amount=%d, new_balance=%d", userID, amount, newBalance)
	return nil
}
