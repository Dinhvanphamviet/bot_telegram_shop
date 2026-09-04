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
	stateManager   *StateManager
}

// NewCallbackHandler creates a new CallbackHandler.
func NewCallbackHandler(
	bot *Bot,
	cfg *config.Config,
	userService *service.UserService,
	productService *service.ProductService,
	orderService *service.OrderService,
	walletService *service.WalletService,
	stateManager *StateManager,
) *CallbackHandler {
	return &CallbackHandler{
		bot:            bot,
		cfg:            cfg,
		userService:    userService,
		productService: productService,
		orderService:   orderService,
		walletService:  walletService,
		stateManager:   stateManager,
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
	case data == "close":
		h.stateManager.Clear(cb.From.ID)
		_ = h.bot.DeleteMessage(chatID, msgID)

	case data == "products" || data == "products:refresh":
		h.stateManager.Clear(cb.From.ID)
		if data == "products:refresh" {
			_ = h.bot.AnswerCallbackQuery(cb.ID, "🔄 Đã làm mới danh sách!")
		}
		h.handleProducts(ctx, chatID, msgID)

	case data == "wallet":
		h.stateManager.Clear(cb.From.ID)
		h.handleWallet(ctx, chatID, msgID, user.ID)

	case data == "orders":
		h.stateManager.Clear(cb.From.ID)
		h.handleOrders(ctx, chatID, msgID, user.ID, 0)

	case data == "help":
		h.stateManager.Clear(cb.From.ID)
		h.bot.EditMessageText(chatID, msgID, MsgHelp(), KbBackToMenu())

	case strings.HasPrefix(data, "product:"):
		h.stateManager.Clear(cb.From.ID)
		h.handleProductDetail(ctx, chatID, msgID, user.ID, data[8:])

	case strings.HasPrefix(data, "item:"):
		h.stateManager.Clear(cb.From.ID)
		h.handleItemDetail(ctx, chatID, msgID, user.ID, data[5:])

	case strings.HasPrefix(data, "qty:"):
		// Format: qty:{itemID}:{count}
		h.stateManager.Clear(cb.From.ID)
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			qty, _ := strconv.Atoi(parts[2])
			h.handleSelectQuantity(ctx, chatID, msgID, user.ID, parts[1], qty)
		}

	case strings.HasPrefix(data, "qty_custom:"):
		// Format: qty_custom:{itemID}
		h.handlePromptCustomQuantity(ctx, chatID, msgID, cb.From.ID, data[11:])

	case strings.HasPrefix(data, "confirm_buy_wallet:"):
		// Format: confirm_buy_wallet:{itemID}:{count}
		h.stateManager.Clear(cb.From.ID)
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			qty, _ := strconv.Atoi(parts[2])
			h.handleConfirmBuyWallet(ctx, chatID, msgID, user.ID, parts[1], qty)
		}

	case strings.HasPrefix(data, "confirm_buy_qr:"):
		// Format: confirm_buy_qr:{itemID}:{count}
		h.stateManager.Clear(cb.From.ID)
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			qty, _ := strconv.Atoi(parts[2])
			h.handleConfirmBuyQR(ctx, chatID, msgID, user.ID, parts[1], qty)
		}

	case strings.HasPrefix(data, "buy_qr:"):
		h.handleBuyQR(ctx, chatID, msgID, user.ID, data[7:])

	case strings.HasPrefix(data, "buy_wallet:"):
		h.handleBuyWallet(ctx, chatID, msgID, user.ID, data[11:])

	case strings.HasPrefix(data, "confirm_wallet:"):
		h.handleConfirmWallet(ctx, chatID, msgID, user.ID, data[15:])

	case strings.HasPrefix(data, "deposit:"):
		h.handleDeposit(ctx, chatID, msgID, user.ID, cb.From.ID, data[8:])

	case data == "wallet:history":
		h.handleWalletHistory(ctx, chatID, msgID, user.ID)

	case strings.HasPrefix(data, "orders:page:"):
		page, _ := strconv.Atoi(data[12:])
		h.handleOrders(ctx, chatID, msgID, user.ID, page)

	case data == "back:menu":
		h.stateManager.Clear(cb.From.ID)
		firstName := cb.From.FirstName
		if firstName == "" {
			firstName = "bạn"
		}
		h.bot.EditMessageText(chatID, msgID, MsgWelcome(firstName), KbMainMenu())

	case data == "back:products":
		h.stateManager.Clear(cb.From.ID)
		h.handleProducts(ctx, chatID, msgID)

	case strings.HasPrefix(data, "back:product:"):
		h.stateManager.Clear(cb.From.ID)
		h.handleProductDetail(ctx, chatID, msgID, user.ID, data[13:])

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
		h.bot.EditMessageText(chatID, msgID, MsgNoProducts(), KbNoProducts())
		return
	}
	h.bot.EditMessageText(chatID, msgID, MsgProductList(), KbProductList(products))
}

func (h *CallbackHandler) handleProductDetail(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string) {
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

	// Nếu sản phẩm chỉ có đúng 1 gói duy nhất: Bỏ qua màn hình trung gian, nhảy thẳng vào chi tiết và chọn số lượng!
	if len(items) == 1 {
		h.handleItemDetail(ctx, chatID, msgID, userID, items[0].ID.String())
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

func (h *CallbackHandler) handleItemDetail(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string) {
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

	product, items, _ := h.productService.GetProductWithItems(ctx, item.ProductID)
	productName := ""
	if product != nil {
		productName = product.Name
	}

	balance, _ := h.walletService.GetBalance(ctx, userID)
	shortID := strings.ToUpper(item.ID.String()[:4])

	// Nếu sản phẩm chỉ có 1 gói duy nhất thì nút Quay lại sẽ về thẳng danh sách sản phẩm
	backCallback := fmt.Sprintf("back:product:%s", item.ProductID)
	if len(items) <= 1 {
		backCallback = "products"
	}

	text := MsgItemDetail(productName, item.Name, item.Description, shortID, item.Price, balance, availableCount)
	h.bot.EditMessageText(chatID, msgID, text, KbItemDetail(itemID, availableCount, backCallback))
}

func (h *CallbackHandler) handleSelectQuantity(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string, quantity int) {
	if quantity <= 0 {
		quantity = 1
	}

	itemID, err := uuid.Parse(idStr)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	item, availableCount, err := h.productService.GetItemDetail(ctx, itemID)
	if err != nil || item == nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	_, items, _ := h.productService.GetProductWithItems(ctx, item.ProductID)
	backCallback := fmt.Sprintf("back:product:%s", item.ProductID)
	if len(items) <= 1 {
		backCallback = "products"
	}

	if availableCount < quantity {
		h.bot.EditMessageText(chatID, msgID,
			fmt.Sprintf("⚠️ Kho hiện chỉ còn <b>%d</b> sản phẩm, không đủ số lượng bạn chọn (%d).", availableCount, quantity),
			KbItemDetail(itemID, availableCount, backCallback))
		return
	}

	balance, _ := h.walletService.GetBalance(ctx, userID)
	totalAmount := item.Price * int64(quantity)
	canWallet := balance >= totalAmount

	text := MsgConfirmOrder(item.Name, quantity, item.Price, totalAmount, balance)
	h.bot.EditMessageText(chatID, msgID, text, KbConfirmOrder(itemID, quantity, canWallet))
}

func (h *CallbackHandler) handlePromptCustomQuantity(ctx context.Context, chatID, msgID int64, telegramID int64, idStr string) {
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	item, availableCount, err := h.productService.GetItemDetail(ctx, itemID)
	if err != nil || item == nil {
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	if availableCount <= 0 {
		h.bot.EditMessageText(chatID, msgID, "🔴 Sản phẩm này hiện đã hết hàng.", KbBackToMenu())
		return
	}

	h.stateManager.Set(telegramID, UserState{
		Action:    ActionWaitingQuantity,
		ItemID:    itemID,
		ProductID: item.ProductID,
		ItemName:  item.Name,
		Stock:     availableCount,
	})

	h.bot.EditMessageText(chatID, msgID,
		MsgPromptCustomQuantity(item.Name, availableCount),
		KbCancel(fmt.Sprintf("item:%s", itemID)))
}

func (h *CallbackHandler) handleConfirmBuyWallet(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string, quantity int) {
	if quantity <= 0 {
		quantity = 1
	}

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

	order, links, err := h.orderService.PurchaseWithBalance(ctx, userID, itemID, quantity)
	if err != nil {
		log.Printf("Error purchasing with wallet: %v", err)
		h.bot.EditMessageText(chatID, msgID, fmt.Sprintf("❌ %s", err.Error()), KbBackToMenu())
		return
	}

	var linkTexts []string
	for _, l := range links {
		linkTexts = append(linkTexts, l.Link)
	}

	h.bot.EditMessageText(chatID, msgID,
		MsgPurchaseSuccessMulti(item.Name, quantity, order.Amount, linkTexts),
		KbBackToMenu())
}

func (h *CallbackHandler) handleConfirmBuyQR(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string, quantity int) {
	if quantity <= 0 {
		quantity = 1
	}

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

	order, payment, qrURL, err := h.orderService.PurchaseWithQR(ctx, userID, itemID, quantity)
	if err != nil {
		log.Printf("Error creating QR purchase: %v", err)
		h.bot.EditMessageText(chatID, msgID, fmt.Sprintf("❌ %s", err.Error()), KbBackToMenu())
		return
	}

	caption := MsgQRPaymentOrder(item.Name, quantity, order.Amount, payment.TransferContent, 15)
	if err := h.bot.SendPhoto(chatID, qrURL, caption, KbBackToMenu()); err != nil {
		log.Printf("Error sending purchase QR: %v", err)
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}
	h.bot.EditMessageText(chatID, msgID, "✅ Đã tạo mã QR thanh toán bên dưới 👇", nil)
}

func (h *CallbackHandler) handleBuyQR(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string) {
	h.handleConfirmBuyQR(ctx, chatID, msgID, userID, idStr, 1)
}

func (h *CallbackHandler) handleBuyWallet(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string) {
	h.handleSelectQuantity(ctx, chatID, msgID, userID, idStr, 1)
}

func (h *CallbackHandler) handleConfirmWallet(ctx context.Context, chatID, msgID int64, userID uuid.UUID, idStr string) {
	h.handleConfirmBuyWallet(ctx, chatID, msgID, userID, idStr, 1)
}

func (h *CallbackHandler) handleDeposit(ctx context.Context, chatID, msgID int64, userID uuid.UUID, telegramID int64, amountStr string) {
	if amountStr == "custom" {
		h.stateManager.Set(telegramID, UserState{
			Action: ActionWaitingDepositAmount,
		})
		h.bot.EditMessageText(chatID, msgID, MsgPromptCustomDeposit(), KbCancel("wallet"))
		return
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount < 10000 {
		h.bot.EditMessageText(chatID, msgID, "❌ Số tiền không hợp lệ (tối thiểu 10.000đ)", KbBackToMenu())
		return
	}

	payment, qrURL, err := h.walletService.DepositWithQR(ctx, userID, amount)
	if err != nil {
		log.Printf("Error creating deposit QR: %v", err)
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}

	if err := h.bot.SendPhoto(chatID, qrURL, MsgDepositQR(amount, payment.TransferContent), KbBackToMenu()); err != nil {
		log.Printf("Error sending deposit QR: %v", err)
		h.bot.EditMessageText(chatID, msgID, MsgError(), KbBackToMenu())
		return
	}
	h.bot.EditMessageText(chatID, msgID, "✅ Đã tạo mã QR nạp tiền bên dưới 👇", nil)
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

