package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Bot wraps the Telegram Bot API.
type Bot struct {
	token  string
	apiURL string
	client *http.Client
}

// NewBot creates a new Telegram Bot instance.
func NewBot(token string) *Bot {
	return &Bot{
		token:  token,
		apiURL: fmt.Sprintf("https://api.telegram.org/bot%s", token),
		client: &http.Client{},
	}
}

// Update represents an incoming update from Telegram.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text,omitempty"`
}

// CallbackQuery represents a callback query from an inline keyboard.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}

// User represents a Telegram user.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID int64 `json:"id"`
}

// InlineKeyboardMarkup represents an inline keyboard.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a button in an inline keyboard.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// ReplyKeyboardMarkup represents a custom keyboard with reply options.
type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard"`
	IsPersistent    bool               `json:"is_persistent"`
	OneTimeKeyboard bool               `json:"one_time_keyboard,omitempty"`
}

// KeyboardButton represents one button of the reply keyboard.
type KeyboardButton struct {
	Text string `json:"text"`
}

// SendMessage sends a text message.
func (b *Bot) SendMessage(chatID int64, text string, keyboard interface{}) error {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if keyboard != nil {
		params["reply_markup"] = keyboard
	}
	return b.callAPI("sendMessage", params)
}

// SendPhoto sends a photo with optional caption.
func (b *Bot) SendPhoto(chatID int64, photoURL, caption string, keyboard interface{}) error {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"photo":      photoURL,
		"caption":    caption,
		"parse_mode": "HTML",
	}
	if keyboard != nil {
		params["reply_markup"] = keyboard
	}
	return b.callAPI("sendPhoto", params)
}

// EditMessageText edits the text of an existing message.
func (b *Bot) EditMessageText(chatID int64, messageID int64, text string, keyboard *InlineKeyboardMarkup) error {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if keyboard != nil {
		params["reply_markup"] = keyboard
	}
	return b.callAPI("editMessageText", params)
}

// DeleteMessage deletes a message in a chat.
func (b *Bot) DeleteMessage(chatID int64, messageID int64) error {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	return b.callAPI("deleteMessage", params)
}

// DeleteMessages deletes multiple messages in a chat simultaneously (up to 100 messages).
func (b *Bot) DeleteMessages(chatID int64, messageIDs []int64) error {
	if len(messageIDs) == 0 {
		return nil
	}
	if len(messageIDs) > 100 {
		messageIDs = messageIDs[:100]
	}
	params := map[string]interface{}{
		"chat_id":     chatID,
		"message_ids": messageIDs,
	}
	return b.callAPI("deleteMessages", params)
}

// AnswerCallbackQuery answers a callback query.
func (b *Bot) AnswerCallbackQuery(callbackQueryID, text string) error {
	params := map[string]interface{}{
		"callback_query_id": callbackQueryID,
	}
	if text != "" {
		params["text"] = text
	}
	return b.callAPI("answerCallbackQuery", params)
}

// SetWebhook sets the webhook URL for the bot.
func (b *Bot) SetWebhook(url string) error {
	params := map[string]interface{}{
		"url":             url,
		"allowed_updates": []string{"message", "callback_query"},
	}
	return b.callAPI("setWebhook", params)
}

// DeleteWebhook removes the webhook.
func (b *Bot) DeleteWebhook() error {
	return b.callAPI("deleteWebhook", nil)
}

// callAPI makes a POST request to the Telegram Bot API.
func (b *Bot) callAPI(method string, params map[string]interface{}) error {
	url := fmt.Sprintf("%s/%s", b.apiURL, method)

	var body io.Reader
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Telegram API error: status=%d, body=%s, method=%s", resp.StatusCode, string(respBody), method)
		return fmt.Errorf("telegram API error: %d", resp.StatusCode)
	}

	return nil
}

// SendMessageToUser sends a message to a specific user by their Telegram ID.
// Convenience method for notification use cases.
func (b *Bot) SendMessageToUser(telegramID int64, text string) error {
	return b.SendMessage(telegramID, text, nil)
}
