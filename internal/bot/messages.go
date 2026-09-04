package bot

import (
	"fmt"
	"strings"
	"time"
)

// FormatMoney formats VND amount with dot separators (50000 → "50.000đ").
func FormatMoney(amount int64) string {
	if amount < 0 {
		return "-" + FormatMoney(-amount)
	}
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return s + "đ"
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, byte(c))
	}
	return string(result) + "đ"
}

// FormatDate formats a time for display.
func FormatDate(t time.Time) string {
	return t.In(time.FixedZone("ICT", 7*3600)).Format("15:04 02/01/2006")
}

// FormatOrderStatus translates order status to Vietnamese.
func FormatOrderStatus(status string) string {
	switch status {
	case "PENDING":
		return "⏳ Chờ thanh toán"
	case "PAID":
		return "✅ Đã thanh toán"
	case "CANCELLED":
		return "❌ Đã hủy"
	case "EXPIRED":
		return "⏰ Hết hạn"
	default:
		return status
	}
}

// FormatPaymentMethod translates payment method.
func FormatPaymentMethod(method string) string {
	switch method {
	case "QR":
		return "💳 QR Chuyển khoản"
	case "WALLET":
		return "💰 Ví"
	default:
		return method
	}
}

// FormatWalletType translates wallet transaction type.
func FormatWalletType(txType string) string {
	switch txType {
	case "DEPOSIT":
		return "➕ Nạp tiền"
	case "PURCHASE":
		return "🛒 Mua hàng"
	case "REFUND":
		return "↩️ Hoàn tiền"
	default:
		return txType
	}
}

// MsgWelcome returns the welcome message.
func MsgWelcome(firstName string) string {
	return fmt.Sprintf(`👋 <b>Xin chào %s!</b>

🛍 Chào mừng bạn đến với <b>Shop</b>!

Chọn một mục bên dưới để bắt đầu:`, firstName)
}

// MsgProductList returns the product listing header.
func MsgProductList() string {
	return `🛒 <b>Danh sách sản phẩm</b>

Chọn sản phẩm bạn muốn xem:`
}

// MsgNoProducts returns message when no products available.
func MsgNoProducts() string {
	return `😔 Hiện tại chưa có sản phẩm nào.

Vui lòng quay lại sau!`
}

// MsgWalletInfo returns wallet info message.
func MsgWalletInfo(balance int64) string {
	return fmt.Sprintf(`💰 <b>Ví của bạn</b>

💵 Số dư: <b>%s</b>

Chọn thao tác:`, FormatMoney(balance))
}

// MsgHelp returns the help message.
func MsgHelp() string {
	return `❓ <b>Hướng dẫn sử dụng</b>

🛒 <b>/products</b> — Xem danh sách sản phẩm
💰 <b>/wallet</b> — Xem ví & nạp tiền
📦 <b>/orders</b> — Xem đơn hàng
📋 <b>/menu</b> — Menu chính
🧹 <b>/clear</b> — Xóa tin nhắn cũ, làm sạch màn hình chat

<b>Cách mua hàng:</b>
1️⃣ Chọn sản phẩm từ danh sách
2️⃣ Chọn số lượng muốn mua
3️⃣ Thanh toán bằng QR hoặc số dư ví
4️⃣ Nhận link sản phẩm ngay lập tức!

<b>Nạp tiền:</b>
Vào /wallet → Nạp tiền (chọn sẵn hoặc tự nhập số khác) → Quét QR → Tiền vào ví tự động

💬 Liên hệ hỗ trợ: @dinhvpviet`
}

// MsgOrderHistory returns order history header.
func MsgOrderHistory() string {
	return `📦 <b>Đơn hàng của bạn</b>
`
}

// MsgNoOrders returns message when no orders.
func MsgNoOrders() string {
	return `📦 Bạn chưa có đơn hàng nào.

Bắt đầu mua sắm tại /products!`
}

// MsgConfirmPurchase returns purchase confirmation message.
func MsgConfirmPurchase(itemName string, price int64, balance int64) string {
	return fmt.Sprintf(`🛒 <b>Xác nhận mua hàng</b>

📦 Sản phẩm: <b>%s</b>
💰 Giá: <b>%s</b>
💵 Số dư ví: <b>%s</b>

Bạn có muốn thanh toán bằng ví?`, itemName, FormatMoney(price), FormatMoney(balance))
}

// MsgPurchaseSuccess returns successful purchase message.
func MsgPurchaseSuccess(itemName, link string) string {
	return fmt.Sprintf(`✅ <b>Mua hàng thành công!</b>

📦 Sản phẩm: <b>%s</b>

🔗 Link của bạn:
<code>%s</code>

⚠️ Lưu ý: Mỗi link chỉ dùng được 1 lần. Vui lòng sao chép và sử dụng ngay.`, itemName, link)
}

// MsgQRPayment returns QR payment instruction message.
func MsgQRPayment(itemName string, amount int64, transferContent string, expireMinutes int) string {
	return fmt.Sprintf(`💳 <b>Thanh toán QR</b>

📦 Sản phẩm: <b>%s</b>
💰 Số tiền: <b>%s</b>
📝 Nội dung CK: <code>%s</code>

⏳ Hết hạn sau: <b>%d phút</b>

📱 Quét mã QR bên trên để thanh toán.

⚠️ <b>Quan trọng:</b> Chuyển <b>đúng số tiền</b> và <b>đúng nội dung</b> để hệ thống tự động xác nhận.`,
		itemName, FormatMoney(amount), transferContent, expireMinutes)
}

// MsgDepositQR returns deposit QR instruction.
func MsgDepositQR(amount int64, transferContent string) string {
	return fmt.Sprintf(`💵 <b>Nạp tiền vào ví</b>

💰 Số tiền: <b>%s</b>
📝 Nội dung CK: <code>%s</code>

📱 Quét mã QR bên trên để nạp tiền.

⚠️ Chuyển <b>đúng số tiền</b> và <b>đúng nội dung</b> để hệ thống tự động xác nhận.
⏳ Hết hạn sau 30 phút.`,
		FormatMoney(amount), transferContent)
}

// MsgPaymentReceived returns the message sent when payment webhook is received.
func MsgPaymentReceived(itemName, link string) string {
	return fmt.Sprintf(`✅ <b>Thanh toán thành công!</b>

📦 Sản phẩm: <b>%s</b>

🔗 Link của bạn:
<code>%s</code>

⚠️ Mỗi link chỉ dùng được 1 lần.`, itemName, link)
}

// MsgDepositReceived returns the message sent when deposit is confirmed.
func MsgDepositReceived(amount, newBalance int64) string {
	return fmt.Sprintf(`✅ <b>Nạp tiền thành công!</b>

💵 Đã nạp: <b>%s</b>
💰 Số dư mới: <b>%s</b>`,
		FormatMoney(amount), FormatMoney(newBalance))
}

// MsgRefundNotice returns a refund notification.
func MsgRefundNotice(amount int64, reason string) string {
	return fmt.Sprintf(`↩️ <b>Hoàn tiền</b>

💵 Đã hoàn: <b>%s</b> vào ví
📝 Lý do: %s

Xin lỗi vì sự bất tiện!`, FormatMoney(amount), reason)
}

// MsgOutOfStock returns out of stock indicator.
func MsgOutOfStock() string {
	return "❌ Hết hàng"
}

// MsgInStock returns in stock indicator.
func MsgInStock(count int) string {
	return fmt.Sprintf("✅ Còn %d", count)
}

// MsgError returns a generic error message.
func MsgError() string {
	return "❌ Đã xảy ra lỗi. Vui lòng thử lại sau."
}

// MsgPaymentExpired returns message when payment expires.
func MsgPaymentExpired(content string) string {
	return fmt.Sprintf(`⏰ <b>Yêu cầu thanh toán đã hết hạn!</b>

Mã giao dịch: <code>%s</code>
Đã quá thời gian chờ thanh toán nên hệ thống đã tự động hủy yêu cầu này.

💡 <i>Nếu bạn vẫn muốn nạp tiền hoặc mua hàng, vui lòng tạo mã mới từ Menu nhé!</i>`, content)
}

// MsgItemDetail returns product details formatted matching reference UI screenshot.
func MsgItemDetail(productName, itemName, description string, shortID string, price, balance int64, stock int) string {
	var sb strings.Builder
	sb.WriteString("🛍 <b>THÔNG TIN SẢN PHẨM</b>\n")
	sb.WriteString("───────────────────\n")
	if productName != "" && productName != itemName {
		sb.WriteString(fmt.Sprintf("📦 <b>%s — %s</b>\n\n", productName, itemName))
	} else {
		sb.WriteString(fmt.Sprintf("📦 <b>%s</b>\n\n", itemName))
	}

	if description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", description))
	}

	if shortID != "" {
		sb.WriteString(fmt.Sprintf("🆔 Mã SP: <code>#%s</code>\n", shortID))
	}
	sb.WriteString(fmt.Sprintf("💵 Giá bán: <b>%s</b>\n", FormatMoney(price)))

	if stock > 0 {
		sb.WriteString(fmt.Sprintf("📦 Tồn kho: <b>%d</b>\n", stock))
	} else {
		sb.WriteString("📦 Tồn kho: 🔴 <b>Hết hàng</b>\n")
	}

	sb.WriteString(fmt.Sprintf("💰 Số dư ví: <b>%s</b>\n", FormatMoney(balance)))

	if stock == 0 {
		sb.WriteString("\n⚠️ <i>Sản phẩm này hiện đang tạm hết hàng, bạn vui lòng quay lại sau nhé!</i>")
	} else if balance < price {
		sb.WriteString("⚠️ <i>Ví chưa đủ để mua 1 sản phẩm.</i>\n\n👇 <b>Chọn số lượng muốn mua bên dưới:</b>")
	} else {
		canBuy := balance / price
		sb.WriteString(fmt.Sprintf("✅ <i>Số dư ví đủ để mua <b>%d</b> sản phẩm.</i>\n\n👇 <b>Chọn số lượng muốn mua bên dưới:</b>", canBuy))
	}

	return sb.String()
}

// MsgConfirmOrder returns the order confirmation screen before payment.
func MsgConfirmOrder(itemName string, quantity int, unitPrice, totalAmount, balance int64) string {
	var walletNotice string
	if balance >= totalAmount {
		walletNotice = "✅ <i>Ví đủ số dư để thanh toán ngay lập tức.</i>"
	} else {
		deficit := totalAmount - balance
		walletNotice = fmt.Sprintf("⚠️ <i>Ví còn thiếu <b>%s</b>. Bạn có thể thanh toán trực tiếp qua SePay QR hoặc nạp thêm tiền.</i>", FormatMoney(deficit))
	}

	return fmt.Sprintf(`🛒 <b>XÁC NHẬN ĐƠN HÀNG</b>
───────────────────
📦 <b>Sản phẩm:</b> %s
🔢 <b>Số lượng:</b> <b>%d</b>
💵 <b>Đơn giá:</b> %s
💰 <b>Tổng thanh toán:</b> <b>%s</b>
───────────────────
💳 <b>Số dư ví hiện tại:</b> %s
%s`, itemName, quantity, FormatMoney(unitPrice), FormatMoney(totalAmount), FormatMoney(balance), walletNotice)
}

// MsgPurchaseSuccessMulti returns successful purchase message with multiple links.
func MsgPurchaseSuccessMulti(itemName string, quantity int, totalAmount int64, links []string) string {
	var sb strings.Builder
	sb.WriteString("🎉 <b>MUA HÀNG THÀNH CÔNG!</b>\n")
	sb.WriteString("───────────────────\n")
	sb.WriteString(fmt.Sprintf("📦 <b>Sản phẩm:</b> %s\n", itemName))
	sb.WriteString(fmt.Sprintf("🔢 <b>Số lượng:</b> %d\n", quantity))
	sb.WriteString(fmt.Sprintf("💰 <b>Tổng tiền:</b> %s\n", FormatMoney(totalAmount)))
	sb.WriteString("───────────────────\n")
	sb.WriteString("🔑 <b>THÔNG TIN SẢN PHẨM / LINK NHẬN HÀNG:</b>\n\n")

	for i, link := range links {
		sb.WriteString(fmt.Sprintf("%d. <code>%s</code>\n", i+1, link))
	}

	sb.WriteString("\n⚠️ <i>Lưu ý: Mỗi link chỉ dùng được 1 lần. Bạn có thể xem lại trong /orders bất cứ lúc nào!</i>")
	return sb.String()
}

// MsgPromptCustomQuantity prompts the user to type custom quantity.
func MsgPromptCustomQuantity(itemName string, stock int) string {
	return fmt.Sprintf(`✏️ <b>Vui lòng nhập số lượng bạn muốn mua:</b>

📦 <i>Sản phẩm: %s</i>
📊 <i>(Tối thiểu: 1 — Tối đa hiện có: %d)</i>

👇 Hãy gửi tin nhắn số lượng bạn muốn mua vào đây:`, itemName, stock)
}

// MsgPromptCustomDeposit prompts the user to type custom deposit amount.
func MsgPromptCustomDeposit() string {
	return `✏️ <b>Vui lòng nhập số tiền bạn muốn nạp (VNĐ):</b>

<i>Ví dụ: 75000, 150.000, 50k, 100k, hoặc 1tr</i>
<i>(Tối thiểu: 10.000đ — Tối đa: 50.000.000đ)</i>

👇 Hãy gửi tin nhắn số tiền bạn muốn nạp vào đây:`
}

// MsgQRPaymentOrder returns QR payment instruction message for an order with quantity.
func MsgQRPaymentOrder(itemName string, quantity int, amount int64, transferContent string, expireMinutes int) string {
	return fmt.Sprintf(`💳 <b>Thanh toán đơn hàng qua QR</b>

📦 <b>Sản phẩm:</b> %s (x%d)
💰 <b>Tổng số tiền:</b> <b>%s</b>
📝 <b>Nội dung CK:</b> <code>%s</code>

⏳ Hết hạn sau: <b>%d phút</b>

📱 Quét mã QR bên trên để thanh toán.
⚠️ <b>Quan trọng:</b> Chuyển <b>đúng số tiền</b> và <b>đúng nội dung</b> để hệ thống tự động xác nhận và trả hàng ngay lập tức!`,
		itemName, quantity, FormatMoney(amount), transferContent, expireMinutes)
}


