package repository

import (
	"context"
	"errors"

	"telegram-shop/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ItemRepo handles item database operations.
type ItemRepo struct {
	db *pgxpool.Pool
}

// NewItemRepo creates a new ItemRepo.
func NewItemRepo(db *pgxpool.Pool) *ItemRepo {
	return &ItemRepo{db: db}
}

// FindByProductID returns all active items for a product with available link count.
func (r *ItemRepo) FindByProductID(ctx context.Context, productID uuid.UUID) ([]model.ItemWithStock, error) {
	rows, err := r.db.Query(ctx,
		`SELECT i.id, i.product_id, i.name, i.description, i.price, i.is_active, i.sort_order,
		        i.created_at, i.updated_at,
		        COALESCE(
		            (SELECT COUNT(*) FROM product_links pl WHERE pl.item_id = i.id AND pl.status = 'AVAILABLE'),
		            0
		        ) AS available_count
		 FROM items i
		 WHERE i.product_id = $1 AND i.is_active = true
		 ORDER BY i.sort_order ASC, i.price ASC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ItemWithStock
	for rows.Next() {
		var it model.ItemWithStock
		if err := rows.Scan(
			&it.ID, &it.ProductID, &it.Name, &it.Description, &it.Price,
			&it.IsActive, &it.SortOrder, &it.CreatedAt, &it.UpdatedAt,
			&it.AvailableCount,
		); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// FindByID returns an item by its UUID.
func (r *ItemRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Item, error) {
	var it model.Item
	err := r.db.QueryRow(ctx,
		`SELECT id, product_id, name, description, price, is_active, sort_order, created_at, updated_at
		 FROM items WHERE id = $1`, id,
	).Scan(&it.ID, &it.ProductID, &it.Name, &it.Description, &it.Price, &it.IsActive, &it.SortOrder, &it.CreatedAt, &it.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// CountAvailableLinks returns the number of available links for an item.
func (r *ItemRepo) CountAvailableLinks(ctx context.Context, itemID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM product_links WHERE item_id = $1 AND status = 'AVAILABLE'`,
		itemID,
	).Scan(&count)
	return count, err
}

// Create inserts a new item.
func (r *ItemRepo) Create(ctx context.Context, it *model.Item) error {
	it.ID = uuid.New()
	return r.db.QueryRow(ctx,
		`INSERT INTO items (id, product_id, name, description, price, is_active, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at, updated_at`,
		it.ID, it.ProductID, it.Name, it.Description, it.Price, it.IsActive, it.SortOrder,
	).Scan(&it.CreatedAt, &it.UpdatedAt)
}

// Update updates an item.
func (r *ItemRepo) Update(ctx context.Context, it *model.Item) error {
	_, err := r.db.Exec(ctx,
		`UPDATE items SET name=$1, description=$2, price=$3, is_active=$4, sort_order=$5
		 WHERE id=$6`,
		it.Name, it.Description, it.Price, it.IsActive, it.SortOrder, it.ID)
	return err
}

// Delete removes an item by ID.
func (r *ItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	return err
}
