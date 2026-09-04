package bot

import (
	"fmt"

	"telegram-shop/internal/model"

	"github.com/google/uuid"
)

// KbMainMenu builds the main menu keyboard.
func KbMainMenu() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "🛒 Sản phẩm", CallbackData: "products"}},
			{
				{Text: "💰 Ví của tôi", CallbackData: "wallet"},
				{Text: "📦 Đơn hàng", CallbackData: "orders"},
			},
			{{Text: "❓ Trợ giúp", CallbackData: "help"}},
		},
	}
}

// KbProductList builds the product list keyboard.
func KbProductList(products []model.Product) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, p := range products {
		rows = append(rows, []InlineKeyboardButton{
			{Text: fmt.Sprintf("📦 %s", p.Name), CallbackData: fmt.Sprintf("product:%s", p.ID)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔙 Menu", CallbackData: "back:menu"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// KbItemList builds the item list keyboard for a product.
func KbItemList(items []model.ItemWithStock, productID uuid.UUID) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, it := range items {
		var stockIndicator string
		if it.AvailableCount > 0 {
			stockIndicator = MsgInStock(it.AvailableCount)
		} else {
			stockIndicator = MsgOutOfStock()
		}
		text := fmt.Sprintf("%s — %s %s", it.Name, FormatMoney(it.Price), stockIndicator)
		rows = append(rows, []InlineKeyboardButton{
			{Text: text, CallbackData: fmt.Sprintf("item:%s", it.ID)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔙 Quay lại", CallbackData: "back:products"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// KbItemDetail builds the item detail keyboard with buy buttons.
func KbItemDetail(itemID uuid.UUID, available int, productID uuid.UUID) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	if available > 0 {
		rows = append(rows, []InlineKeyboardButton{
			{Text: "💳 Mua bằng QR", CallbackData: fmt.Sprintf("buy_qr:%s", itemID)},
		})
		rows = append(rows, []InlineKeyboardButton{
			{Text: "💰 Mua bằng Ví", CallbackData: fmt.Sprintf("buy_wallet:%s", itemID)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔙 Quay lại", CallbackData: fmt.Sprintf("back:product:%s", productID)},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// KbConfirmWalletPurchase builds the confirmation keyboard for wallet purchase.
func KbConfirmWalletPurchase(itemID uuid.UUID) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "✅ Xác nhận mua", CallbackData: fmt.Sprintf("confirm_wallet:%s", itemID)},
				{Text: "❌ Hủy", CallbackData: "back:menu"},
			},
		},
	}
}

// KbWallet builds the wallet menu keyboard.
func KbWallet() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "💵 Nạp 50.000đ", CallbackData: "deposit:50000"},
				{Text: "💵 Nạp 100.000đ", CallbackData: "deposit:100000"},
			},
			{
				{Text: "💵 Nạp 200.000đ", CallbackData: "deposit:200000"},
				{Text: "💵 Nạp 500.000đ", CallbackData: "deposit:500000"},
			},
			{{Text: "📜 Lịch sử giao dịch", CallbackData: "wallet:history"}},
			{{Text: "🔙 Menu", CallbackData: "back:menu"}},
		},
	}
}

// KbBackToMenu builds a simple back-to-menu keyboard.
func KbBackToMenu() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "🔙 Menu", CallbackData: "back:menu"}},
		},
	}
}

// KbOrders builds the orders navigation keyboard.
func KbOrders(hasMore bool, currentPage int) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	var navRow []InlineKeyboardButton
	if currentPage > 0 {
		navRow = append(navRow, InlineKeyboardButton{
			Text: "⬅️ Trước", CallbackData: fmt.Sprintf("orders:page:%d", currentPage-1),
		})
	}
	if hasMore {
		navRow = append(navRow, InlineKeyboardButton{
			Text: "Tiếp ➡️", CallbackData: fmt.Sprintf("orders:page:%d", currentPage+1),
		})
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔙 Menu", CallbackData: "back:menu"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// KbAdminMenu builds the admin menu keyboard.
func KbAdminMenu() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "📦 Sản phẩm", CallbackData: "admin:products"}},
			{{Text: "🔗 Thêm Link", CallbackData: "admin:add_links"}},
			{{Text: "👤 Users", CallbackData: "admin:users"}},
			{{Text: "📊 Thống kê", CallbackData: "admin:stats"}},
			{{Text: "🔙 Menu", CallbackData: "back:menu"}},
		},
	}
}
