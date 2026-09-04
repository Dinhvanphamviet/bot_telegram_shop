package model

import (
	"time"

	"github.com/google/uuid"
)

// Item represents a variant/package of a product (e.g. Netflix Premium 1 tháng).
type Item struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       int64     `json:"price"` // đơn vị đồng (VND)
	IsActive    bool      `json:"is_active"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ItemWithStock is an Item with available link count for display.
type ItemWithStock struct {
	Item
	AvailableCount int `json:"available_count"`
}
