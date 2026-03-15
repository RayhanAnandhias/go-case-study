-- Buat schema terpisah untuk masing-masing service
CREATE SCHEMA IF NOT EXISTS order_schema;
CREATE SCHEMA IF NOT EXISTS inventory_schema;
CREATE SCHEMA IF NOT EXISTS payment_schema;

-- ==========================================
-- Skema untuk Order Service (order_schema)
-- ==========================================
CREATE TABLE order_schema.orders (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    total_amount NUMERIC(10, 2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_schema.order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES order_schema.orders(id),
    product_id UUID NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    price NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ==========================================
-- Skema untuk Inventory Service (inventory_schema)
-- ==========================================
CREATE TABLE inventory_schema.products (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    stock INT NOT NULL CHECK (stock >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE inventory_schema.processed_events (
    event_id UUID PRIMARY KEY,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert dummy data untuk dites
INSERT INTO inventory_schema.products (id, name, price, stock) VALUES
(gen_random_uuid(), 'Laptop', 15000000.00, 100),
(gen_random_uuid(), 'Mouse', 250000.00, 500);

-- ==========================================
-- Skema untuk Payment Service (payment_schema)
-- ==========================================
CREATE TABLE payment_schema.payments (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
