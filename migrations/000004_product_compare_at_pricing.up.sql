ALTER TABLE products
    ADD COLUMN compare_at_price_amount BIGINT NULL,
    ADD COLUMN discount_start_at TIMESTAMPTZ NULL,
    ADD COLUMN discount_end_at TIMESTAMPTZ NULL,
    ADD COLUMN is_discount_active BOOLEAN NOT NULL DEFAULT FALSE;
