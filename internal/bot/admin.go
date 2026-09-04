package bot

import (
	"context"
	"fmt"
)

// handleAdmin handles the /admin command.
func (h *CommandHandler) handleAdmin(_ context.Context, msg *Message) {
	if msg.From == nil || !h.cfg.IsAdmin(msg.From.ID) {
		h.bot.SendMessage(msg.Chat.ID, "⛔ <b>Bạn không có quyền truy cập.</b>", nil)
		return
	}
	h.bot.SendMessage(msg.Chat.ID, "🔧 <b>Admin Panel</b>\n\nChọn chức năng:", KbAdminMenu())
}

// handleAdminCallback routes admin-specific callbacks.
func (h *CallbackHandler) handleAdminCallback(ctx context.Context, chatID, msgID int64, action string) {
	switch action {
	case "products":
		products, err := h.productService.GetAllProducts(ctx)
		if err != nil {
			h.bot.EditMessageText(chatID, msgID, MsgError(), KbAdminMenu())
			return
		}
		text := "📦 <b>Sản phẩm (Admin)</b>\n"
		for _, p := range products {
			status := "✅"
			if !p.IsActive {
				status = "❌"
			}
			text += fmt.Sprintf("\n%s %s (ID: <code>%s</code>)", status, p.Name, p.ID.String()[:8])
		}
		if len(products) == 0 {
			text += "\nChưa có sản phẩm. Dùng API để thêm."
		}
		text += "\n\n💡 Dùng REST API để thêm/sửa/xóa sản phẩm."
		h.bot.EditMessageText(chatID, msgID, text, KbAdminMenu())

	case "add_links":
		text := `🔗 <b>Thêm Link sản phẩm</b>

Dùng REST API để thêm link:

<code>POST /api/admin/product-links</code>

Body:
<code>{
  "item_id": "uuid",
  "links": ["link1", "link2", "link3"]
}</code>`
		h.bot.EditMessageText(chatID, msgID, text, KbAdminMenu())

	case "users":
		text := `👤 <b>Quản lý Users</b>

Dùng REST API để nạp tiền (Topup):
<code>POST /api/admin/users/{user_id}/topup</code>

Body:
<code>{
  "amount": 50000,
  "note": "Nạp thủ công"
}</code>`
		h.bot.EditMessageText(chatID, msgID, text, KbAdminMenu())

	case "stats":
		text := "📊 <b>Thống kê</b>\n\n💡 Dùng REST API để xem chi tiết danh sách đơn hàng và người dùng."
		h.bot.EditMessageText(chatID, msgID, text, KbAdminMenu())

	default:
		h.bot.EditMessageText(chatID, msgID, "🔧 Chức năng đang phát triển.", KbAdminMenu())
	}
}
