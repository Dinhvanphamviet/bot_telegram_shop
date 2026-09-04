package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a Telegram user in the system.
type User struct {
	ID         uuid.UUID `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	FirstName  string    `json:"first_name"`
	Balance    int64     `json:"balance"` // đơn vị đồng (VND)
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
