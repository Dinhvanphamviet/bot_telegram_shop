-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- USERS
-- ============================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    telegram_id BIGINT NOT NULL UNIQUE,
    username VARCHAR(255),
    first_name VARCHAR(255),
    balance BIGINT NOT NULL DEFAULT 0, -- đơn vị đồng (VND)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);

-- ============================================
-- PRODUCTS
-- ============================================
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    image_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_active ON products(is_active) WHERE is_active = true;

-- ============================================
-- ITEMS (biến thể/gói của product)
-- ============================================
CREATE TABLE items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price BIGINT NOT NULL, -- đơn vị đồng (VND)
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_items_product_id ON items(product_id);
CREATE INDEX idx_items_active ON items(product_id, is_active) WHERE is_active = true;

-- ============================================
-- ORDERS
-- ============================================
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    item_id UUID NOT NULL REFERENCES items(id),
    product_link_id UUID, -- gán sau khi thanh toán thành công
    amount BIGINT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- PENDING, PAID, CANCELLED, EXPIRED
    payment_method VARCHAR(20) NOT NULL DEFAULT 'QR', -- QR, WALLET
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status) WHERE status = 'PENDING';

-- ============================================
-- PRODUCT_LINKS (kho link)
-- ============================================
CREATE TABLE product_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    link TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE', -- AVAILABLE, SOLD
    assigned_user_id UUID REFERENCES users(id),
    order_id UUID REFERENCES orders(id),
    assigned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_links_available ON product_links(item_id, status) WHERE status = 'AVAILABLE';
CREATE INDEX idx_product_links_order_id ON product_links(order_id) WHERE order_id IS NOT NULL;

-- Add FK from orders to product_links (deferred because of circular reference)
ALTER TABLE orders ADD CONSTRAINT fk_orders_product_link
    FOREIGN KEY (product_link_id) REFERENCES product_links(id);

-- ============================================
-- PAYMENTS (SePay)
-- ============================================
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID REFERENCES orders(id),
    user_id UUID NOT NULL REFERENCES users(id),
    provider VARCHAR(50) NOT NULL DEFAULT 'SEPAY',
    provider_transaction_id VARCHAR(255),
    amount BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- PENDING, SUCCESS, FAILED, EXPIRED
    qr_url TEXT,
    transfer_content VARCHAR(100) NOT NULL UNIQUE, -- mã nội dung chuyển khoản unique
    payment_type VARCHAR(20) NOT NULL DEFAULT 'ORDER', -- ORDER, DEPOSIT
    expired_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_transfer_content ON payments(transfer_content);
CREATE INDEX idx_payments_order_id ON payments(order_id) WHERE order_id IS NOT NULL;
CREATE INDEX idx_payments_status ON payments(status) WHERE status = 'PENDING';

-- ============================================
-- WALLET_TRANSACTIONS
-- ============================================
CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    amount BIGINT NOT NULL, -- positive = tiền vào, negative = tiền ra
    type VARCHAR(20) NOT NULL, -- DEPOSIT, PURCHASE, REFUND
    status VARCHAR(20) NOT NULL DEFAULT 'SUCCESS',
    description TEXT,
    reference_id VARCHAR(255), -- order_id hoặc payment_id
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallet_transactions_user_id ON wallet_transactions(user_id);

-- ============================================
-- Updated_at trigger function
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_items_updated_at BEFORE UPDATE ON items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payments_updated_at BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Migration upgrade safeguard for existing installations
ALTER TABLE orders ADD COLUMN IF NOT EXISTS quantity INT NOT NULL DEFAULT 1;
