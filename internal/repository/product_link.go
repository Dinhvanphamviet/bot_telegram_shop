package repository

import (
	"context"
	"errors"
	"time"

	"telegram-shop/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductLinkRepo handles product link database operations.
type ProductLinkRepo struct {
	db *pgxpool.Pool
}

// NewProductLinkRepo creates a new ProductLinkRepo.
func NewProductLinkRepo(db *pgxpool.Pool) *ProductLinkRepo {
	return &ProductLinkRepo{db: db}
}

// ClaimLink claims an available link for an item using row-level locking.
// Uses FOR UPDATE SKIP LOCKED to prevent duplicate selling.
// Must be called within a transaction.
func (r *ProductLinkRepo) ClaimLink(ctx context.Context, tx pgx.Tx, itemID uuid.UUID) (*model.ProductLink, error) {
	var pl model.ProductLink
	err := tx.QueryRow(ctx,
		`SELECT id, item_id, link, status, created_at
		 FROM product_links
		 WHERE item_id = $1 AND status = 'AVAILABLE'
		 ORDER BY created_at ASC
		 LIMIT 1
		 FOR UPDATE SKIP LOCKED`, itemID,
	).Scan(&pl.ID, &pl.ItemID, &pl.Link, &pl.Status, &pl.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // no available link
	}
	if err != nil {
		return nil, err
	}
	return &pl, nil
}

// AssignLink marks a link as SOLD and assigns it to a user/order.
// Must be called within a transaction.
func (r *ProductLinkRepo) AssignLink(ctx context.Context, tx pgx.Tx, linkID, userID, orderID uuid.UUID) error {
	now := time.Now()
	_, err := tx.Exec(ctx,
		`UPDATE product_links
		 SET status = 'SOLD', assigned_user_id = $1, order_id = $2, assigned_at = $3
		 WHERE id = $4`,
		userID, orderID, now, linkID)
	return err
}

// FindByOrderID returns the link assigned to an order.
func (r *ProductLinkRepo) FindByOrderID(ctx context.Context, orderID uuid.UUID) (*model.ProductLink, error) {
	var pl model.ProductLink
	err := r.db.QueryRow(ctx,
		`SELECT id, item_id, link, status, assigned_user_id, order_id, assigned_at, created_at
		 FROM product_links WHERE order_id = $1`, orderID,
	).Scan(&pl.ID, &pl.ItemID, &pl.Link, &pl.Status, &pl.AssignedUserID, &pl.OrderID, &pl.AssignedAt, &pl.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pl, nil
}

// Create inserts a new product link.
func (r *ProductLinkRepo) Create(ctx context.Context, pl *model.ProductLink) error {
	pl.ID = uuid.New()
	pl.Status = model.LinkStatusAvailable
	return r.db.QueryRow(ctx,
		`INSERT INTO product_links (id, item_id, link, status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at`,
		pl.ID, pl.ItemID, pl.Link, pl.Status,
	).Scan(&pl.CreatedAt)
}

// CreateBatch inserts multiple links for an item.
func (r *ProductLinkRepo) CreateBatch(ctx context.Context, itemID uuid.UUID, links []string) (int, error) {
	batch := &pgx.Batch{}
	for _, link := range links {
		id := uuid.New()
		batch.Queue(
			`INSERT INTO product_links (id, item_id, link, status) VALUES ($1, $2, $3, 'AVAILABLE')`,
			id, itemID, link,
		)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	count := 0
	for range links {
		_, err := br.Exec()
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// CountByItemID returns available and total link counts for an item.
func (r *ProductLinkRepo) CountByItemID(ctx context.Context, itemID uuid.UUID) (available, total int, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT
		    COUNT(*) FILTER (WHERE status = 'AVAILABLE') AS available,
		    COUNT(*) AS total
		 FROM product_links WHERE item_id = $1`, itemID,
	).Scan(&available, &total)
	return
}

// FindByItemID returns all links for an item (admin).
func (r *ProductLinkRepo) FindByItemID(ctx context.Context, itemID uuid.UUID) ([]model.ProductLink, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, item_id, link, status, assigned_user_id, order_id, assigned_at, created_at
		 FROM product_links WHERE item_id = $1
		 ORDER BY created_at ASC`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []model.ProductLink
	for rows.Next() {
		var pl model.ProductLink
		if err := rows.Scan(&pl.ID, &pl.ItemID, &pl.Link, &pl.Status, &pl.AssignedUserID, &pl.OrderID, &pl.AssignedAt, &pl.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, pl)
	}
	return links, rows.Err()
}

// Delete removes a product link by ID (only if AVAILABLE).
func (r *ProductLinkRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM product_links WHERE id = $1 AND status = 'AVAILABLE'`, id)
	return err
}
