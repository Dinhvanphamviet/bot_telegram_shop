package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the application.
type Config struct {
	DatabaseURL      string
	TelegramBotToken string
	WebhookURL       string
	Port             string
	AdminTelegramIDs []int64
	AdminAPIKey      string

	// SePay Payment
	SepayAPIKey        string
	SepayBankCode      string
	SepayAccountNumber string
	SepayWebhookSecret string
}

// Load reads configuration from environment variables and validates required fields.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		WebhookURL:         os.Getenv("WEBHOOK_URL"),
		Port:               os.Getenv("PORT"),
		AdminAPIKey:        os.Getenv("ADMIN_API_KEY"),
		SepayAPIKey:        os.Getenv("SEPAY_API_KEY"),
		SepayBankCode:      os.Getenv("SEPAY_BANK_CODE"),
		SepayAccountNumber: os.Getenv("SEPAY_ACCOUNT_NUMBER"),
		SepayWebhookSecret: os.Getenv("SEPAY_WEBHOOK_SECRET"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Parse admin Telegram IDs
	adminIDsStr := os.Getenv("ADMIN_TELEGRAM_IDS")
	if adminIDsStr != "" {
		parts := strings.Split(adminIDsStr, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid admin telegram id %q: %w", p, err)
			}
			cfg.AdminTelegramIDs = append(cfg.AdminTelegramIDs, id)
		}
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("WEBHOOK_URL is required")
	}

	return cfg, nil
}

// IsAdmin checks if a given Telegram ID is an admin.
func (c *Config) IsAdmin(telegramID int64) bool {
	for _, id := range c.AdminTelegramIDs {
		if id == telegramID {
			return true
		}
	}
	return false
}
