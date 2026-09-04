package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"telegram-shop/internal/config"
	"telegram-shop/internal/service"

	"github.com/google/uuid"
)

// CommandHandler handles Telegram text commands.
type CommandHandler struct {
	bot            *Bot
	cfg            *config.Config
	userService    *service.UserService
	productService *service.ProductService
	orderService   *service.OrderService
	walletService  *service.WalletService
	stateManager   *StateManager
}

// NewCommandHandler creates a new CommandHandler.
func NewCommandHandler(
	bot *Bot,
	cfg *config.Config,
	userService *service.UserService,
	productService *service.ProductService,
	orderService *service.OrderService,
	walletService *service.WalletService,
	stateManager *StateManager,
) *CommandHandler {
	return &CommandHandler{
		bot:            bot,
		cfg:            cfg,
		userService:    userService,
		productService: productService,
		orderService:   orderService,
		walletService:  walletService,
		stateManager:   stateManager,
	}
}

// HandleCommand routes text commands and handles conversational input states.
func (h *CommandHandler) HandleCommand(ctx context.Context, msg *Message) {
	if msg.From == nil {
		return
	}

	// Ensure user exists
	user, err := h.userService.GetOrCreateUser(ctx, msg.From.ID, msg.From.Username, msg.From.FirstName)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		h.bot.SendMessage(msg.Chat.ID, MsgError(), nil)
		return
	}

	// If message is a slash command, clear state and route command
	if strings.HasPrefix(msg.Text, "/") {
		h.stateManager.Clear(msg.From.ID)
		switch msg.Text {
		case "/start", "/menu":
			h.handleStart(ctx, msg)
		case "/products":
			h.handleProducts(ctx, msg)
		case "/wallet":
			h.handleWallet(ctx, msg)
		case "/orders":
			h.handleOrders(ctx, msg)
		case "/clear", "/clean":
			h.handleClear(ctx, msg)
		case "/help":
			h.handleHelp(ctx, msg)
		case "/admin":
			h.handleAdmin(ctx, msg)
		default:
			h.handleStart(ctx, msg)
		}
		return
	}

	// Check if user is in an active input state
	state, exists := h.stateManager.Get(msg.From.ID)
	if !exists {
		// Normal text without state — show main menu
		h.handleStart(ctx, msg)
		return
	}

	switch state.Action {
	case ActionWaitingDepositAmount:
		h.handleInputDepositAmount(ctx, msg, user.ID)
	case ActionWaitingQuantity:
		h.handleInputQuantity(ctx, msg, state, user.Balance)
	default:
		h.handleStart(ctx, msg)
	}
}

func (h *CommandHandler) handleStart(_ context.Context, msg *Message) {
	firstName := "bạn"
	if msg.From != nil && msg.From.FirstName != "" {
		firstName = msg.From.FirstName
	}
	h.bot.SendMessage(msg.Chat.ID, MsgWelcome(firstName), KbMainMenu())
}

func (h *CommandHandler) handleProducts(ctx context.Context, msg *Message) {
	products, err := h.productService.ListProducts(ctx)
	if err != nil {
		log.Printf("Error listing products: %v", err)
		h.bot.SendMessage(msg.Chat.ID, MsgError(), nil)
		return
	}
	if len(products) == 0 {
		h.bot.SendMessage(msg.Chat.ID, MsgNoProducts(), KbNoProducts())
		return
	}
	h.bot.SendMessage(msg.Chat.ID, MsgProductList(), KbProductList(products))
}

func (h *CommandHandler) handleWallet(ctx context.Context, msg *Message) {
	user, err := h.userService.GetUserByTelegramID(ctx, msg.From.ID)
	if err != nil || user == nil {
		log.Printf("Error getting user: %v", err)
		h.bot.SendMessage(msg.Chat.ID, MsgError(), nil)
		return
	}
	h.bot.SendMessage(msg.Chat.ID, MsgWalletInfo(user.Balance), KbWallet())
}

func (h *CommandHandler) handleOrders(ctx context.Context, msg *Message) {
	user, err := h.userService.GetUserByTelegramID(ctx, msg.From.ID)
	if err != nil || user == nil {
		log.Printf("Error getting user: %v", err)
		h.bot.SendMessage(msg.Chat.ID, MsgError(), nil)
		return
	}

	orders, err := h.orderService.GetUserOrders(ctx, user.ID, 5, 0)
	if err != nil {
		log.Printf("Error getting orders: %v", err)
		h.bot.SendMessage(msg.Chat.ID, MsgError(), nil)
		return
	}

	if len(orders) == 0 {
		h.bot.SendMessage(msg.Chat.ID, MsgNoOrders(), KbBackToMenu())
		return
	}

	text := MsgOrderHistory()
	for _, o := range orders {
		text += FormatOrderLine(o.ProductName, o.ItemName, o.Status, o.PaymentMethod, o.Amount, FormatDate(o.CreatedAt))
	}

	hasMore := len(orders) == 5
	h.bot.SendMessage(msg.Chat.ID, text, KbOrders(hasMore, 0))
}

func (h *CommandHandler) handleHelp(_ context.Context, msg *Message) {
	h.bot.SendMessage(msg.Chat.ID, MsgHelp(), KbBackToMenu())
}

func (h *CommandHandler) handleClear(_ context.Context, msg *Message) {
	// Collect recent message IDs to delete
	var ids []int64
	for i := int64(0); i < 100 && (msg.MessageID-i) > 0; i++ {
		ids = append(ids, msg.MessageID-i)
	}

	err := h.bot.DeleteMessages(msg.Chat.ID, ids)
	if err != nil {
		// Fallback: delete one by one
		for i := int64(0); i < 25 && (msg.MessageID-i) > 0; i++ {
			_ = h.bot.DeleteMessage(msg.Chat.ID, msg.MessageID-i)
		}
	}

	firstName := "bạn"
	if msg.From != nil && msg.From.FirstName != "" {
		firstName = msg.From.FirstName
	}
	h.bot.SendMessage(msg.Chat.ID, fmt.Sprintf("🧹 <b>Đã dọn dẹp đoạn chat!</b>\n\n%s", MsgWelcome(firstName)), KbMainMenu())
}



// FormatOrderLine formats a single order for display in order history.
func FormatOrderLine(productName, itemName, status, paymentMethod string, amount int64, createdAt string) string {
	return fmt.Sprintf(
		"\n📦 <b>%s</b> — %s\n💰 %s | %s | %s\n🕐 %s\n",
		productName, itemName,
		FormatMoney(amount), FormatOrderStatus(status), FormatPaymentMethod(paymentMethod),
		createdAt,
	)
}

func (h *CommandHandler) handleInputDepositAmount(ctx context.Context, msg *Message, userID uuid.UUID) {
	amount, err := ParseAmount(msg.Text)
	if err != nil || amount < 10000 || amount > 50000000 {
		h.bot.SendMessage(msg.Chat.ID,
			"⚠️ <b>Số tiền không hợp lệ!</b>\n\nVui lòng nhập số tiền từ <b>10.000đ</b> đến <b>50.000.000đ</b>\n<i>(Ví dụ: <code>50000</code> hoặc <code>50k</code>)</i>:",
			KbCancel("wallet"))
		return
	}

	h.stateManager.Clear(msg.From.ID)

	payment, qrURL, err := h.walletService.DepositWithQR(ctx, userID, amount)
	if err != nil {
		log.Printf("Error creating deposit QR: %v", err)
		h.bot.SendMessage(msg.Chat.ID, MsgError(), KbBackToMenu())
		return
	}

	if err := h.bot.SendPhoto(msg.Chat.ID, qrURL, MsgDepositQR(amount, payment.TransferContent), KbBackToMenu()); err != nil {
		log.Printf("Error sending deposit QR: %v", err)
		h.bot.SendMessage(msg.Chat.ID, MsgError(), KbBackToMenu())
		return
	}
}

func (h *CommandHandler) handleInputQuantity(ctx context.Context, msg *Message, state UserState, userBalance int64) {
	text := strings.TrimSpace(msg.Text)
	qty, err := strconv.Atoi(text)
	if err != nil || qty <= 0 {
		h.bot.SendMessage(msg.Chat.ID,
			"⚠️ <b>Số lượng không hợp lệ!</b>\n\nVui lòng nhập một số tự nhiên lớn hơn 0 (ví dụ: <code>1</code>, <code>2</code>, <code>5</code>):",
			KbCancel(fmt.Sprintf("item:%s", state.ItemID)))
		return
	}

	item, availableCount, err := h.productService.GetItemDetail(ctx, state.ItemID)
	if err != nil || item == nil {
		h.bot.SendMessage(msg.Chat.ID, MsgError(), KbBackToMenu())
		return
	}

	if qty > availableCount {
		h.bot.SendMessage(msg.Chat.ID,
			fmt.Sprintf("⚠️ <b>Số lượng vượt quá tồn kho!</b>\n\nKho hiện chỉ còn <b>%d</b> sản phẩm. Vui lòng nhập số lượng từ 1 đến %d:", availableCount, availableCount),
			KbCancel(fmt.Sprintf("item:%s", state.ItemID)))
		return
	}

	h.stateManager.Clear(msg.From.ID)

	totalAmount := item.Price * int64(qty)
	canWallet := userBalance >= totalAmount

	confirmText := MsgConfirmOrder(item.Name, qty, item.Price, totalAmount, userBalance)
	h.bot.SendMessage(msg.Chat.ID, confirmText, KbConfirmOrder(state.ItemID, qty, canWallet))
}

