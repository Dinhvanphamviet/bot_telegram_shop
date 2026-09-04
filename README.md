# Telegram Shop Bot

Hệ thống Bot bán hàng tự động trên Telegram viết bằng **Go (Golang)**, tích hợp thanh toán VietQR tự động qua **SePay**, cơ sở dữ liệu **Neon PostgreSQL** và triển khai trên **Railway**.

## 🚀 Tính năng chính

- **Bán hàng tự động 24/7**: Mỗi đơn hàng giao 1 link/tài khoản riêng biệt.
- **Chống bán trùng link tuyệt đối**: Sử dụng cơ chế hàng chờ `FOR UPDATE SKIP LOCKED` ở tầng PostgreSQL.
- **Thanh toán VietQR SePay**: Tạo mã QR động kèm nội dung chuyển khoản định danh, tự động kích hoạt đơn hàng khi tiền vào tài khoản ngân hàng.
- **Tích hợp ví tiền (Wallet)**: Cho phép người dùng nạp tiền vào ví và thanh toán tức thì bằng số dư.
- **Tự động hoàn tiền (Refund)**: Nếu kho hàng vừa hết link sau khi khách thanh toán thành công, hệ thống tự động hoàn tiền vào ví người dùng.
- **Hệ thống Quản trị (Admin)**: Quản lý trực tiếp qua lệnh `/admin` trên Telegram và REST API được bảo vệ bằng `X-API-Key`.

## 🛠 Công nghệ

- **Backend**: Go 1.22+
- **HTTP Router**: Chi router
- **Database**: Neon PostgreSQL (pgx pool)
- **Payment**: SePay (VietQR + HMAC-SHA256 Webhook)
- **Deploy**: Docker / Railway

## ⚙️ Cấu hình môi trường (.env)

Tạo file `.env` dựa trên `.env.example`:

```env
DATABASE_URL=postgresql://user:password@host/database?sslmode=require
TELEGRAM_BOT_TOKEN=your_bot_token
WEBHOOK_URL=https://your-app.up.railway.app
PORT=8080
ADMIN_TELEGRAM_IDS=your_telegram_id
SEPAY_BANK_CODE=MBB
SEPAY_ACCOUNT_NUMBER=your_account_number
SEPAY_WEBHOOK_SECRET=your_secret
ADMIN_API_KEY=your_admin_api_key
```

## 📦 Chạy ứng dụng

### Chạy migration database:
```bash
go run ./cmd/migrate
```

### Chạy server:
```bash
go run ./cmd/server
```
