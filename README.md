# Standard E-Commerce Backend (Modular Monolith)

Backend MVP for a standard e-commerce platform using a modular monolith architecture.

## 1) Project Overview

This service provides:
- **User Storefront API**: customer auth, profile/address, catalog browsing, cart, checkout, payment, order history.
- **Admin Backoffice API**: admin auth, category/product management, customer/order visibility, reports, audit logs.
- **Payment Provider Adapter**: provider abstraction with Midtrans/Xendit handlers.

## 2) Tech Stack

- Go
- Gin
- GORM
- PostgreSQL
- JWT
- bcrypt
- Zap
- Docker Compose

## 3) Prerequisites

- Go **1.22+**
- Docker + Docker Compose plugin
- `curl`
- `jq` (recommended for token/id extraction)
- `make` (optional; only if you add your own aliases)

## 4) Setup (Clean Clone)

```bash
git clone <your-repo-url>
cd go-be-mono-commerce
cp .env.example .env
```

Start PostgreSQL (from repo root):

```bash
docker compose -f deployments/docker-compose.yml up -d postgres
```

Run migrations (manual SQL order expected by this repo):

```bash
export PGPASSWORD='postgres'
psql -h localhost -p 5432 -U postgres -d ecommerce -f migrations/000001_init.up.sql
psql -h localhost -p 5432 -U postgres -d ecommerce -f migrations/000003_ecommerce_schema.up.sql
psql -h localhost -p 5432 -U postgres -d ecommerce -f migrations/000002_seed_admin.up.sql
```

Seed admin:
- Already handled by `migrations/000002_seed_admin.up.sql`.
- Default seeded admin (for local dev):
  - `admin@example.com`
  - `admin12345`

Run app:

```bash
go run ./cmd/api
```

## 5) Environment Variables

`.env.example` contains baseline values.

Required/important values:

- `HTTP_PORT`: API port (default `8080`).
  - Note: the app reads `HTTP_PORT` (not `APP_PORT`).
- `DB_DSN`: PostgreSQL DSN used by GORM.
  - Note: current config uses **single DSN** (`DB_DSN`), not split `DB_HOST/DB_PORT/...`.
- `JWT_SECRET`: JWT signing secret.
- `JWT_TTL_HOURS`: token TTL in hours.
- `PAYMENT_PROVIDER`: `midtrans` or `xendit`.
- `PAYMENT_MOCK_MODE`: `true`/`false` (mock provider behavior).
- `MIDTRANS_SERVER_KEY`: Midtrans credential.
- `XENDIT_SECRET_KEY`: Xendit credential.
- `XENDIT_CALLBACK_TOKEN`: callback token used for webhook validation.

Also used:
- `APP_ENV`
- `CORS_ALLOW_ORIGIN`
- `SEED_ADMIN_EMAIL`
- `SEED_ADMIN_PASSWORD`

## 6) Full cURL Flow (Validated Against Current Handlers)

Set shared variables:

```bash
export BASE_URL="http://localhost:8080"
export API_V1="$BASE_URL/api/v1"
```

### 6.1 Health check

```bash
curl -s "$BASE_URL/healthz" | jq
```

### 6.2 Register customer

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

### 6.3 Login customer and export `CUSTOMER_TOKEN`

```bash
CUSTOMER_LOGIN_RESPONSE=$(curl -s -X POST "$API_V1/auth/customer/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "sena.arya@example.com",
    "password": "Secret123!"
  }')

echo "$CUSTOMER_LOGIN_RESPONSE" | jq
export CUSTOMER_TOKEN=$(echo "$CUSTOMER_LOGIN_RESPONSE" | jq -r '.data.access_token')
```

### 6.4 Login admin and export `ADMIN_TOKEN`

```bash
ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "$API_V1/auth/admin/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin12345"
  }')

echo "$ADMIN_LOGIN_RESPONSE" | jq
export ADMIN_TOKEN=$(echo "$ADMIN_LOGIN_RESPONSE" | jq -r '.data.access_token')
```

### 6.5 Create category (admin)

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

### 6.6 Create product (admin)

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
    \"discount_start_at\": \"2026-05-01T00:00:00Z\",
    \"discount_end_at\": \"2026-05-31T23:59:59Z\",
    \"stock\": 50,
    \"images\": [
      {"image_url":"https://example.com/images/speaker-x1-main.jpg","is_primary":true}
    ]
  }")

echo "$CREATE_PRODUCT_RESPONSE" | jq
export PRODUCT_ID=$(echo "$CREATE_PRODUCT_RESPONSE" | jq -r '.data.id')
```

### 6.7 List products (public)

```bash
curl -s "$API_V1/products?page=1&limit=10&sort_by=created_at&sort_order=DESC" | jq
```

### 6.8 Create customer address (required before checkout)

```bash
CREATE_ADDRESS_RESPONSE=$(curl -s -X POST "$API_V1/customers/me/addresses" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "receiver_name": "Sena Arya",
    "phone": "+6281234567890",
    "address": "Jl. Mawar No. 10",
    "city": "Bandung",
    "province": "Jawa Barat",
    "postal_code": "40111",
    "is_default": true
  }')

echo "$CREATE_ADDRESS_RESPONSE" | jq
export ADDRESS_ID=$(echo "$CREATE_ADDRESS_RESPONSE" | jq -r '.data.id')
```

### 6.9 Add product to cart

```bash
curl -s -X POST "$API_V1/cart/items" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"product_id\": \"$PRODUCT_ID\",
    \"quantity\": 2
  }" | jq
```

### 6.10 View cart

```bash
CART_RESPONSE=$(curl -s "$API_V1/cart" -H "Authorization: Bearer $CUSTOMER_TOKEN")
echo "$CART_RESPONSE" | jq
export CART_ITEM_ID=$(echo "$CART_RESPONSE" | jq -r '.data.items[0].id')
```

### 6.11 Checkout

```bash
CHECKOUT_RESPONSE=$(curl -s -X POST "$API_V1/orders/checkout" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"address_id\": \"$ADDRESS_ID\",
    \"notes\": \"Leave at front desk\"
  }")

echo "$CHECKOUT_RESPONSE" | jq
export ORDER_ID=$(echo "$CHECKOUT_RESPONSE" | jq -r '.data.id')
```

### 6.12 Create payment for order

```bash
CREATE_PAYMENT_RESPONSE=$(curl -s -X POST "$API_V1/payments/orders/$ORDER_ID/pay" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN")

echo "$CREATE_PAYMENT_RESPONSE" | jq
export PAYMENT_ID=$(echo "$CREATE_PAYMENT_RESPONSE" | jq -r '.data.payment_id')
export PROVIDER_REFERENCE=$(echo "$CREATE_PAYMENT_RESPONSE" | jq -r '.data.provider_reference')
```

### 6.13 Simulate Midtrans paid webhook

```bash
curl -s -X POST "$API_V1/webhooks/payments/midtrans" \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\": \"$ORDER_ID\",
    \"transaction_status\": \"settlement\",
    \"status_code\": \"200\",
    \"signature_key\": \"mock-signature\",
    \"transaction_id\": \"$PROVIDER_REFERENCE\"
  }" | jq
```

### 6.14 Simulate Xendit paid webhook

```bash
curl -s -X POST "$API_V1/webhooks/payments/xendit" \
  -H "Content-Type: application/json" \
  -H "X-CALLBACK-TOKEN: ${XENDIT_CALLBACK_TOKEN:-dummy-token}" \
  -d "{
    \"id\": \"evt-$ORDER_ID\",
    \"external_id\": \"$ORDER_ID\",
    \"status\": \"PAID\",
    \"reference_id\": \"$PROVIDER_REFERENCE\"
  }" | jq
```

### 6.15 View customer order(s)

```bash
curl -s "$API_V1/orders" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
curl -s "$API_V1/orders/$ORDER_ID" -H "Authorization: Bearer $CUSTOMER_TOKEN" | jq
```

### 6.16 View admin order(s)

```bash
curl -s "$API_V1/admin/orders" -H "Authorization: Bearer $ADMIN_TOKEN" | jq
curl -s "$API_V1/admin/orders/$ORDER_ID" -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

### 6.17 Update order status (admin)

```bash
curl -s -X PATCH "$API_V1/admin/orders/$ORDER_ID/status" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"PROCESSING"}' | jq
```

### 6.18 View admin reports

```bash
curl -s "$API_V1/admin/reports/orders?date_from=2026-01-01&date_to=2026-12-31" -H "Authorization: Bearer $ADMIN_TOKEN" | jq
curl -s "$API_V1/admin/reports/sales?date_from=2026-01-01&date_to=2026-12-31" -H "Authorization: Bearer $ADMIN_TOKEN" | jq
curl -s "$API_V1/admin/reports/products?date_from=2026-01-01&date_to=2026-12-31" -H "Authorization: Bearer $ADMIN_TOKEN" | jq
curl -s "$API_V1/admin/reports/payments?date_from=2026-01-01&date_to=2026-12-31" -H "Authorization: Bearer $ADMIN_TOKEN" | jq
```

## 7) Troubleshooting

- **DB connection failed**
  - Confirm PostgreSQL container is up: `docker compose -f deployments/docker-compose.yml ps`
  - Verify `DB_DSN` in `.env` and DB name/user/password.
- **Migration failed**
  - Ensure DB exists and extension privileges are available.
  - Re-run SQL files in required order.
- **JWT invalid / unauthorized**
  - Ensure `Authorization: Bearer <token>` format.
  - Re-login if token expired (`JWT_TTL_HOURS`).
  - Ensure `JWT_SECRET` is consistent between token generation and API runtime.
- **Admin seed missing**
  - Re-run `migrations/000002_seed_admin.up.sql` and verify `admin_users` table.
- **Payment webhook duplicate**
  - Current payment service is idempotent-aware; duplicate events are recorded and should not double-apply status.
- **Insufficient stock**
  - Checkout validates stock in DB transaction; reduce quantity or update product stock from admin endpoint.

## 8) Test

```bash
go test ./...
```

Race detector (recommended locally):

```bash
go test ./... -race
```

## 9) Notes on Current Implementation Honesty

- This runbook matches currently registered routes and request DTOs.
- Some integrations may run in mock mode depending on `PAYMENT_MOCK_MODE` and provider credentials.
- If provider-specific webhook signatures are strict in your environment, pass matching headers/tokens accordingly.
