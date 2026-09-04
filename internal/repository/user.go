package repository

import (
	"context"
	"errors"

	"telegram-shop/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo handles user database operations.
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

// FindByTelegramID finds a user by their Telegram ID.
func (r *UserRepo) FindByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx,
		`SELECT id, telegram_id, username, first_name, balance, created_at, updated_at
		 FROM users WHERE telegram_id = $1`, telegramID,
	).Scan(&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.Balance, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Create inserts a new user.
func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	u.ID = uuid.New()
	return r.db.QueryRow(ctx,
		`INSERT INTO users (id, telegram_id, username, first_name, balance)
		 VALUES ($1, $2, $3, $4, 0)
		 RETURNING created_at, updated_at`,
		u.ID, u.TelegramID, u.Username, u.FirstName,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

// FindByID finds a user by UUID.
func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx,
		`SELECT id, telegram_id, username, first_name, balance, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.TelegramID, &u.Username, &u.FirstName, &u.Balance, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateBalance atomically updates user balance within a transaction.
// delta can be positive (deposit) or negative (purchase).
func (r *UserRepo) UpdateBalance(ctx context.Context, tx pgx.Tx, userID uuid.UUID, delta int64) (int64, error) {
	var newBalance int64
	err := tx.QueryRow(ctx,
		`UPDATE users SET balance = balance + $1
		 WHERE id = $2 AND balance + $1 >= 0
		 RETURNING balance`,
		delta, userID,
	).Scan(&newBalance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errors.New("insufficient balance")
	}
	return newBalance, err
}

// GetBalance returns the current balance for a user.
func (r *UserRepo) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	var balance int64
	err := r.db.QueryRow(ctx, `SELECT balance FROM users WHERE id = $1`, userID).Scan(&balance)
	return balance, err
}
