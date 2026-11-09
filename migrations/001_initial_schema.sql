CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    keycloak_id VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(32),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('customer', 'driver', 'vendor', 'admin')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stores (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(160) UNIQUE NOT NULL,
    description TEXT,
    logo_url TEXT,
    cover_url TEXT,
    location GEOMETRY(Point, 4326) NOT NULL,
    address_line TEXT NOT NULL,
    commission_rate NUMERIC(5, 2) NOT NULL DEFAULT 10.00,
    is_open BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stores_location ON stores USING GIST(location);
CREATE INDEX IF NOT EXISTS idx_stores_owner ON stores(owner_id);

CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) UNIQUE NOT NULL,
    image_url TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    base_price NUMERIC(12, 2) NOT NULL CHECK (base_price >= 0),
    is_available BOOLEAN NOT NULL DEFAULT true,
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_store ON products(store_id);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_id);

DO $$ BEGIN
    CREATE TYPE order_status AS ENUM (
        'placed', 
        'accepted_by_store', 
        'in_kitchen',
        'preparing', 
        'ready_for_pickup', 
        'driver_assigned', 
        'driver_in_transit',
        'in_transit', 
        'delivered', 
        'cancelled'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

ALTER TYPE order_status ADD VALUE IF NOT EXISTS 'in_kitchen';
ALTER TYPE order_status ADD VALUE IF NOT EXISTS 'driver_in_transit';

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_number SERIAL,
    customer_id UUID NOT NULL REFERENCES users(id),
    store_id UUID NOT NULL REFERENCES stores(id),
    driver_id UUID REFERENCES users(id),
    status order_status NOT NULL DEFAULT 'placed',
    subtotal NUMERIC(12, 2) NOT NULL CHECK (subtotal >= 0),
    delivery_fee NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (delivery_fee >= 0),
    service_fee NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (service_fee >= 0),
    tip_usd NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (tip_usd >= 0),
    commission_fee NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (commission_fee >= 0),
    total NUMERIC(12, 2) NOT NULL CHECK (total >= 0),
    total_usd NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (total_usd >= 0),
    bs_rate NUMERIC(12, 4) NOT NULL DEFAULT 0.0000 CHECK (bs_rate >= 0),
    total_bs NUMERIC(14, 2) NOT NULL DEFAULT 0.00 CHECK (total_bs >= 0),
    delivery_location GEOMETRY(Point, 4326) NOT NULL,
    delivery_address TEXT NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    payment_status VARCHAR(30) NOT NULL DEFAULT 'pending',
    placed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ
);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS tip_usd NUMERIC(12, 2) NOT NULL DEFAULT 0.00;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS commission_fee NUMERIC(12, 2) NOT NULL DEFAULT 0.00;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS total_usd NUMERIC(12, 2) NOT NULL DEFAULT 0.00;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS bs_rate NUMERIC(12, 4) NOT NULL DEFAULT 0.0000;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS total_bs NUMERIC(14, 2) NOT NULL DEFAULT 0.00;

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_customer ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_store ON orders(store_id);
CREATE INDEX IF NOT EXISTS idx_orders_driver ON orders(driver_id);

CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(12, 2) NOT NULL CHECK (unit_price >= 0),
    subtotal NUMERIC(12, 2) NOT NULL CHECK (subtotal >= 0),
    options JSONB DEFAULT '{}'
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) DEFAULT '';

CREATE TABLE IF NOT EXISTS payment_receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    payment_method VARCHAR(50) NOT NULL,
    reference_number VARCHAR(100) NOT NULL,
    issuing_bank VARCHAR(100),
    receiving_bank VARCHAR(100),
    amount_usd NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    amount_bs NUMERIC(14, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    verified_at TIMESTAMPTZ,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_receipts_order ON payment_receipts(order_id);
CREATE INDEX IF NOT EXISTS idx_payment_receipts_reference ON payment_receipts(reference_number);

CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
    driver_id UUID REFERENCES users(id) ON DELETE SET NULL,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reviews_order ON reviews(order_id);
CREATE INDEX IF NOT EXISTS idx_reviews_store ON reviews(store_id);
CREATE INDEX IF NOT EXISTS idx_reviews_driver ON reviews(driver_id);

DO $$ BEGIN
    CREATE TYPE ride_status AS ENUM (
        'requested',
        'driver_assigned',
        'in_trip',
        'completed',
        'cancelled'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS rides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID NOT NULL REFERENCES users(id),
    driver_id UUID REFERENCES users(id),
    vehicle_type VARCHAR(30) NOT NULL DEFAULT 'standard',
    status ride_status NOT NULL DEFAULT 'requested',
    pickup_location GEOMETRY(Point, 4326) NOT NULL,
    pickup_address TEXT NOT NULL,
    dropoff_location GEOMETRY(Point, 4326) NOT NULL,
    dropoff_address TEXT NOT NULL,
    distance_km NUMERIC(8, 2) NOT NULL DEFAULT 0.00,
    fare_usd NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    bs_rate NUMERIC(12, 4) NOT NULL DEFAULT 0.0000,
    fare_bs NUMERIC(14, 2) NOT NULL DEFAULT 0.00,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'cash',
    payment_status VARCHAR(30) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_rides_status ON rides(status);
CREATE INDEX IF NOT EXISTS idx_rides_customer ON rides(customer_id);
CREATE INDEX IF NOT EXISTS idx_rides_driver ON rides(driver_id);

-- Seed System Vendors & Default Demo Users
INSERT INTO users (id, keycloak_id, email, phone, first_name, last_name, role)
VALUES 
    ('11111111-1111-1111-1111-111111111111', 'usr_vendor_burger', 'vendor@burgercrave.com', '+584121000001', 'Chef', 'Burger', 'vendor'),
    ('22222222-2222-2222-2222-222222222222', 'usr_vendor_sushi', 'vendor@sushiomakase.com', '+584121000002', 'Kenji', 'Sato', 'vendor'),
    ('33333333-3333-3333-3333-333333333333', 'usr_driver_carlos', 'driver@traigalo.com', '+584121000003', 'Carlos', 'Mendoza', 'driver'),
    ('44444444-4444-4444-4444-444444444444', 'usr_customer_demo', 'cliente@traigalo.com', '+584121000004', 'Freddy', 'Campos', 'customer')
ON CONFLICT (keycloak_id) DO NOTHING;
