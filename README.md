# Go E-Commerce Modular Monolith (MVP Backend)

A modular monolith backend for a standard e-commerce platform. It provides:
- **Storefront APIs** for customers (register/login, browse products, cart, checkout, payments).
- **Admin Backoffice APIs** for catalog management, orders, customer visibility, reports, and audit logs.
- **Payment Gateway Adapter Layer** via a provider abstraction (Midtrans/Xendit implementations).

---

## 1) Project Overview

This project is an MVP commerce backend that supports end-to-end ordering flow:
1. Customer registers and logs in.
2. Admin logs in and manages category/product catalog.
3. Customer adds items to cart and checks out.
4. Customer creates payment for an order.
5. Payment webhook updates payment and order status.

It is intentionally kept practical for MVP delivery while preserving clean module boundaries.

---

## 2) Architecture Summary

### Modular monolith
- Single deployable Go service.
- Domain modules under `internal/*` (auth, customer, category, product, cart, order, payment, report, audit, etc.).
- Repository → Service → Handler layering where applicable.

### User storefront API
- Public endpoints: product/category listing.
- Authenticated customer endpoints: profile, address, cart, checkout, payment, order history.

### Admin backoffice API
- Admin-authenticated endpoints for:
  - category/product CRUD
  - order list/detail/status update
  - customer visibility
  - reports and audit logs

### Payment gateway adapter
- Uses `PaymentProvider` abstraction.
- Provider selected via `PAYMENT_PROVIDER` environment variable.
- Midtrans and Xendit providers exposed through webhook endpoints.

---

## 3) Tech Stack

- **Language:** Go (1.22+)
- **Web:** Gin
- **ORM/DB:** GORM + PostgreSQL
- **Auth:** JWT
- **Password Hashing:** bcrypt
- **Logging:** Zap
- **Container/Local DB:** Docker Compose

---

## 4) Prerequisites

Make sure these are installed:
- Go 1.22+
- Docker + Docker Compose plugin
- `curl`
- Optional but recommended: `jq`

On Debian/Ubuntu (optional):

```bash
sudo apt-get update
sudo apt-get install -y jq curl
```

---

## 5) Environment Variables

1. Copy sample environment:

```bash
cp .env.example .env
```

2. Review/edit `.env` values (typical fields):
- `APP_ENV`
- `APP_PORT`
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
- `JWT_SECRET`
- `PAYMENT_PROVIDER` (`midtrans` or `xendit`)
- Provider keys/secrets if needed by your integration

> Do not commit real secrets. Keep secrets only in local `.env` or secret manager.

---

## 6) Run PostgreSQL with Docker Compose

Start PostgreSQL:

```bash
docker compose -f deployments/docker-compose.yml up -d postgres
```

Check status:

```bash
docker compose -f deployments/docker-compose.yml ps
```

Stop services:

```bash
docker compose -f deployments/docker-compose.yml down
```

---

## 7) Run Migrations

SQL migrations are located in `migrations/`.

Run in this order for empty DB:
1. `000001_init.up.sql`
2. `000003_ecommerce_schema.up.sql`
3. `000002_seed_admin.up.sql`

If you use `psql` directly (example):

```bash
export PGPASSWORD='postgres'
psql -h localhost -p 5432 -U postgres -d commerce -f migrations/000001_init.up.sql
psql -h localhost -p 5432 -U postgres -d commerce -f migrations/000003_ecommerce_schema.up.sql
psql -h localhost -p 5432 -U postgres -d commerce -f migrations/000002_seed_admin.up.sql
```

> Adjust host/port/user/db/password to your `.env`.

---

## 8) Start API

```bash
go run ./cmd/api
```

Default local base URL used in examples:

```bash
export BASE_URL="http://localhost:8080"
export API_V1="$BASE_URL/api/v1"
```

---

## 9) Default Seeded Admin

After running seed migration, default admin account is:
- **Email:** `admin@example.com`
- **Password:** `admin12345`
- **Role:** `SUPER_ADMIN`

Use this only for local development. Change credentials for any non-local environment.

---

## 10) Full End-to-End Curl Flow (Copy-Paste Friendly)

> Tips:
> - Commands below use `jq` for easy token/ID extraction.
> - If you do not use `jq`, copy token/IDs manually from JSON response and set env vars yourself.

### 10.1 Health check

```bash
curl -s "$BASE_URL/healthz" | jq
```

### 10.2 Register customer

```bash
curl -s -X POST "$API_V1/auth/customer/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Sena Arya",
    "email": "sena.arya@example.com",
    "phone": "+6281234567890",
    "password": "Secret123!"
  }' | jq
```

### 10.3 Login customer and export token

```bash
CUSTOMER_LOGIN_RESPONSE=$(curl -s -X POST "$API_V1/auth/customer/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "sena.arya@example.com",
    "password": "Secret123!"
  }')

echo "$CUSTOMER_LOGIN_RESPONSE" | jq

# With jq:
export CUSTOMER_TOKEN=$(echo "$CUSTOMER_LOGIN_RESPONSE" | jq -r '.data.token')

# Manual alternative (without jq):
# 1) Copy token from response JSON
# 2) export CUSTOMER_TOKEN='<paste_customer_token_here>'
```

### 10.4 Login admin and export token

```bash
ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "$API_V1/auth/admin/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin12345"
  }')

echo "$ADMIN_LOGIN_RESPONSE" | jq

export ADMIN_TOKEN=$(echo "$ADMIN_LOGIN_RESPONSE" | jq -r '.data.token')

# Manual alternative:
# export ADMIN_TOKEN='<paste_admin_token_here>'
```

### 10.5 Create category

```bash
CREATE_CATEGORY_RESPONSE=$(curl -s -X POST "$API_V1/admin/categories" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Audio",
    "slug": "audio"
  }')

echo "$CREATE_CATEGORY_RESPONSE" | jq

export CATEGORY_ID=$(echo "$CREATE_CATEGORY_RESPONSE" | jq -r '.data.id')
```

### 10.6 Create product

```bash
CREATE_PRODUCT_RESPONSE=$(curl -s -X POST "$API_V1/admin/products" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"category_id\": \"$CATEGORY_ID\",
    \"name\": \"Bluetooth Speaker X1\",
    \"slug\": \"bluetooth-speaker-x1\",
    \"description\": \"Portable 20W speaker with Bluetooth 5.3\",
    \"price_amount\": 250000,
    \"compare_at_price_amount\": 350000,
    \"is_discount_active\": true,
    \"discount_start_at\": \"2026-05-01T00:00:00+07:00\",
    \"discount_end_at\": \"2026-05-31T23:59:59+07:00\",
    \"stock\": 50,
    \"images\": [
      {
        \"image_url\": \"https://images.example.com/products/bluetooth-speaker-x1-main.jpg\",
        \"is_primary\": true
      }
    ]
  }")

echo "$CREATE_PRODUCT_RESPONSE" | jq

export PRODUCT_ID=$(echo "$CREATE_PRODUCT_RESPONSE" | jq -r '.data.id')
```

### 10.7 List products

```bash
curl -s "$API_V1/products?page=1&limit=10&search=speaker&category_slug=audio&sort_by=created_at&sort_order=desc" | jq
```

### 10.8 Add product to cart

```bash
ADD_TO_CART_RESPONSE=$(curl -s -X POST "$API_V1/cart/items" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"product_id\": \"$PRODUCT_ID\",
    \"quantity\": 2
  }")

echo "$ADD_TO_CART_RESPONSE" | jq
```

### 10.9 View cart

```bash
curl -s -X GET "$API_V1/cart" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

### 10.10 Checkout

> You need a customer address ID. Create/get address first if needed.

Create an address:

```bash
CREATE_ADDRESS_RESPONSE=$(curl -s -X POST "$API_V1/customers/me/addresses" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "receiver_name": "Sena Arya",
    "phone": "+6281234567890",
    "address": "Jl. Merdeka No. 10",
    "city": "Bandung",
    "province": "Jawa Barat",
    "postal_code": "40123",
    "is_default": true
  }')

echo "$CREATE_ADDRESS_RESPONSE" | jq

export ADDRESS_ID=$(echo "$CREATE_ADDRESS_RESPONSE" | jq -r '.data.id')
```

Checkout active cart:

```bash
CHECKOUT_RESPONSE=$(curl -s -X POST "$API_V1/orders/checkout" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"address_id\": \"$ADDRESS_ID\",
    \"notes\": \"Please pack with extra bubble wrap\"
  }")

echo "$CHECKOUT_RESPONSE" | jq

export ORDER_ID=$(echo "$CHECKOUT_RESPONSE" | jq -r '.data.id')
```

### 10.11 Create payment

```bash
CREATE_PAYMENT_RESPONSE=$(curl -s -X POST "$API_V1/payments/orders/$ORDER_ID/pay" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN")

echo "$CREATE_PAYMENT_RESPONSE" | jq

export PAYMENT_ID=$(echo "$CREATE_PAYMENT_RESPONSE" | jq -r '.data.id')
export PROVIDER_REFERENCE=$(echo "$CREATE_PAYMENT_RESPONSE" | jq -r '.data.provider_reference')
```

### 10.12 Simulate Midtrans paid webhook

```bash
curl -s -X POST "$API_V1/webhooks/payments/midtrans" \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\": \"$ORDER_ID\",
    \"transaction_status\": \"settlement\",
    \"fraud_status\": \"accept\",
    \"status_code\": \"200\",
    \"transaction_id\": \"${PROVIDER_REFERENCE:-midtrans-tx-demo-001}\"
  }" | jq
```

### 10.13 View customer order

```bash
curl -s -X GET "$API_V1/orders/$ORDER_ID" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

### 10.14 View admin order

```bash
curl -s -X GET "$API_V1/admin/orders/$ORDER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

### 10.15 Update order status to PROCESSING

```bash
curl -s -X PATCH "$API_V1/admin/orders/$ORDER_ID/status" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "PROCESSING"
  }' | jq
```

### 10.16 View reports

```bash
# Orders report
curl -s "$API_V1/admin/reports/orders?date_from=2026-01-01&date_to=2026-12-31" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq

# Sales report
curl -s "$API_V1/admin/reports/sales?date_from=2026-01-01&date_to=2026-12-31" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq

# Products report
curl -s "$API_V1/admin/reports/products?date_from=2026-01-01&date_to=2026-12-31" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq

# Payments report
curl -s "$API_V1/admin/reports/payments?date_from=2026-01-01&date_to=2026-12-31" \
  -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

---

## Troubleshooting

### App cannot connect to DB
- Ensure PostgreSQL container is running.
- Verify `.env` DB host/port/user/password/db name.
- Confirm migrations already ran successfully.

### `401 Unauthorized`
- Token missing/expired/invalid.
- Ensure you pass `Authorization: Bearer <token>`.
- Ensure admin endpoint uses admin token, not customer token.

### `403 Forbidden`
- Role mismatch (customer token used for admin endpoint).

### Checkout fails with stock error
- Product stock may be insufficient.
- Update stock from admin endpoint, then retry cart/checkout flow.

### Webhook does not update payment/order
- Ensure webhook payload matches selected provider format.
- Ensure provider reference/order ID maps to existing payment record.
- Ensure `PAYMENT_PROVIDER` is configured as expected.

---

## Common Errors

- **Validation error**: missing required JSON field (e.g., `product_id`, `quantity`, `address_id`).
- **Duplicate slug/email**: unique DB constraint hit.
- **Inactive product**: cannot be added to cart or checked out.
- **Empty cart**: checkout blocked until cart has items.

---

## Reset Local Database

> This deletes local DB data.

```bash
docker compose -f deployments/docker-compose.yml down -v

docker compose -f deployments/docker-compose.yml up -d postgres

# Re-run migrations after DB reset
# (example shown in migration section)
```

If you also want to remove app container images created locally:

```bash
docker compose -f deployments/docker-compose.yml down --rmi local -v
```

---

## Run Tests

```bash
go test ./...
```

Optional verbose run:

```bash
go test -v ./...
```

---

## 15) Database Defensive Constraints (Integrity Guardrails)

The database enforces additional **CHECK constraints** and **indexes** to prevent invalid commerce data, even if application-level validation is bypassed.

### Enforced checks
- `products`
  - `stock >= 0`
  - `price_amount >= 0`
  - `compare_at_price_amount IS NULL OR compare_at_price_amount > price_amount`
- `cart_items`
  - `quantity > 0`
  - `price_snapshot_amount >= 0`
- `order_items`
  - `quantity > 0`
  - `price_amount >= 0`
  - `subtotal_amount >= 0`
- `orders`
  - `total_amount >= 0`
  - status is restricted to:
    `PENDING_PAYMENT, PAID, PROCESSING, READY_TO_SHIP, SHIPPED, COMPLETED, CANCELLED, EXPIRED, FAILED`
- `payments`
  - `amount >= 0`
  - status is restricted to:
    `PENDING, PAID, EXPIRED, FAILED, CANCELLED, REFUNDED`

### Added indexes
- `products(is_active)`
- `products(category_id)`
- `orders(customer_id)`
- `orders(status)`
- `payments(order_id)`
- `payments(provider, provider_reference)`
- `cart_items(cart_id, product_id)`
- unique `payment_webhook_events(provider, event_id)`

### Migration file
Apply:
- `migrations/000007_commerce_integrity_constraints.up.sql`

Rollback:
- `migrations/000007_commerce_integrity_constraints.down.sql`
