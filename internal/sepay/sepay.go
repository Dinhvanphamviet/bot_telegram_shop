package sepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
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
	if strings.ToUpper(bankCode) == "MBB" {
		bankCode = "MBBank"
	}
	return fmt.Sprintf(
		"https://qr.sepay.vn/img?acc=%s&bank=%s&amount=%d&des=%s",
		url.QueryEscape(accountNumber),
		url.QueryEscape(bankCode),
		amount,
		url.QueryEscape(content),
	)
}

// VerifyWebhook verifies incoming SePay webhook request using either:
// 1. Authorization header with "Apikey <key>", "Bearer <key>", or plain key
// 2. X-SePay-Signature header with HMAC-SHA256 signature
func VerifyWebhook(secret, apiKey, payload string, authHeader, xSignature string) bool {
	// If neither secret nor apiKey is configured, skip verification (development mode)
	if secret == "" && apiKey == "" {
		return true
	}

	// 1. Check Authorization header (API Key method - default in SePay)
	if authHeader != "" {
		token := strings.TrimSpace(authHeader)
		if strings.HasPrefix(strings.ToLower(token), "apikey ") {
			token = strings.TrimSpace(token[7:])
		} else if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = strings.TrimSpace(token[7:])
		}

		if secret != "" && token == secret {
			return true
		}
		if apiKey != "" && token == apiKey {
			return true
		}
	}

	// 2. Check X-SePay-Signature header (HMAC-SHA256 method)
	if xSignature != "" && secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(payload))
		expectedMAC := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(expectedMAC), []byte(xSignature)) {
			return true
		}
	}

	// 3. Fallback: If authHeader had the raw HMAC signature directly
	if authHeader != "" && secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(payload))
		expectedMAC := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(expectedMAC), []byte(authHeader)) {
			return true
		}
	}

	return false
}

// VerifyWebhookSignature verifies SePay webhook (backwards compatibility).
func VerifyWebhookSignature(secret, payload, signature string) bool {
	return VerifyWebhook(secret, "", payload, signature, signature)
}

// GenerateTransferContent creates a unique transfer content string for an order/payment.
// Format: SHOP{short_id} — max ~20 chars to fit in bank transfer content.
func GenerateTransferContent(shortID string) string {
	return fmt.Sprintf("SHOP%s", shortID)
}
