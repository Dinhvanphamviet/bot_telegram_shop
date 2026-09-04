package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"telegram-shop/internal/bot"
	"telegram-shop/internal/config"
	"telegram-shop/internal/sepay"
	"telegram-shop/internal/service"
)

// TelegramHandler handles incoming Telegram webhook and SePay webhook requests.
type TelegramHandler struct {
	bot             *bot.Bot
	cfg             *config.Config
	commandHandler  *bot.CommandHandler
	callbackHandler *bot.CallbackHandler
	paymentService  *service.PaymentService
	userService     *service.UserService
}

// NewTelegramHandler creates a new TelegramHandler.
func NewTelegramHandler(
	b *bot.Bot,
	cfg *config.Config,
	commandHandler *bot.CommandHandler,
	callbackHandler *bot.CallbackHandler,
	paymentService *service.PaymentService,
	userService *service.UserService,
) *TelegramHandler {
	return &TelegramHandler{
		bot:             b,
		cfg:             cfg,
		commandHandler:  commandHandler,
		callbackHandler: callbackHandler,
		paymentService:  paymentService,
		userService:     userService,
	}
}

// HandleTelegramWebhook processes incoming Telegram updates.
func (h *TelegramHandler) HandleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading telegram webhook body: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var update bot.Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("Error parsing telegram update: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Process asynchronously to respond quickly
	go func() {
		ctx := context.Background()
		if update.Message != nil {
			h.commandHandler.HandleCommand(ctx, update.Message)
		} else if update.CallbackQuery != nil {
			h.callbackHandler.HandleCallback(ctx, update.CallbackQuery)
		}
	}()

	// Always respond 200 OK to Telegram
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

// HandleSepayWebhook processes incoming SePay payment notifications.
func (h *TelegramHandler) HandleSepayWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading sepay webhook body: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log raw webhook for audit
	log.Printf("SePay webhook received: %s", string(body))

	// Verify signature
	signature := r.Header.Get("Authorization")
	if !sepay.VerifyWebhookSignature(h.cfg.SepayWebhookSecret, string(body), signature) {
		log.Printf("SePay webhook signature verification failed")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload sepay.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Error parsing sepay webhook: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Process payment
	ctx := r.Context()
	result, err := h.paymentService.HandleSepayWebhook(ctx, &payload)
	if err != nil {
		log.Printf("Error processing sepay webhook: %v", err)
		// Still return 200 to prevent SePay from retrying
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
		return
	}

	// Send Telegram notification to user
	if result != nil {
		go h.notifyUser(result)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}

// notifyUser sends a Telegram message to the user about payment result.
func (h *TelegramHandler) notifyUser(result *service.PaymentResult) {
	ctx := context.Background()
	user, err := h.userService.GetUserByID(ctx, result.UserID)
	if err != nil || user == nil {
		log.Printf("Cannot find user to notify: %s, err: %v", result.UserID, err)
		return
	}

	var text string
	if result.Success && result.Link != nil {
		text = bot.MsgPaymentReceived("Sản phẩm", result.Link.Link)
	} else if result.NeedsRefund {
		text = bot.MsgRefundNotice(result.Payment.Amount, "Sản phẩm đã hết hàng")
	} else if result.Success && result.Payment != nil && result.Payment.PaymentType == "DEPOSIT" {
		text = result.Message
	} else {
		text = result.Message
	}

	if text != "" {
		h.bot.SendMessage(user.TelegramID, text, nil)
	}
}

// HandleHealth returns a simple health check response.
func (h *TelegramHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}
