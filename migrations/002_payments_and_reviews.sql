CREATE TABLE IF NOT EXISTS payment_receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    payment_method VARCHAR(50) NOT NULL,
    reference_number VARCHAR(100),
    bank_origin VARCHAR(100),
    bank_destination VARCHAR(100),
    amount_usd NUMERIC(12, 2) NOT NULL CHECK (amount_usd >= 0),
    amount_bs NUMERIC(14, 2) NOT NULL CHECK (amount_bs >= 0),
    bs_rate NUMERIC(12, 4) NOT NULL CHECK (bs_rate >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'verified', 'rejected')),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_payment_receipts_order ON payment_receipts(order_id);
CREATE INDEX IF NOT EXISTS idx_payment_receipts_status ON payment_receipts(status);

CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
    driver_id UUID REFERENCES users(id) ON DELETE SET NULL,
    store_rating INT CHECK (store_rating IS NULL OR (store_rating >= 1 AND store_rating <= 5)),
    driver_rating INT CHECK (driver_rating IS NULL OR (driver_rating >= 1 AND driver_rating <= 5)),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reviews_order ON reviews(order_id);
CREATE INDEX IF NOT EXISTS idx_reviews_customer ON reviews(customer_id);
CREATE INDEX IF NOT EXISTS idx_reviews_store ON reviews(store_id);
CREATE INDEX IF NOT EXISTS idx_reviews_driver ON reviews(driver_id);
