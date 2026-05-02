package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-be-mono-commerce/internal/auth"
	"go-be-mono-commerce/internal/config"
	"go-be-mono-commerce/internal/database"
	"go-be-mono-commerce/internal/middleware"
	"go-be-mono-commerce/internal/payment"
	"go-be-mono-commerce/pkg/response"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Server struct {
	engine   *gin.Engine
	cfg      config.Config
	db       *gorm.DB
	provider payment.PaymentProvider
}

func New(cfg config.Config) (*Server, error) {
	db, err := database.New(cfg)
	if err != nil {
		return nil, err
	}
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS(cfg.CorsAllowOrigin))
	s := &Server{engine: r, cfg: cfg, db: db, provider: pickProvider(cfg.PaymentProvider)}
	s.seedAdmin()
	v1 := r.Group("/api/v1")
	s.registerRoutes(v1)
	return s, nil
}

func pickProvider(name string) payment.PaymentProvider {
	if strings.ToLower(name) == "xendit" {
		return &payment.XenditProvider{}
	}
	return &payment.MidtransProvider{}
}

func (s *Server) seedAdmin() {
	var c int64
	s.db.Model(&database.AdminUser{}).Where("email=?", s.cfg.SeedAdminEmail).Count(&c)
	if c == 0 {
		h, _ := bcrypt.GenerateFromPassword([]byte(s.cfg.SeedAdminPass), bcrypt.DefaultCost)
		s.db.Create(&database.AdminUser{Email: s.cfg.SeedAdminEmail, PasswordHash: string(h), Name: "Super Admin"})
	}
}

func (s *Server) registerRoutes(v1 *gin.RouterGroup) {
	v1.GET("/products", s.listProducts)
	v1.GET("/products/:slug", s.productBySlug)
	v1.GET("/categories", s.listCategories)
	a := v1.Group("/auth")
	a.POST("/customer/register", s.registerCustomer)
	a.POST("/customer/login", s.loginCustomer)
	a.POST("/admin/login", s.loginAdmin)
	a.POST("/forgot-password", ok)
	a.POST("/reset-password", ok)
	a.GET("/me", middleware.JWT(s.cfg.JWTSecret, "customer", "admin"), s.me)
	c := v1.Group("/customers/me", middleware.JWT(s.cfg.JWTSecret, "customer"))
	c.GET("", s.customerMe)
	c.PUT("", s.customerUpdate)
	c.GET("/addresses", s.addrList)
	c.POST("/addresses", s.addrCreate)
	c.PUT("/addresses/:id", s.addrUpdate)
	c.DELETE("/addresses/:id", s.addrDelete)
	c.GET("/orders", s.myOrders)
	c.GET("/orders/:id", s.myOrder)
	cart := v1.Group("/cart", middleware.JWT(s.cfg.JWTSecret, "customer"))
	cart.GET("", s.getCart)
	cart.POST("/items", s.addCartItem)
	cart.PUT("/items/:id", s.updateCartItem)
	cart.DELETE("/items/:id", s.deleteCartItem)
	cart.DELETE("", s.clearCart)
	ord := v1.Group("/orders", middleware.JWT(s.cfg.JWTSecret, "customer"))
	ord.POST("/checkout", s.checkout)
	ord.GET("", s.myOrders)
	ord.GET("/:id", s.myOrder)
	pay := v1.Group("/payments", middleware.JWT(s.cfg.JWTSecret, "customer"))
	pay.POST("/orders/:order_id/pay", s.createPayment)
	pay.GET("/:id/status", s.paymentStatus)
	v1.POST("/webhooks/payments/midtrans", s.webhook)
	v1.POST("/webhooks/payments/xendit", s.webhook)
	ad := v1.Group("/admin", middleware.JWT(s.cfg.JWTSecret, "admin"))
	ad.GET("/customers", s.adminCustomers)
	ad.GET("/customers/:id", s.adminCustomer)
	ad.GET("/customers/:id/orders", s.adminCustomerOrders)
	ad.POST("/products", s.createProduct)
	ad.PUT("/products/:id", s.updateProduct)
	ad.DELETE("/products/:id", s.deleteProduct)
	ad.PATCH("/products/:id/publish", s.publishProduct)
	ad.PATCH("/products/:id/unpublish", s.unpublishProduct)
	ad.PUT("/products/:id/stock", s.stockProduct)
	ad.POST("/categories", s.createCategory)
	ad.PUT("/categories/:id", s.updateCategory)
	ad.DELETE("/categories/:id", s.deleteCategory)
	ad.GET("/orders", s.adminOrders)
	ad.GET("/orders/:id", s.adminOrder)
	ad.PATCH("/orders/:id/status", s.adminOrderStatus)
	ad.POST("/uploads/images", ok)
	ad.GET("/reports/orders", ok)
	ad.GET("/reports/sales", ok)
	ad.GET("/reports/products", ok)
	ad.GET("/reports/payments", ok)
	ad.GET("/audit-logs", s.auditLogs)
}

func uid(c *gin.Context) uuid.UUID { id, _ := uuid.Parse(c.GetString("user_id")); return id }
func ok(c *gin.Context)            { response.OK(c, gin.H{"todo": true}) }
func (s *Server) Run() error       { return s.engine.Run(":" + s.cfg.HTTPPort) }

func (s *Server) registerCustomer(c *gin.Context) {
	var in struct{ Name, Email, Password string }
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", nil)
		return
	}
	h, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	u := database.Customer{Name: in.Name, Email: in.Email, PasswordHash: string(h)}
	if err := s.db.Create(&u).Error; err != nil {
		response.Fail(c, 400, "Create failed", "CREATE_FAILED", err.Error())
		return
	}
	response.Created(c, u)
}
func (s *Server) loginCustomer(c *gin.Context) { s.login(c, "customer") }
func (s *Server) loginAdmin(c *gin.Context)    { s.login(c, "admin") }
func (s *Server) login(c *gin.Context, role string) {
	var in struct{ Email, Password string }
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", nil)
		return
	}
	var id uuid.UUID
	var hash string
	if role == "customer" {
		var u database.Customer
		if s.db.Where("email=?", in.Email).First(&u).Error != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		id = u.ID
		hash = u.PasswordHash
	} else {
		var u database.AdminUser
		if s.db.Where("email=?", in.Email).First(&u).Error != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		id = u.ID
		hash = u.PasswordHash
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	t, _ := auth.Sign(s.cfg.JWTSecret, id.String(), role)
	response.OK(c, gin.H{"access_token": t})
}
func (s *Server) me(c *gin.Context) {
	response.OK(c, gin.H{"user_id": c.GetString("user_id"), "role": c.GetString("role")})
}
func (s *Server) customerMe(c *gin.Context) {
	var u database.Customer
	s.db.First(&u, "id=?", uid(c))
	response.OK(c, u)
}
func (s *Server) customerUpdate(c *gin.Context) {
	var in struct{ Name string }
	c.ShouldBindJSON(&in)
	s.db.Model(&database.Customer{}).Where("id=?", uid(c)).Update("name", in.Name)
	s.customerMe(c)
}
func (s *Server) addrList(c *gin.Context) {
	var a []database.CustomerAddress
	s.db.Where("customer_id=?", uid(c)).Find(&a)
	response.OK(c, a)
}
func (s *Server) addrCreate(c *gin.Context) {
	var in database.CustomerAddress
	c.ShouldBindJSON(&in)
	in.CustomerID = uid(c)
	s.db.Create(&in)
	response.Created(c, in)
}
func (s *Server) addrUpdate(c *gin.Context) {
	var in database.CustomerAddress
	c.ShouldBindJSON(&in)
	s.db.Model(&database.CustomerAddress{}).Where("id=? and customer_id=?", c.Param("id"), uid(c)).Updates(map[string]any{"label": in.Label, "address": in.Address, "city": in.City})
	response.OK(c, gin.H{"updated": true})
}
func (s *Server) addrDelete(c *gin.Context) {
	s.db.Where("id=? and customer_id=?", c.Param("id"), uid(c)).Delete(&database.CustomerAddress{})
	response.OK(c, gin.H{"deleted": true})
}
func (s *Server) listProducts(c *gin.Context) {
	var p []database.Product
	s.db.Where("published=?", true).Find(&p)
	response.OK(c, p)
}
func (s *Server) productBySlug(c *gin.Context) {
	var p database.Product
	if s.db.Where("slug=?", c.Param("slug")).First(&p).Error != nil {
		response.Fail(c, 404, "Not found", "NOT_FOUND", nil)
		return
	}
	response.OK(c, p)
}
func (s *Server) listCategories(c *gin.Context) {
	var x []database.Category
	s.db.Where("is_active=?", true).Find(&x)
	response.OK(c, x)
}
func (s *Server) createCategory(c *gin.Context) {
	var in database.Category
	c.ShouldBindJSON(&in)
	s.db.Create(&in)
	response.Created(c, in)
}
func (s *Server) updateCategory(c *gin.Context) {
	var in database.Category
	c.ShouldBindJSON(&in)
	s.db.Model(&database.Category{}).Where("id=?", c.Param("id")).Updates(map[string]any{"name": in.Name, "slug": in.Slug, "is_active": in.IsActive})
	response.OK(c, gin.H{"updated": true})
}
func (s *Server) deleteCategory(c *gin.Context) {
	s.db.Delete(&database.Category{}, "id=?", c.Param("id"))
	response.OK(c, gin.H{"deleted": true})
}
func (s *Server) createProduct(c *gin.Context) {
	var in database.Product
	c.ShouldBindJSON(&in)
	s.db.Create(&in)
	s.audit(c, "PRODUCT_CREATE", "product", in.ID.String())
	response.Created(c, in)
}
func (s *Server) updateProduct(c *gin.Context) {
	var in database.Product
	c.ShouldBindJSON(&in)
	s.db.Model(&database.Product{}).Where("id=?", c.Param("id")).Updates(in)
	s.audit(c, "PRODUCT_UPDATE", "product", c.Param("id"))
	response.OK(c, gin.H{"updated": true})
}
func (s *Server) deleteProduct(c *gin.Context) {
	s.db.Delete(&database.Product{}, "id=?", c.Param("id"))
	s.audit(c, "PRODUCT_DELETE", "product", c.Param("id"))
	response.OK(c, gin.H{"deleted": true})
}
func (s *Server) publishProduct(c *gin.Context) {
	s.db.Model(&database.Product{}).Where("id=?", c.Param("id")).Update("published", true)
	response.OK(c, gin.H{"published": true})
}
func (s *Server) unpublishProduct(c *gin.Context) {
	s.db.Model(&database.Product{}).Where("id=?", c.Param("id")).Update("published", false)
	response.OK(c, gin.H{"published": false})
}
func (s *Server) stockProduct(c *gin.Context) {
	var in struct{ Stock int }
	c.ShouldBindJSON(&in)
	s.db.Model(&database.Product{}).Where("id=?", c.Param("id")).Update("stock", in.Stock)
	response.OK(c, gin.H{"stock": in.Stock})
}
func (s *Server) getCart(c *gin.Context) {
	cart := s.findOrCreateCart(uid(c))
	var items []database.CartItem
	s.db.Where("cart_id=?", cart.ID).Find(&items)
	response.OK(c, gin.H{"cart": cart, "items": items})
}
func (s *Server) addCartItem(c *gin.Context) {
	var in struct {
		ProductID string
		Qty       int
	}
	c.ShouldBindJSON(&in)
	p := database.Product{}
	if s.db.First(&p, "id=?", in.ProductID).Error != nil {
		response.Fail(c, 404, "Not found", "NOT_FOUND", nil)
		return
	}
	cart := s.findOrCreateCart(uid(c))
	it := database.CartItem{CartID: cart.ID, ProductID: p.ID, Qty: in.Qty, PriceSnapshot: p.Price}
	s.db.Create(&it)
	response.Created(c, it)
}
func (s *Server) updateCartItem(c *gin.Context) {
	var in struct{ Qty int }
	c.ShouldBindJSON(&in)
	s.db.Model(&database.CartItem{}).Where("id=?", c.Param("id")).Update("qty", in.Qty)
	response.OK(c, gin.H{"updated": true})
}
func (s *Server) deleteCartItem(c *gin.Context) {
	s.db.Delete(&database.CartItem{}, "id=?", c.Param("id"))
	response.OK(c, gin.H{"deleted": true})
}
func (s *Server) clearCart(c *gin.Context) {
	cart := s.findOrCreateCart(uid(c))
	s.db.Where("cart_id=?", cart.ID).Delete(&database.CartItem{})
	response.OK(c, gin.H{"cleared": true})
}
func (s *Server) findOrCreateCart(customerID uuid.UUID) database.Cart {
	var cart database.Cart
	if s.db.Where("customer_id=? and is_active=true", customerID).First(&cart).Error == nil {
		return cart
	}
	cart = database.Cart{CustomerID: customerID, IsActive: true}
	s.db.Create(&cart)
	return cart
}
func (s *Server) checkout(c *gin.Context) {
	cart := s.findOrCreateCart(uid(c))
	var items []database.CartItem
	s.db.Where("cart_id=?", cart.ID).Find(&items)
	if len(items) == 0 {
		response.Fail(c, 400, "Cart empty", "CART_EMPTY", nil)
		return
	}
	tx := s.db.Begin()
	total := int64(0)
	for _, it := range items {
		total += it.PriceSnapshot * int64(it.Qty)
	}
	on := fmt.Sprintf("ORD-%d", time.Now().Unix())
	o := database.Order{CustomerID: uid(c), OrderNumber: on, Status: "PENDING_PAYMENT", TotalAmount: total}
	tx.Create(&o)
	for _, it := range items {
		var p database.Product
		tx.First(&p, "id=?", it.ProductID)
		if p.Stock < it.Qty {
			tx.Rollback()
			response.Fail(c, 400, "Insufficient stock", "INSUFFICIENT_STOCK", nil)
			return
		}
		tx.Model(&p).Update("stock", p.Stock-it.Qty)
		tx.Create(&database.OrderItem{OrderID: o.ID, ProductID: it.ProductID, ProductName: p.Name, Qty: it.Qty, Price: it.PriceSnapshot})
	}
	tx.Where("cart_id=?", cart.ID).Delete(&database.CartItem{})
	tx.Commit()
	response.Created(c, o)
}
func (s *Server) myOrders(c *gin.Context) {
	var o []database.Order
	s.db.Where("customer_id=?", uid(c)).Find(&o)
	response.OK(c, o)
}
func (s *Server) myOrder(c *gin.Context) {
	var o database.Order
	if s.db.Where("id=? and customer_id=?", c.Param("id"), uid(c)).First(&o).Error != nil {
		response.Fail(c, 404, "Not found", "NOT_FOUND", nil)
		return
	}
	response.OK(c, o)
}
func (s *Server) adminOrders(c *gin.Context) {
	var o []database.Order
	s.db.Find(&o)
	response.OK(c, o)
}
func (s *Server) adminOrder(c *gin.Context) {
	var o database.Order
	if s.db.First(&o, "id=?", c.Param("id")).Error != nil {
		response.Fail(c, 404, "Not found", "NOT_FOUND", nil)
		return
	}
	response.OK(c, o)
}
func (s *Server) adminOrderStatus(c *gin.Context) {
	var in struct{ Status string }
	c.ShouldBindJSON(&in)
	s.db.Model(&database.Order{}).Where("id=?", c.Param("id")).Update("status", in.Status)
	s.audit(c, "ORDER_STATUS_UPDATE", "order", c.Param("id"))
	response.OK(c, gin.H{"status": in.Status})
}
func (s *Server) createPayment(c *gin.Context) {
	var o database.Order
	if s.db.Where("id=? and customer_id=?", c.Param("order_id"), uid(c)).First(&o).Error != nil {
		response.Fail(c, 404, "Order not found", "NOT_FOUND", nil)
		return
	}
	pRes, _ := s.provider.CreatePayment(c, o2req(o))
	pay := database.Payment{OrderID: o.ID, Provider: s.cfg.PaymentProvider, ProviderReference: pRes.ProviderReference, Status: pRes.Status, Amount: o.TotalAmount}
	s.db.Create(&pay)
	response.Created(c, gin.H{"payment": pay, "redirect_url": pRes.RedirectURL})
}
func o2req(o database.Order) payment.CreatePaymentRequest {
	return payment.CreatePaymentRequest{OrderID: o.ID.String(), OrderNumber: o.OrderNumber, Amount: o.TotalAmount}
}
func (s *Server) paymentStatus(c *gin.Context) {
	var p database.Payment
	if s.db.First(&p, "id=?", c.Param("id")).Error != nil {
		response.Fail(c, 404, "Not found", "NOT_FOUND", nil)
		return
	}
	st, _ := s.provider.GetPaymentStatus(c, p.ProviderReference)
	response.OK(c, st)
}
func (s *Server) webhook(c *gin.Context) {
	raw, _ := c.GetRawData()
	headers := map[string]string{}
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	_ = s.provider.ValidateWebhook(c, headers, raw)
	ev, _ := s.provider.ParseWebhook(c, raw)
	s.db.Model(&database.Payment{}).Where("provider_reference=?", ev.ProviderReference).Update("status", ev.Status)
	if ev.Status == "PAID" {
		var p database.Payment
		if s.db.Where("provider_reference=?", ev.ProviderReference).First(&p).Error == nil {
			s.db.Model(&database.Order{}).Where("id=?", p.OrderID).Update("status", "PAID")
		}
	}
	s.audit(c, "PAYMENT_WEBHOOK", "payment", ev.ProviderReference)
	response.OK(c, gin.H{"received": true})
}
func (s *Server) adminCustomers(c *gin.Context) {
	var x []database.Customer
	s.db.Find(&x)
	response.OK(c, x)
}
func (s *Server) adminCustomer(c *gin.Context) {
	var x database.Customer
	if s.db.First(&x, "id=?", c.Param("id")).Error != nil {
		response.Fail(c, 404, "Not found", "NOT_FOUND", nil)
		return
	}
	response.OK(c, x)
}
func (s *Server) adminCustomerOrders(c *gin.Context) {
	var o []database.Order
	s.db.Where("customer_id=?", c.Param("id")).Find(&o)
	response.OK(c, o)
}
func (s *Server) audit(c *gin.Context, action, resType, resID string) {
	actor := uid(c)
	s.db.Create(&database.AuditLog{ActorID: &actor, Action: action, ResourceType: resType, ResourceID: resID})
}
func (s *Server) auditLogs(c *gin.Context) {
	var a []database.AuditLog
	page, size := 1, 20
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if z, err := strconv.Atoi(c.Query("size")); err == nil && z > 0 {
		size = z
	}
	s.db.Offset((page - 1) * size).Limit(size).Order("created_at desc").Find(&a)
	response.OK(c, a)
}
