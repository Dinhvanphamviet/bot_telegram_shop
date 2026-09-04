package repository

import (
	"context"

	"telegram-shop/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WalletRepo handles wallet transaction database operations.
type WalletRepo struct {
	db *pgxpool.Pool
}

// NewWalletRepo creates a new WalletRepo.
func NewWalletRepo(db *pgxpool.Pool) *WalletRepo {
	return &WalletRepo{db: db}
}

// Create inserts a new wallet transaction. Can be called within a transaction.
func (r *WalletRepo) Create(ctx context.Context, tx pgx.Tx, wt *model.WalletTransaction) error {
	wt.ID = uuid.New()
	return tx.QueryRow(ctx,
		`INSERT INTO wallet_transactions (id, user_id, amount, type, status, description, reference_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		wt.ID, wt.UserID, wt.Amount, wt.Type, wt.Status, wt.Description, wt.ReferenceID,
	).Scan(&wt.CreatedAt)
}

// CreateNoTx inserts a wallet transaction without an explicit transaction.
func (r *WalletRepo) CreateNoTx(ctx context.Context, wt *model.WalletTransaction) error {
	wt.ID = uuid.New()
	return r.db.QueryRow(ctx,
		`INSERT INTO wallet_transactions (id, user_id, amount, type, status, description, reference_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		wt.ID, wt.UserID, wt.Amount, wt.Type, wt.Status, wt.Description, wt.ReferenceID,
	).Scan(&wt.CreatedAt)
}

// FindByUserID returns wallet transactions for a user (newest first).
func (r *WalletRepo) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.WalletTransaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, amount, type, status, description, reference_id, created_at
		 FROM wallet_transactions
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []model.WalletTransaction
	for rows.Next() {
		var wt model.WalletTransaction
		if err := rows.Scan(&wt.ID, &wt.UserID, &wt.Amount, &wt.Type, &wt.Status, &wt.Description, &wt.ReferenceID, &wt.CreatedAt); err != nil {
			return nil, err
		}
		transactions = append(transactions, wt)
	}
	return transactions, rows.Err()
}
