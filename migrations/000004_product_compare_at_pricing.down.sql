ALTER TABLE products
    DROP COLUMN IF EXISTS is_discount_active,
    DROP COLUMN IF EXISTS discount_end_at,
    DROP COLUMN IF EXISTS discount_start_at,
    DROP COLUMN IF EXISTS compare_at_price_amount;
