package repository

import (
	"context"
	"errors"

	"telegram-shop/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductRepo handles product database operations.
type ProductRepo struct {
	db *pgxpool.Pool
}

// NewProductRepo creates a new ProductRepo.
func NewProductRepo(db *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{db: db}
}

// FindAllActive returns all active products sorted by sort_order.
func (r *ProductRepo) FindAllActive(ctx context.Context) ([]model.Product, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, image_url, is_active, sort_order, created_at, updated_at
		 FROM products WHERE is_active = true
		 ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.ImageURL, &p.IsActive, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// FindAll returns all products (admin).
func (r *ProductRepo) FindAll(ctx context.Context) ([]model.Product, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, image_url, is_active, sort_order, created_at, updated_at
		 FROM products ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.ImageURL, &p.IsActive, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// FindByID returns a product by its UUID.
func (r *ProductRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Product, error) {
	var p model.Product
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, image_url, is_active, sort_order, created_at, updated_at
		 FROM products WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.ImageURL, &p.IsActive, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Create inserts a new product.
func (r *ProductRepo) Create(ctx context.Context, p *model.Product) error {
	p.ID = uuid.New()
	return r.db.QueryRow(ctx,
		`INSERT INTO products (id, name, description, image_url, is_active, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING created_at, updated_at`,
		p.ID, p.Name, p.Description, p.ImageURL, p.IsActive, p.SortOrder,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
}

// Update updates a product.
func (r *ProductRepo) Update(ctx context.Context, p *model.Product) error {
	_, err := r.db.Exec(ctx,
		`UPDATE products SET name=$1, description=$2, image_url=$3, is_active=$4, sort_order=$5
		 WHERE id=$6`,
		p.Name, p.Description, p.ImageURL, p.IsActive, p.SortOrder, p.ID)
	return err
}

// Delete removes a product by ID.
func (r *ProductRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	return err
}
