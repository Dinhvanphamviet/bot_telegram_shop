package bot

import (
	"context"
	"fmt"
	"log"

	"telegram-shop/internal/config"
	"telegram-shop/internal/service"
)

// CommandHandler handles Telegram text commands.
type CommandHandler struct {
	bot            *Bot
	cfg            *config.Config
	userService    *service.UserService
	productService *service.ProductService
	orderService   *service.OrderService
	walletService  *service.WalletService
}

// NewCommandHandler creates a new CommandHandler.
func NewCommandHandler(
	bot *Bot,
	cfg *config.Config,
	userService *service.UserService,
	productService *service.ProductService,
	orderService *service.OrderService,
	walletService *service.WalletService,
) *CommandHandler {
	return &CommandHandler{
		bot:            bot,
		cfg:            cfg,
		userService:    userService,
		productService: productService,
		orderService:   orderService,
		walletService:  walletService,
	}
}

// HandleCommand routes text commands to their handlers.
func (h *CommandHandler) HandleCommand(ctx context.Context, msg *Message) {
	if msg.From == nil {
		return
	}

	// Ensure user exists
	_, err := h.userService.GetOrCreateUser(ctx, msg.From.ID, msg.From.Username, msg.From.FirstName)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		h.bot.SendMessage(msg.Chat.ID, MsgError(), nil)
		return
	}

	switch msg.Text {
	case "/start":
		h.handleStart(ctx, msg)
	case "/menu":
		h.handleStart(ctx, msg)
	case "/products":
		h.handleProducts(ctx, msg)
	case "/wallet":
		h.handleWallet(ctx, msg)
	case "/orders":
		h.handleOrders(ctx, msg)
	case "/help":
		h.handleHelp(ctx, msg)
	case "/admin":
		h.handleAdmin(ctx, msg)
	default:
		// Unknown command — show menu
		h.handleStart(ctx, msg)
	}
}

func (h *CommandHandler) handleStart(ctx context.Context, msg *Message) {
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
		h.bot.SendMessage(msg.Chat.ID, MsgNoProducts(), KbBackToMenu())
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

func (h *CommandHandler) handleHelp(ctx context.Context, msg *Message) {
	h.bot.SendMessage(msg.Chat.ID, MsgHelp(), KbBackToMenu())
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
