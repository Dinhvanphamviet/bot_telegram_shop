package sepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
)

// WebhookPayload represents the data SePay sends via webhook.
type WebhookPayload struct {
	Gateway         string `json:"gateway"`
	TransactionDate string `json:"transactionDate"`
	AccountNumber   string `json:"accountNumber"`
	Code            string `json:"code"`
	Content         string `json:"content"`
	TransferType    string `json:"transferType"` // "in" = tiền vào, "out" = tiền ra
	TransferAmount  int64  `json:"transferAmount"`
	Accumulated     int64  `json:"accumulated"`
	SubAccount      string `json:"subAccount"`
	ReferenceCode   string `json:"referenceCode"`
	Description     string `json:"description"`
	ID              int64  `json:"id"`
}

// GenerateQRURL creates a VietQR payment URL for SePay.
// The URL returns a QR code image that can be sent directly to users.
func GenerateQRURL(bankCode, accountNumber string, amount int64, content string) string {
	return fmt.Sprintf(
		"https://qr.sepay.vn/img?acc=%s&bank=%s&amount=%d&des=%s",
		url.QueryEscape(accountNumber),
		url.QueryEscape(bankCode),
		amount,
		url.QueryEscape(content),
	)
}

// VerifyWebhookSignature verifies SePay webhook using HMAC-SHA256.
// The signature is sent in the request header.
func VerifyWebhookSignature(secret, payload, signature string) bool {
	if secret == "" {
		// If no secret configured, skip verification (development mode)
		return true
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

// GenerateTransferContent creates a unique transfer content string for an order/payment.
// Format: SHOP{short_id} — max ~20 chars to fit in bank transfer content.
func GenerateTransferContent(shortID string) string {
	return fmt.Sprintf("SHOP%s", shortID)
}
