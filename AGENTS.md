Current context:
- The repo uses Go, Gin, GORM, PostgreSQL, JWT, bcrypt, and Zap.
- The project already has folders such as internal/auth, internal/customer, internal/product, internal/category, internal/cart, internal/order, internal/payment, internal/report, internal/upload, internal/audit, internal/server, internal/database, pkg, migrations, deployments.
- Payment module already has PaymentProvider interface and Midtrans/Xendit skeleton providers.
- Many handlers/services are still skeleton or placeholders.
- README currently contains empty curl examples and needs to be updated with real examples.

Main goal:
Turn this skeleton into a working MVP backend for a Standard E-Commerce Platform.

Important rules:
1. Do NOT rewrite the whole repository from zero.
2. Keep the modular monolith architecture.
3. Keep package boundaries clean.
4. Follow repository-service-handler pattern when possible.
5. Make sure `go test ./...` passes.
6. Make sure `go run ./cmd/api` works.
7. Make sure Docker Compose can start PostgreSQL and the app.
8. Do not hardcode secrets.
9. Use environment variables from `.env`.
10. Add meaningful errors and consistent JSON responses.
11. Update README with real working curl examples.
12. Keep payment providers as abstractions. Do not hardcode Xendit or Midtrans directly inside order service.
13. Use database transactions for checkout and payment webhook updates.
14. Make webhook handling idempotent.
15. Avoid over-engineering. This is still an MVP, not NASA commerce gateway.

Expected response:
- First inspect the repository.
- Identify what is missing.
- Implement changes incrementally.
- After each major change, run:
  - gofmt
  - go test ./...
  - go run ./cmd/api or at least verify compile with go test
- Summarize changed files and next steps.

Functional MVP scope:

A. Auth
Implement:
- Customer register
- Customer login
- Admin login
- JWT generation
- JWT middleware
- Role-based middleware for admin/customer
- Password hashing using bcrypt
- Seed initial admin user through migration or seed command

Endpoints:
POST /api/v1/auth/customer/register
POST /api/v1/auth/customer/login
POST /api/v1/auth/admin/login
GET  /api/v1/auth/me

B. Customer
Implement:
- Get customer profile
- Update customer profile
- Address book CRUD
- Customer order history

Endpoints:
GET    /api/v1/customers/me
PUT    /api/v1/customers/me
GET    /api/v1/customers/me/addresses
POST   /api/v1/customers/me/addresses
PUT    /api/v1/customers/me/addresses/:id
DELETE /api/v1/customers/me/addresses/:id
GET    /api/v1/customers/me/orders
GET    /api/v1/customers/me/orders/:id

C. Product and Category
Implement:
- Public product list with pagination
- Public product detail by slug
- Public category list
- Admin product CRUD
- Admin category CRUD
- Publish/unpublish product
- Update stock
- Product image URL support

Public endpoints:
GET /api/v1/products
GET /api/v1/products/:slug
GET /api/v1/categories

Admin endpoints:
POST   /api/v1/admin/products
PUT    /api/v1/admin/products/:id
DELETE /api/v1/admin/products/:id
PATCH  /api/v1/admin/products/:id/publish
PATCH  /api/v1/admin/products/:id/unpublish
PUT    /api/v1/admin/products/:id/stock
POST   /api/v1/admin/categories
PUT    /api/v1/admin/categories/:id
DELETE /api/v1/admin/categories/:id

D. Cart
Implement:
- One active cart per customer
- Add product to cart
- Update quantity
- Remove item
- Clear cart
- Validate product exists
- Validate product is active
- Validate stock
- Store price snapshot at cart item level

Endpoints:
GET    /api/v1/cart
POST   /api/v1/cart/items
PUT    /api/v1/cart/items/:id
DELETE /api/v1/cart/items/:id
DELETE /api/v1/cart

E. Order and Checkout
Implement:
- Checkout from active cart
- Validate cart is not empty
- Validate stock again inside DB transaction
- Create order
- Create order items
- Save price snapshot
- Reduce stock or reserve stock
- Mark cart as checked out
- Generate unique order number
- Customer order list
- Customer order detail
- Admin order list
- Admin order detail
- Admin update order status

Endpoints:
POST  /api/v1/orders/checkout
GET   /api/v1/orders
GET   /api/v1/orders/:id
GET   /api/v1/admin/orders
GET   /api/v1/admin/orders/:id
PATCH /api/v1/admin/orders/:id/status

Order statuses:
PENDING_PAYMENT
PAID
PROCESSING
READY_TO_SHIP
SHIPPED
COMPLETED
CANCELLED
EXPIRED
FAILED

Checkout transaction requirements:
- Use database transaction.
- Lock product rows during stock validation/update.
- If stock is insufficient, rollback.
- If any insert fails, rollback.
- Do not allow checkout if cart is empty.
- Do not allow checkout for inactive product.
- Store item price snapshot, do not rely on current product price later.

F. Payment
Improve existing payment module:
- Use existing PaymentProvider interface.
- Keep MidtransProvider and XenditProvider.
- Provider selected using PAYMENT_PROVIDER env.
- Create payment for order.
- Prevent creating duplicate active payment for same order if still pending.
- Save payment record.
- Store provider reference.
- Store redirect URL/payment URL if field exists or add it.
- Implement webhook handling:
  - Validate provider
  - Validate signature placeholder
  - Parse webhook payload
  - Find payment by provider + provider_reference
  - Idempotent status update
  - Update payment status
  - Update order status
  - Write audit log

Endpoints:
POST /api/v1/payments/orders/:order_id/pay
GET  /api/v1/payments/:id/status
POST /api/v1/webhooks/payments/midtrans
POST /api/v1/webhooks/payments/xendit

Payment statuses:
PENDING
PAID
EXPIRED
FAILED
CANCELLED
REFUNDED

Webhook idempotency:
- If the incoming webhook event has already been processed, do not update twice.
- If current status is already PAID, do not downgrade to PENDING.
- If current status is PAID, do not change it to FAILED/EXPIRED from late webhook unless explicitly allowed.
- Add audit log for duplicate webhook.

G. Admin Customer
Implement:
GET /api/v1/admin/customers
GET /api/v1/admin/customers/:id
GET /api/v1/admin/customers/:id/orders

H. Reports
Implement basic report:
GET /api/v1/admin/reports/orders
GET /api/v1/admin/reports/sales
GET /api/v1/admin/reports/products
GET /api/v1/admin/reports/payments

Reports should support:
- date_from
- date_to
- simple aggregation

I. Audit Log
Implement:
GET /api/v1/admin/audit-logs

Write audit logs for:
- Admin login
- Product create/update/delete
- Order status update
- Payment webhook received
- Duplicate webhook received

J. Database
Ensure database models and migrations are aligned.

Tables required:
- customers
- customer_addresses
- admin_users
- categories
- products
- product_images
- carts
- cart_items
- orders
- order_items
- payments
- payment_webhook_events if needed
- audit_logs

Requirements:
- UUID primary keys
- created_at and updated_at
- soft delete where useful
- unique customer email
- unique admin email
- unique product slug
- unique category slug
- unique order number
- index payment provider_reference
- numeric or bigint amount handling must be consistent
- add constraints and indexes where needed

K. Response Format
Use consistent response:

Success:
{
  "success": true,
  "message": "OK",
  "data": {},
  "error": null
}

Error:
{
  "success": false,
  "message": "Validation error",
  "data": null,
  "error": {
    "code": "VALIDATION_ERROR",
    "details": []
  }
}

L. README
Update README with:
- setup instructions
- env example explanation
- migration command
- seed admin command or default seeded admin
- real curl examples for:
  - register customer
  - login customer
  - login admin
  - create category
  - create product
  - list products
  - add to cart
  - checkout
  - create payment
  - simulate Midtrans webhook
  - simulate Xendit webhook
  - admin order list
  - update order status

M. Testing
Add tests for:
- auth password hashing/login
- JWT parsing
- product creation validation
- cart add/update
- checkout transaction insufficient stock
- checkout success
- payment provider selection
- webhook idempotency
- paid webhook updates order to PAID

Acceptance criteria:
- `go test ./...` passes
- `go run ./cmd/api` starts
- docker compose starts postgres
- register/login works
- admin login works
- admin can create category and product
- customer can add product to cart
- customer can checkout
- payment can be created
- webhook can update payment and order status
- README curl examples work
