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

// KbPersistentReplyMenu builds the persistent bottom keyboard with quick access buttons.
func KbPersistentReplyMenu() *ReplyKeyboardMarkup {
	return &ReplyKeyboardMarkup{
		Keyboard: [][]KeyboardButton{
			{
				{Text: "🛒 Sản phẩm"},
				{Text: "💰 Ví của tôi"},
			},
			{
				{Text: "📦 Đơn hàng"},
				{Text: "🧹 Làm sạch"},
			},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// KbProductList builds the product list keyboard.
func KbProductList(products []model.Product) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	for _, p := range products {
		status := "✅"
		if p.TotalStock <= 0 {
			status = "❌"
		}
		rows = append(rows, []InlineKeyboardButton{
			{Text: fmt.Sprintf("📦 %s (%s)", p.Name, status), CallbackData: fmt.Sprintf("product:%s", p.ID)},
		})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔄 Làm mới", CallbackData: "products:refresh"},
		{Text: "🔙 Menu", CallbackData: "back:menu"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// KbNoProducts builds the keyboard when no products are found, with Refresh and Menu on one row.
func KbNoProducts() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔄 Làm mới", CallbackData: "products:refresh"},
				{Text: "🔙 Menu", CallbackData: "back:menu"},
			},
		},
	}
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
		{Text: "🔄 Làm mới", CallbackData: fmt.Sprintf("product:%s", productID)},
		{Text: "🔙 Quay lại", CallbackData: "back:products"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// KbItemDetail builds the item detail keyboard with quantity selection buttons matching reference UI.
func KbItemDetail(itemID uuid.UUID, available int, backCallback string) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	if available > 0 {
		rows = append(rows, []InlineKeyboardButton{
			{Text: "1", CallbackData: fmt.Sprintf("qty:%s:1", itemID)},
			{Text: "2", CallbackData: fmt.Sprintf("qty:%s:2", itemID)},
		})
		rows = append(rows, []InlineKeyboardButton{
			{Text: "3", CallbackData: fmt.Sprintf("qty:%s:3", itemID)},
			{Text: "5", CallbackData: fmt.Sprintf("qty:%s:5", itemID)},
		})
		rows = append(rows, []InlineKeyboardButton{
			{Text: "10", CallbackData: fmt.Sprintf("qty:%s:10", itemID)},
		})
		rows = append(rows, []InlineKeyboardButton{
			{Text: "✏️ Nhập số khác", CallbackData: fmt.Sprintf("qty_custom:%s", itemID)},
		})
	}
	if backCallback == "" {
		backCallback = "products"
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "🔄 Làm mới", CallbackData: fmt.Sprintf("item:%s", itemID)},
		{Text: "⬅️ Quay lại", CallbackData: backCallback},
		{Text: "❌ Đóng", CallbackData: "close"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// KbConfirmOrder builds the purchase confirmation keyboard for a specific quantity.
func KbConfirmOrder(itemID uuid.UUID, quantity int, canWallet bool) *InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	if canWallet {
		rows = append(rows, []InlineKeyboardButton{
			{Text: "💰 Mua bằng Ví (nhanh)", CallbackData: fmt.Sprintf("confirm_buy_wallet:%s:%d", itemID, quantity)},
		})
		rows = append(rows, []InlineKeyboardButton{
			{Text: "💳 Mua bằng QR", CallbackData: fmt.Sprintf("confirm_buy_qr:%s:%d", itemID, quantity)},
		})
	} else {
		rows = append(rows, []InlineKeyboardButton{
			{Text: "💳 Mua bằng QR", CallbackData: fmt.Sprintf("confirm_buy_qr:%s:%d", itemID, quantity)},
		})
		rows = append(rows, []InlineKeyboardButton{
			{Text: "💵 Nạp tiền vào ví", CallbackData: "wallet"},
		})
	}
	rows = append(rows, []InlineKeyboardButton{
		{Text: "⬅️ Chọn lại số lượng", CallbackData: fmt.Sprintf("item:%s", itemID)},
		{Text: "❌ Đóng", CallbackData: "close"},
	})
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}

// KbCancel builds a cancel button returning to a specific callback.
func KbCancel(backCallback string) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "❌ Hủy thao tác", CallbackData: backCallback},
			},
		},
	}
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

// KbWallet builds the wallet menu keyboard with custom deposit option.
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
			{
				{Text: "✏️ Nhập số khác", CallbackData: "deposit:custom"},
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

// KbCancelPayment builds a keyboard with a cancel button for pending payments.
func KbCancelPayment(paymentID string) *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "❌ Hủy giao dịch", CallbackData: fmt.Sprintf("cancel_payment:%s", paymentID)},
			},
			{
				{Text: "🔙 Menu", CallbackData: "back:menu"},
			},
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
