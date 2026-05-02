# Go E-Commerce Modular Monolith Skeleton

## Stack
Go 1.22+, Gin, GORM, PostgreSQL, JWT, bcrypt, Zap.

## Run locally
1. Copy env: `cp .env.example .env`
2. Start postgres: `docker compose -f deployments/docker-compose.yml up -d postgres`
3. Run tests: `go test ./...`
4. Run API: `go run ./cmd/api`
5. Health check: `curl -s localhost:8080/healthz`

## Migrations
SQL files are in `migrations/` (intended for golang-migrate/goose integration).

Run in order on an empty PostgreSQL database:
1. `000001_init.up.sql`
2. `000003_ecommerce_schema.up.sql`
3. `000002_seed_admin.up.sql`

The admin seed inserts a default super admin account:
- Email: `admin@example.com`
- Password: `admin12345`
- Role: `SUPER_ADMIN`
- Password storage uses PostgreSQL `pgcrypto` bcrypt hash via `crypt(..., gen_salt('bf'))`.

## API
Base URL: `http://localhost:8080/api/v1`

### Example curl
- Register customer: `curl -X POST localhost:8080/api/v1/auth/customer/register -H 'Content-Type: application/json' -d '{"name":"Sena","email":"sena@example.com","phone":"08123456789","password":"secret123"}'`
- Login customer: `curl -X POST localhost:8080/api/v1/auth/customer/login -H 'Content-Type: application/json' -d '{"email":"sena@example.com","password":"secret123"}'`
- Login admin: `curl -X POST localhost:8080/api/v1/auth/admin/login -H 'Content-Type: application/json' -d '{"email":"admin@example.com","password":"admin12345"}'`
- Create product: `curl -X POST localhost:8080/api/v1/admin/products -H 'Authorization: Bearer <admin_token>'`
- Add to cart: `curl -X POST localhost:8080/api/v1/cart/items -H 'Authorization: Bearer <customer_token>'`
- Checkout: `curl -X POST localhost:8080/api/v1/orders/checkout -H 'Authorization: Bearer <customer_token>'`
- Create payment: `curl -X POST localhost:8080/api/v1/payments/orders/<order_id>/pay -H 'Authorization: Bearer <customer_token>'`
- Simulate webhook: `curl -X POST localhost:8080/api/v1/webhooks/payments/midtrans -d '{}'`

## Notes
- Payment providers are abstractions with Midtrans/Xendit skeleton implementations.
- Replace TODO placeholders with real gateway calls and signature validation.

- Auth me: `curl -X GET localhost:8080/api/v1/auth/me -H 'Authorization: Bearer <token>'`
