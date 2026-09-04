package model

import (
	"time"

	"github.com/google/uuid"
)

// Order status constants.
const (
	OrderStatusPending   = "PENDING"
	OrderStatusPaid      = "PAID"
	OrderStatusCancelled = "CANCELLED"
	OrderStatusExpired   = "EXPIRED"
)

// Payment method constants.
const (
	PaymentMethodQR     = "QR"
	PaymentMethodWallet = "WALLET"
)

// Order represents a user's purchase order.
type Order struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	ItemID        uuid.UUID  `json:"item_id"`
	ProductLinkID *uuid.UUID `json:"product_link_id,omitempty"`
	Amount        int64      `json:"amount"`
	Quantity      int        `json:"quantity"`
	Status        string     `json:"status"`         // PENDING, PAID, CANCELLED, EXPIRED
	PaymentMethod string     `json:"payment_method"` // QR, WALLET
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// OrderDetail includes related info for display.
type OrderDetail struct {
	Order
	ItemName    string   `json:"item_name"`
	ProductName string   `json:"product_name"`
	Link        string   `json:"link,omitempty"`
	Links       []string `json:"links,omitempty"`
}
