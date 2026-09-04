package model

import (
	"time"

	"github.com/google/uuid"
)

// ProductLink status constants.
const (
	LinkStatusAvailable = "AVAILABLE"
	LinkStatusSold      = "SOLD"
)

// ProductLink represents a deliverable link/account for an item.
type ProductLink struct {
	ID             uuid.UUID  `json:"id"`
	ItemID         uuid.UUID  `json:"item_id"`
	Link           string     `json:"link"`
	Status         string     `json:"status"` // AVAILABLE, SOLD
	AssignedUserID *uuid.UUID `json:"assigned_user_id,omitempty"`
	OrderID        *uuid.UUID `json:"order_id,omitempty"`
	AssignedAt     *time.Time `json:"assigned_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
