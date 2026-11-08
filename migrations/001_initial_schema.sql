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
        'preparing', 
        'ready_for_pickup', 
        'driver_assigned', 
        'in_transit', 
        'delivered', 
        'cancelled'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

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

CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);

-- Seed System Vendors & Default Demo Users
INSERT INTO users (id, keycloak_id, email, phone, first_name, last_name, role)
VALUES 
    ('11111111-1111-1111-1111-111111111111', 'usr_vendor_burger', 'vendor@burgercrave.com', '+584121000001', 'Chef', 'Burger', 'vendor'),
    ('22222222-2222-2222-2222-222222222222', 'usr_vendor_sushi', 'vendor@sushiomakase.com', '+584121000002', 'Kenji', 'Sato', 'vendor'),
    ('33333333-3333-3333-3333-333333333333', 'usr_driver_carlos', 'driver@traigalo.com', '+584121000003', 'Carlos', 'Mendoza', 'driver'),
    ('44444444-4444-4444-4444-444444444444', 'usr_customer_demo', 'cliente@traigalo.com', '+584121000004', 'Freddy', 'Campos', 'customer')
ON CONFLICT (keycloak_id) DO NOTHING;

-- Seed Initial Stores & Menu
INSERT INTO stores (id, owner_id, name, slug, description, location, address_line, commission_rate, is_open)
VALUES 
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'Burger Crave', 'burger-crave', 'Smash burgers gourmet y complementos', ST_SetSRID(ST_MakePoint(-66.8620, 10.4910), 4326), 'Av. Principal Las Mercedes, Caracas', 12.00, true),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'Sushi Omakase Roll', 'sushi-omakase', 'Rolls japoneses y bowls frescos', ST_SetSRID(ST_MakePoint(-66.8550, 10.4930), 4326), 'Calle Madrid, Las Mercedes, Caracas', 15.00, true)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO products (id, store_id, name, description, base_price, is_available)
VALUES 
    ('55555555-5555-5555-5555-555555555555', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Crispy Supreme Smash', 'Doble carne smash, cebolla caramelizada, cheddar fundido y salsa secreta', 14.50, true),
    ('66666666-6666-6666-6666-666666666666', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Smoked Bacon BBQ Melt', 'Carne madurada, doble tocineta ahumada, BBQ artesanal', 15.00, true),
    ('77777777-7777-7777-7777-777777777777', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Papas Rústicas Trufadas', 'Papas rústicas con aceite de trufa y queso parmesano gratinado', 5.50, true),
    ('88888888-8888-8888-8888-888888888888', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Combo Tokio Special 16 pcs', 'Variedad de rolls tempura, salmón fresco y atún', 32.00, true)
ON CONFLICT (id) DO NOTHING;
