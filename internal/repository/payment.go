package repository

import (
	"context"
	"errors"

	"telegram-shop/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentRepo handles payment database operations.
type PaymentRepo struct {
	db *pgxpool.Pool
}

// NewPaymentRepo creates a new PaymentRepo.
func NewPaymentRepo(db *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{db: db}
}

// Create inserts a new payment.
func (r *PaymentRepo) Create(ctx context.Context, p *model.Payment) error {
	p.ID = uuid.New()
	return r.db.QueryRow(ctx,
		`INSERT INTO payments (id, order_id, user_id, provider, amount, status, qr_url, transfer_content, payment_type, expired_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING created_at, updated_at`,
		p.ID, p.OrderID, p.UserID, p.Provider, p.Amount, p.Status,
		p.QRURL, p.TransferContent, p.PaymentType, p.ExpiredAt,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
}

// FindByTransferContent finds a payment by its unique transfer content (for webhook matching).
func (r *PaymentRepo) FindByTransferContent(ctx context.Context, content string) (*model.Payment, error) {
	var p model.Payment
	err := r.db.QueryRow(ctx,
		`SELECT id, order_id, user_id, provider, provider_transaction_id, amount, status,
		        qr_url, transfer_content, payment_type, expired_at, paid_at, created_at, updated_at
		 FROM payments WHERE transfer_content = $1`, content,
	).Scan(&p.ID, &p.OrderID, &p.UserID, &p.Provider, &p.ProviderTransactionID, &p.Amount, &p.Status,
		&p.QRURL, &p.TransferContent, &p.PaymentType, &p.ExpiredAt, &p.PaidAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// FindByOrderID finds a payment by order ID.
func (r *PaymentRepo) FindByOrderID(ctx context.Context, orderID uuid.UUID) (*model.Payment, error) {
	var p model.Payment
	err := r.db.QueryRow(ctx,
		`SELECT id, order_id, user_id, provider, provider_transaction_id, amount, status,
		        qr_url, transfer_content, payment_type, expired_at, paid_at, created_at, updated_at
		 FROM payments WHERE order_id = $1`, orderID,
	).Scan(&p.ID, &p.OrderID, &p.UserID, &p.Provider, &p.ProviderTransactionID, &p.Amount, &p.Status,
		&p.QRURL, &p.TransferContent, &p.PaymentType, &p.ExpiredAt, &p.PaidAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateStatusTx updates payment status within a transaction.
func (r *PaymentRepo) UpdateStatusTx(ctx context.Context, tx pgx.Tx, paymentID uuid.UUID, status, providerTxID string) error {
	if status == model.PaymentStatusSuccess {
		_, err := tx.Exec(ctx,
			`UPDATE payments SET status = $1, provider_transaction_id = $2, paid_at = NOW() WHERE id = $3`,
			status, providerTxID, paymentID)
		return err
	}
	_, err := tx.Exec(ctx,
		`UPDATE payments SET status = $1, provider_transaction_id = $2 WHERE id = $3`,
		status, providerTxID, paymentID)
	return err
}

// FindPendingExpired returns pending payments that have expired.
func (r *PaymentRepo) FindPendingExpired(ctx context.Context) ([]model.Payment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, order_id, user_id, provider, provider_transaction_id, amount, status,
		        qr_url, transfer_content, payment_type, expired_at, paid_at, created_at, updated_at
		 FROM payments
		 WHERE status = 'PENDING' AND expired_at < NOW()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []model.Payment
	for rows.Next() {
		var p model.Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.UserID, &p.Provider, &p.ProviderTransactionID,
			&p.Amount, &p.Status, &p.QRURL, &p.TransferContent, &p.PaymentType,
			&p.ExpiredAt, &p.PaidAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

// ExpirePayment marks a payment as EXPIRED (no transaction needed).
func (r *PaymentRepo) ExpirePayment(ctx context.Context, paymentID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE payments SET status = 'EXPIRED' WHERE id = $1 AND status = 'PENDING'`, paymentID)
	return err
}
