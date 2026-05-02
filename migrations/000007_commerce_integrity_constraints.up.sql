ALTER TABLE products
    ADD CONSTRAINT products_price_amount_non_negative CHECK (price_amount >= 0),
    ADD CONSTRAINT products_compare_at_price_gt_price CHECK (compare_at_price_amount IS NULL OR compare_at_price_amount > price_amount);

ALTER TABLE cart_items
    ADD CONSTRAINT cart_items_quantity_positive CHECK (quantity > 0),
    ADD CONSTRAINT cart_items_price_snapshot_amount_non_negative CHECK (price_snapshot_amount >= 0);

ALTER TABLE order_items
    ADD CONSTRAINT order_items_quantity_positive CHECK (quantity > 0),
    ADD CONSTRAINT order_items_price_amount_non_negative CHECK (price_amount >= 0),
    ADD CONSTRAINT order_items_subtotal_amount_non_negative CHECK (subtotal_amount >= 0);

ALTER TABLE orders
    ADD CONSTRAINT orders_total_amount_non_negative CHECK (total_amount >= 0),
    ADD CONSTRAINT orders_status_valid CHECK (status IN (
        'PENDING_PAYMENT',
        'PAID',
        'PROCESSING',
        'READY_TO_SHIP',
        'SHIPPED',
        'COMPLETED',
        'CANCELLED',
        'EXPIRED',
        'FAILED'
    ));

ALTER TABLE payments
    ADD CONSTRAINT payments_amount_non_negative CHECK (amount >= 0),
    ADD CONSTRAINT payments_status_valid CHECK (status IN (
        'PENDING',
        'PAID',
        'EXPIRED',
        'FAILED',
        'CANCELLED',
        'REFUNDED'
    ));

CREATE INDEX IF NOT EXISTS idx_products_is_active ON products (is_active);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_payments_order_id_provider ON payments (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_provider_provider_reference ON payments (provider, provider_reference);
CREATE INDEX IF NOT EXISTS idx_cart_items_cart_id_product_id ON cart_items (cart_id, product_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_webhook_events_provider_event_id ON payment_webhook_events (provider, event_id);
