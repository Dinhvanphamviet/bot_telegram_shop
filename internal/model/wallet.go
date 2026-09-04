package model

import (
	"time"

	"github.com/google/uuid"
)

// Wallet transaction type constants.
const (
	WalletTypeDeposit  = "DEPOSIT"
	WalletTypePurchase = "PURCHASE"
	WalletTypeRefund   = "REFUND"
)

// WalletTransaction represents a wallet balance change.
type WalletTransaction struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Amount      int64     `json:"amount"` // positive = tiền vào, negative = tiền ra
	Type        string    `json:"type"`   // DEPOSIT, PURCHASE, REFUND
	Status      string    `json:"status"`
	Description string    `json:"description"`
	ReferenceID string    `json:"reference_id"`
	CreatedAt   time.Time `json:"created_at"`
}
