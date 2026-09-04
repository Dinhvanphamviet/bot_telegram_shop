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

// CallbackHandler handles inline keyboard callback queries.
type CallbackHandler struct {
	bot            *Bot
	cfg            *config.Config
	userService    *service.UserService
	productService *service.ProductService
	orderService   *service.OrderService
	walletService  *service.WalletService
}

// NewCallbackHandler creates a new CallbackHandler.
func NewCallbackHandler(
	bot *Bot,
	cfg *config.Config,
	userService *service.UserService,
	productService *service.ProductService,
	orderService *service.OrderService,
	walletService *service.WalletService,
) *CallbackHandler {
	return &CallbackHandler{
		bot:            bot,
		cfg:            cfg,
		userService:    userService,
		productService: productService,
		orderService:   orderService,
		walletService:  walletService,
	}
}

// HandleCallback routes callback queries to appropriate handlers.
func (h *CallbackHandler) HandleCallback(ctx context.Context, cb *CallbackQuery) {
	// Always answer callback to remove loading indicator
	defer h.bot.AnswerCallbackQuery(cb.ID, "")

	if cb.Message == nil || cb.From == nil {
		return
	}

	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID

	// Ensure user exists
	user, err := h.userService.GetOrCreateUser(ctx, cb.From.ID, cb.From.Username, cb.From.FirstName)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	data := cb.Data

	switch {
	case data == "products":
		h.handleProducts(ctx, chatID, msgID)

	case data == "wallet":
		h.handleWallet(ctx, chatID, msgID, user.ID)

	case data == "orders":
		h.handleOrders(ctx, chatID, msgID, user.ID, 0)

	case data == "help":
		h.bot.EditMessageText(chatID, msgID, MsgHelp(), KbBackToMenu())

	case strings.HasPrefix(data, "product:"):
		h.handleProductDetail(ctx, chatID, msgID, data[8:])

	case strings.HasPrefix(data, "item:"):
		h.handleItemDetail(ctx, chatID, msgID, data[5:])

	case strings.HasPrefix(data, "buy_qr:"):
		h.handleBuyQR(ctx, chatID, msgID, user.ID, data[7:])

	case strings.HasPrefix(data, "buy_wallet:"):
		h.handleBuyWallet(ctx, chatID, msgID, user.ID, data[11:])

	case strings.HasPrefix(data, "confirm_wallet:"):
		h.handleConfirmWallet(ctx, chatID, msgID, user.ID, data[15:])

	case strings.HasPrefix(data, "deposit:"):
		h.handleDeposit(ctx, chatID, msgID, user.ID, data[8:])

	case data == "wallet:history":
		h.handleWalletHistory(ctx, chatID, msgID, user.ID)

	case strings.HasPrefix(data, "orders:page:"):
		page, _ := strconv.Atoi(data[12:])
		h.handleOrders(ctx, chatID, msgID, user.ID, page)

	case data == "back:menu":
		firstName := cb.From.FirstName
		if firstName == "" {
			firstName = "bạn"
		}
		h.bot.EditMessageText(chatID, msgID, MsgWelcome(firstName), KbMainMenu())

	case data == "back:products":
		h.handleProducts(ctx, chatID, msgID)

	case strings.HasPrefix(data, "back:product:"):
		h.handleProductDetail(ctx, chatID, msgID, data[13:])

	case strings.HasPrefix(data, "admin:"):
		if !h.cfg.IsAdmin(cb.From.ID) {
			h.bot.AnswerCallbackQuery(cb.ID, "⛔ Không có quyền")
			return
		}
		h.handleAdminCallback(ctx, chatID, msgID, data[6:])
	}
}

func (h *CallbackHandler) handleProducts(ctx context.Context, chatID, msgID int64) {
	products, err := h.productService.ListProducts(ctx)
	if err != nil {
		log.Printf("Error listing products: %v", err)
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}
	if len(products) == 0 {
		h.bot.EditMessageText(chatID, msgID, MsgNoProducts(), KbBackToMenu())
		return
	}
	h.bot.EditMessageText(chatID, msgID, MsgProductList(), KbProductList(products))
}

func (h *CallbackHandler) handleProductDetail(ctx context.Context, chatID, msgID int64, idStr string) {
	productID, err := uuid.Parse(idStr)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	product, items, err := h.productService.GetProductWithItems(ctx, productID)
	if err != nil || product == nil {
		log.Printf("Error getting product: %v", err)
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	text := fmt.Sprintf("📦 <b>%s</b>\n", product.Name)
	if product.Description != "" {
		text += fmt.Sprintf("\n%s\n", product.Description)
	}
	text += "\nChọn gói bạn muốn mua:"

	if len(items) == 0 {
		text += "\n\n😔 Chưa có gói nào."
	}

	h.bot.EditMessageText(chatID, msgID, text, KbItemList(items, productID))
}

func (h *CallbackHandler) handleItemDetail(ctx context.Context, chatID, msgID int64, idStr string) {
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	item, availableCount, err := h.productService.GetItemDetail(ctx, itemID)
	if err != nil || item == nil {
		log.Printf("Error getting item: %v", err)
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	var stockStatus string
	if availableCount > 0 {
		stockStatus = MsgInStock(availableCount)
	} else {
		stockStatus = MsgOutOfStock()
	}

	text := fmt.Sprintf("🛒 <b>%s</b>\n\n💰 Giá: <b>%s</b>\n📊 Tình trạng: %s",
		item.Name, FormatMoney(item.Price), stockStatus)
	if item.Description != "" {
		text += fmt.Sprintf("\n\n📝 %s", item.Description)
	}

	h.bot.EditMessageText(chatID, msgID, text, KbItemDetail(itemID, availableCount, item.ProductID))
}

func (h *CallbackHandler) handleBuyQR(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string) {
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	item, err := h.productService.GetItem(ctx, itemID)
	if err != nil || item == nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	order, payment, qrURL, err := h.orderService.PurchaseWithQR(ctx, userID, itemID)
	if err != nil {
		log.Printf("Error creating QR purchase: %v", err)
		h.bot.EditMessageText(chatID, msgID, fmt.Sprintf("❌ %s", err.Error()), KbBackToMenu())
		return
	}

	// Send QR image
	caption := MsgQRPayment(item.Name, order.Amount, payment.TransferContent, 15)
	h.bot.EditMessageText(chatID, msgID, "⏳ Đang tạo QR thanh toán...", nil)
	h.bot.SendPhoto(chatID, qrURL, caption, KbBackToMenu())
}

func (h *CallbackHandler) handleBuyWallet(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string) {
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	item, err := h.productService.GetItem(ctx, itemID)
	if err != nil || item == nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	balance, err := h.walletService.GetBalance(ctx, userID)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	if balance < item.Price {
		text := fmt.Sprintf("❌ <b>Số dư không đủ!</b>\n\n💰 Giá: %s\n💵 Số dư: %s\n\nVui lòng nạp thêm tiền.",
			FormatMoney(item.Price), FormatMoney(balance))
		h.bot.EditMessageText(chatID, msgID, text, KbWallet())
		return
	}

	h.bot.EditMessageText(chatID, msgID,
		MsgConfirmPurchase(item.Name, item.Price, balance),
		KbConfirmWalletPurchase(itemID))
}

func (h *CallbackHandler) handleConfirmWallet(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string) {
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	item, err := h.productService.GetItem(ctx, itemID)
	if err != nil || item == nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	_, link, err := h.orderService.PurchaseWithBalance(ctx, userID, itemID)
	if err != nil {
		log.Printf("Error purchasing with wallet: %v", err)
		h.bot.EditMessageText(chatID, msgID, fmt.Sprintf("❌ %s", err.Error()), KbBackToMenu())
		return
	}

	h.bot.EditMessageText(chatID, msgID,
		MsgPurchaseSuccess(item.Name, link.Link),
		KbBackToMenu())
}

func (h *CallbackHandler) handleDeposit(ctx context.Context, chatID, msgID int64, userID uuid.UUID, amountStr string) {
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount < 10000 {
		h.bot.EditMessageText(chatID, msgID, "❌ Số tiền không hợp lệ", KbBackToMenu())
		return
	}

	payment, qrURL, err := h.walletService.DepositWithQR(ctx, userID, amount)
	if err != nil {
		log.Printf("Error creating deposit QR: %v", err)
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	h.bot.EditMessageText(chatID, msgID, "⏳ Đang tạo QR nạp tiền...", nil)
	h.bot.SendPhoto(chatID, qrURL, MsgDepositQR(amount, payment.TransferContent), KbBackToMenu())
}

func (h *CallbackHandler) handleWallet(ctx context.Context, chatID, msgID int64, userID uuid.UUID) {
	balance, err := h.walletService.GetBalance(ctx, userID)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}
	h.bot.EditMessageText(chatID, msgID, MsgWalletInfo(balance), KbWallet())
}

func (h *CallbackHandler) handleWalletHistory(ctx context.Context, chatID, msgID int64, userID uuid.UUID) {
	txs, err := h.walletService.GetTransactions(ctx, userID, 10, 0)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	if len(txs) == 0 {
		h.bot.EditMessageText(chatID, msgID, "📜 Chưa có giao dịch nào.", KbBackToMenu())
		return
	}

	text := "📜 <b>Lịch sử giao dịch</b>\n"
	for _, tx := range txs {
		sign := "+"
		if tx.Amount < 0 {
			sign = ""
		}
		text += fmt.Sprintf("\n%s %s%s\n%s | %s\n",
			FormatWalletType(tx.Type),
			sign, FormatMoney(tx.Amount),
			tx.Description,
			FormatDate(tx.CreatedAt))
	}

	h.bot.EditMessageText(chatID, msgID, text, KbBackToMenu())
}

func (h *CallbackHandler) handleOrders(ctx context.Context, chatID, msgID int64, userID uuid.UUID, page int) {
	limit := 5
	offset := page * limit

	orders, err := h.orderService.GetUserOrders(ctx, userID, limit, offset)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	if len(orders) == 0 && page == 0 {
		h.bot.EditMessageText(chatID, msgID, MsgNoOrders(), KbBackToMenu())
		return
	}

	text := MsgOrderHistory()
	for _, o := range orders {
		text += FormatOrderLine(o.ProductName, o.ItemName, o.Status, o.PaymentMethod, o.Amount, FormatDate(o.CreatedAt))
	}

	hasMore := len(orders) == limit
	h.bot.EditMessageText(chatID, msgID, text, KbOrders(hasMore, page))
}

