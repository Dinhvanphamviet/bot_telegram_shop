-- Drop triggers
DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;
DROP TRIGGER IF EXISTS update_items_updated_at ON items;
DROP TRIGGER IF EXISTS update_products_updated_at ON products;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (respect FK dependencies)
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS payments;

-- Remove FK from orders before dropping product_links
ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_product_link;
DROP TABLE IF EXISTS product_links;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;

-- Drop extension
DROP EXTENSION IF EXISTS "uuid-ossp";
