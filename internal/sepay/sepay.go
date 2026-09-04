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

// VerifyHMACSignature verifies SePay HMAC-SHA256 signature according to official spec:
// - Header X-SePay-Timestamp: Unix timestamp
// - Header X-SePay-Signature: sha256={hex_hash} (or raw hex hash)
// - Signed payload is: "{timestamp}.{raw_body}"
func VerifyHMACSignature(secret, rawBody, timestamp, signature string) bool {
	if secret == "" {
		return true
	}
	if signature == "" {
		return false
	}

	// Prepare data to sign: "{timestamp}.{raw_body}"
	var dataToSign string
	if timestamp != "" {
		dataToSign = fmt.Sprintf("%s.%s", timestamp, rawBody)
	} else {
		dataToSign = rawBody
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(dataToSign))
	expectedHex := hex.EncodeToString(mac.Sum(nil))

	cleanSig := strings.TrimSpace(signature)
	if strings.HasPrefix(strings.ToLower(cleanSig), "sha256=") {
		cleanSig = cleanSig[7:]
	}

	return hmac.Equal([]byte(expectedHex), []byte(cleanSig))
}

// VerifyWebhook verifies incoming SePay webhook request using either:
// 1. HMAC-SHA256 (SePay recommended): X-SePay-Signature + X-SePay-Timestamp
// 2. Authorization header: "Apikey <key>" or "Bearer <key>"
func VerifyWebhook(secret, apiKey, payload string, authHeader, xSignature, xTimestamp string) bool {
	// If neither secret nor apiKey is configured, skip verification (development mode)
	if secret == "" && apiKey == "" {
		return true
	}

	// 1. Check HMAC-SHA256 (SePay primary method as configured in dashboard)
	if xSignature != "" && secret != "" {
		if VerifyHMACSignature(secret, payload, xTimestamp, xSignature) {
			return true
		}
	}

	// 2. Check Authorization header (API Key method)
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

	// 3. Fallback: If no auth headers sent (e.g. "No auth" selected)
	if authHeader == "" && xSignature == "" {
		return true
	}

	return false
}

// VerifyWebhookSignature verifies SePay webhook (backwards compatibility).
func VerifyWebhookSignature(secret, payload, signature string) bool {
	return VerifyWebhook(secret, "", payload, signature, signature, "")
}

// GenerateTransferContent creates a unique transfer content string for an order/payment.
// Format: SHOP{short_id} — max ~20 chars to fit in bank transfer content.
func GenerateTransferContent(shortID string) string {
	return fmt.Sprintf("SHOP%s", shortID)
}
