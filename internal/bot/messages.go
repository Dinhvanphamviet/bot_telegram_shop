package bot

import (
	"fmt"
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

<b>Cách mua hàng:</b>
1️⃣ Chọn sản phẩm từ danh sách
2️⃣ Chọn gói bạn muốn
3️⃣ Thanh toán bằng QR hoặc số dư ví
4️⃣ Nhận link sản phẩm ngay lập tức!

<b>Nạp tiền:</b>
Vào /wallet → Nạp tiền → Quét QR → Tiền vào ví tự động

💬 Liên hệ hỗ trợ: @admin`
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
	return "🔴 Hết hàng"
}

// MsgInStock returns in stock indicator.
func MsgInStock(count int) string {
	return fmt.Sprintf("🟢 Còn %d", count)
}

// MsgError returns a generic error message.
func MsgError() string {
	return "❌ Đã xảy ra lỗi. Vui lòng thử lại sau."
}
