DROP INDEX IF EXISTS uq_payment_webhook_events_provider_event_id;
DROP INDEX IF EXISTS idx_cart_items_cart_id_product_id;
DROP INDEX IF EXISTS idx_payments_provider_provider_reference;
DROP INDEX IF EXISTS idx_payments_order_id_provider;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_products_is_active;

ALTER TABLE payments
    DROP CONSTRAINT IF EXISTS payments_status_valid,
    DROP CONSTRAINT IF EXISTS payments_amount_non_negative;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_status_valid,
    DROP CONSTRAINT IF EXISTS orders_total_amount_non_negative;

ALTER TABLE order_items
    DROP CONSTRAINT IF EXISTS order_items_subtotal_amount_non_negative,
    DROP CONSTRAINT IF EXISTS order_items_price_amount_non_negative,
    DROP CONSTRAINT IF EXISTS order_items_quantity_positive;

ALTER TABLE cart_items
    DROP CONSTRAINT IF EXISTS cart_items_price_snapshot_amount_non_negative,
    DROP CONSTRAINT IF EXISTS cart_items_quantity_positive;

ALTER TABLE products
    DROP CONSTRAINT IF EXISTS products_compare_at_price_gt_price,
    DROP CONSTRAINT IF EXISTS products_price_amount_non_negative;
