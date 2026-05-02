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

## API
Base URL: `http://localhost:8080/api/v1`

### Example curl
- Register customer: `curl -X POST localhost:8080/api/v1/auth/customer/register -H 'Content-Type: application/json' -d '{}'`
- Login customer: `curl -X POST localhost:8080/api/v1/auth/customer/login -H 'Content-Type: application/json' -d '{}'`
- Login admin: `curl -X POST localhost:8080/api/v1/auth/admin/login -H 'Content-Type: application/json' -d '{}'`
- Create product: `curl -X POST localhost:8080/api/v1/admin/products -H 'Authorization: Bearer <admin_token>'`
- Add to cart: `curl -X POST localhost:8080/api/v1/cart/items -H 'Authorization: Bearer <customer_token>'`
- Checkout: `curl -X POST localhost:8080/api/v1/orders/checkout -H 'Authorization: Bearer <customer_token>'`
- Create payment: `curl -X POST localhost:8080/api/v1/payments/orders/<order_id>/pay -H 'Authorization: Bearer <customer_token>'`
- Simulate webhook: `curl -X POST localhost:8080/api/v1/webhooks/payments/midtrans -d '{}'`

## Notes
- Payment providers are abstractions with Midtrans/Xendit skeleton implementations.
- Replace TODO placeholders with real gateway calls and signature validation.
