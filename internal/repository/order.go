package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"telegram-shop/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderRepo handles order database operations.
type OrderRepo struct {
	db *pgxpool.Pool
}

// NewOrderRepo creates a new OrderRepo.
func NewOrderRepo(db *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{db: db}
}

// Create inserts a new order. Can be called within a transaction.
func (r *OrderRepo) Create(ctx context.Context, tx pgx.Tx, o *model.Order) error {
	o.ID = uuid.New()
	if o.Quantity <= 0 {
		o.Quantity = 1
	}
	return tx.QueryRow(ctx,
		`INSERT INTO orders (id, user_id, item_id, amount, quantity, status, payment_method)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		o.ID, o.UserID, o.ItemID, o.Amount, o.Quantity, o.Status, o.PaymentMethod,
	).Scan(&o.CreatedAt)
}

// CreateNoTx inserts a new order without a transaction.
func (r *OrderRepo) CreateNoTx(ctx context.Context, o *model.Order) error {
	o.ID = uuid.New()
	if o.Quantity <= 0 {
		o.Quantity = 1
	}
	return r.db.QueryRow(ctx,
		`INSERT INTO orders (id, user_id, item_id, amount, quantity, status, payment_method)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at`,
		o.ID, o.UserID, o.ItemID, o.Amount, o.Quantity, o.Status, o.PaymentMethod,
	).Scan(&o.CreatedAt)
}

// FindByID returns an order by its UUID.
func (r *OrderRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	var o model.Order
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, item_id, product_link_id, amount, quantity, status, payment_method, created_at, completed_at
		 FROM orders WHERE id = $1`, id,
	).Scan(&o.ID, &o.UserID, &o.ItemID, &o.ProductLinkID, &o.Amount, &o.Quantity, &o.Status, &o.PaymentMethod, &o.CreatedAt, &o.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// FindByUserID returns orders for a user (newest first).
func (r *OrderRepo) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.OrderDetail, error) {
	rows, err := r.db.Query(ctx,
		`SELECT o.id, o.user_id, o.item_id, o.product_link_id, o.amount, o.quantity, o.status, o.payment_method,
		        o.created_at, o.completed_at,
		        i.name AS item_name, p.name AS product_name,
		        COALESCE((SELECT string_agg(pl.link, E'\n') FROM product_links pl WHERE pl.order_id = o.id), COALESCE(pl.link, '')) AS link
		 FROM orders o
		 JOIN items i ON o.item_id = i.id
		 JOIN products p ON i.product_id = p.id
		 LEFT JOIN product_links pl ON o.product_link_id = pl.id
		 WHERE o.user_id = $1
		 ORDER BY o.created_at DESC
		 LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.OrderDetail
	for rows.Next() {
		var od model.OrderDetail
		if err := rows.Scan(
			&od.ID, &od.UserID, &od.ItemID, &od.ProductLinkID, &od.Amount, &od.Quantity, &od.Status, &od.PaymentMethod,
			&od.CreatedAt, &od.CompletedAt,
			&od.ItemName, &od.ProductName, &od.Link,
		); err != nil {
			return nil, err
		}
		if od.Link != "" {
			od.Links = strings.Split(od.Link, "\n")
		}
		orders = append(orders, od)
	}
	return orders, rows.Err()
}

// UpdateStatus updates order status within a transaction.
func (r *OrderRepo) UpdateStatus(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, status string) error {
	var completedAt *time.Time
	if status == model.OrderStatusPaid || status == model.OrderStatusCancelled {
		now := time.Now()
		completedAt = &now
	}
	_, err := tx.Exec(ctx,
		`UPDATE orders SET status = $1, completed_at = $2 WHERE id = $3`,
		status, completedAt, orderID)
	return err
}

// SetProductLink sets the product_link_id on an order within a transaction.
func (r *OrderRepo) SetProductLink(ctx context.Context, tx pgx.Tx, orderID, linkID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE orders SET product_link_id = $1 WHERE id = $2`,
		linkID, orderID)
	return err
}

// FindAll returns all orders (admin).
func (r *OrderRepo) FindAll(ctx context.Context, limit, offset int) ([]model.OrderDetail, error) {
	rows, err := r.db.Query(ctx,
		`SELECT o.id, o.user_id, o.item_id, o.product_link_id, o.amount, o.quantity, o.status, o.payment_method,
		        o.created_at, o.completed_at,
		        i.name AS item_name, p.name AS product_name,
		        COALESCE((SELECT string_agg(pl.link, E'\n') FROM product_links pl WHERE pl.order_id = o.id), COALESCE(pl.link, '')) AS link
		 FROM orders o
		 JOIN items i ON o.item_id = i.id
		 JOIN products p ON i.product_id = p.id
		 LEFT JOIN product_links pl ON o.product_link_id = pl.id
		 ORDER BY o.created_at DESC
		 LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.OrderDetail
	for rows.Next() {
		var od model.OrderDetail
		if err := rows.Scan(
			&od.ID, &od.UserID, &od.ItemID, &od.ProductLinkID, &od.Amount, &od.Quantity, &od.Status, &od.PaymentMethod,
			&od.CreatedAt, &od.CompletedAt,
			&od.ItemName, &od.ProductName, &od.Link,
		); err != nil {
			return nil, err
		}
		if od.Link != "" {
			od.Links = strings.Split(od.Link, "\n")
		}
		orders = append(orders, od)
	}
	return orders, rows.Err()
}

