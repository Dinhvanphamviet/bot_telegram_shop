package model

import (
	"time"

	"github.com/google/uuid"
)

// Payment status constants.
const (
	PaymentStatusPending   = "PENDING"
	PaymentStatusSuccess   = "SUCCESS"
	PaymentStatusFailed    = "FAILED"
	PaymentStatusExpired   = "EXPIRED"
	PaymentStatusCancelled = "CANCELLED"
)

// Payment type constants.
const (
	PaymentTypeOrder   = "ORDER"
	PaymentTypeDeposit = "DEPOSIT"
)

// Payment represents a SePay payment record.
type Payment struct {
	ID                    uuid.UUID  `json:"id"`
	OrderID               *uuid.UUID `json:"order_id,omitempty"`
	UserID                uuid.UUID  `json:"user_id"`
	Provider              string     `json:"provider"`
	ProviderTransactionID *string    `json:"provider_transaction_id,omitempty"`
	Amount                int64      `json:"amount"`
	Status                string     `json:"status"` // PENDING, SUCCESS, FAILED, EXPIRED, CANCELLED
	QRURL                 string     `json:"qr_url,omitempty"`
	TransferContent       string     `json:"transfer_content"`
	PaymentType           string     `json:"payment_type"` // ORDER, DEPOSIT
	ExpiredAt             time.Time  `json:"expired_at"`
	PaidAt                *time.Time `json:"paid_at,omitempty"`
	ChatID                *int64     `json:"chat_id,omitempty"`
	MessageID             *int64     `json:"message_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}
