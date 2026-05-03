package server

import (
	"github.com/gin-gonic/gin"
	"go-be-mono-commerce/internal/audit"
	"go-be-mono-commerce/internal/auth"
	"go-be-mono-commerce/internal/cart"
	"go-be-mono-commerce/internal/category"
	"go-be-mono-commerce/internal/config"
	"go-be-mono-commerce/internal/customer"
	"go-be-mono-commerce/internal/database"
	"go-be-mono-commerce/internal/middleware"
	"go-be-mono-commerce/internal/order"
	"go-be-mono-commerce/internal/payment"
	"go-be-mono-commerce/internal/product"
	"go-be-mono-commerce/internal/report"
	"go-be-mono-commerce/pkg/logger"
	"go-be-mono-commerce/pkg/response"
	"gorm.io/gorm"
)

type Server struct {
	engine *gin.Engine
	cfg    config.Config
	db     *gorm.DB
}

func New(cfg config.Config) (*Server, error) {
	log, err := logger.New()
	if err != nil {
		return nil, err
	}
	db, err := database.New(cfg)
	if err != nil {
		return nil, err
	}
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS(cfg.CorsAllowOrigin), middleware.RequestLogger(log))
	r.GET("/healthz", healthz)
	v1 := r.Group("/api/v1")
	registerRoutes(v1, cfg, db)
	return &Server{engine: r, cfg: cfg, db: db}, nil
}

func registerRoutes(v1 *gin.RouterGroup, cfg config.Config, db *gorm.DB) {
	catSvc := category.NewService(db)
	prodSvc := product.NewService(db)
	v1.GET("/categories", func(c *gin.Context) {
		items, err := catSvc.ListActive()
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, gin.H{"items": items})
	})
	v1.GET("/products", func(c *gin.Context) {
		res, err := prodSvc.ListPublic(c)
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, res)
	})
	v1.GET("/products/:slug", func(c *gin.Context) {
		item, err := prodSvc.DetailBySlug(c.Param("slug"))
		if err != nil {
			product.HandleErr(c, err)
			return
		}
		response.OK(c, item)
	})

	authGroup := v1.Group("/auth")
	authSvc := auth.NewService(db, cfg.JWTSecret, cfg.JWTTTLHours)
	auth.RegisterRoutes(authGroup, authSvc)
	authGroup.GET("/me", middleware.AuthJWT(cfg.JWTSecret), middleware.RequireRoles(auth.RoleCustomer, auth.RoleAdmin), func(c *gin.Context) {
		response.OK(c, gin.H{"user_id": c.GetString("user_id"), "role": c.GetString("role")})
	})

	custRepo := customer.NewRepository(db)
	custSvc := customer.NewService(custRepo)
	custHandler := customer.NewHandler(custSvc)
	cust := v1.Group("/customers/me", middleware.AuthJWT(cfg.JWTSecret), middleware.RequireRoles(auth.RoleCustomer))
	cust.GET("", custHandler.Me)
	cust.PUT("", custHandler.UpdateMe)
	cust.GET("/addresses", custHandler.ListAddresses)
	cust.POST("/addresses", custHandler.CreateAddress)
	cust.PUT("/addresses/:id", custHandler.UpdateAddress)
	cust.DELETE("/addresses/:id", custHandler.DeleteAddress)
	cust.GET("/orders", custHandler.ListMyOrders)
	cust.GET("/orders/:id", custHandler.GetMyOrder)
	cartSvc := cart.NewService(db)
	cartGroup := v1.Group("/cart", middleware.AuthJWT(cfg.JWTSecret), middleware.RequireRoles(auth.RoleCustomer))
	cartGroup.GET("", func(c *gin.Context) {
		uid, err := auth.ParseUUID(c.GetString("user_id"))
		if err != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		out, err := cartSvc.Get(uid)
		if err != nil {
			code, msg, ec, de := cart.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.OK(c, out)
	})
	cartGroup.POST("/items", func(c *gin.Context) {
		uid, err := auth.ParseUUID(c.GetString("user_id"))
		if err != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		var req cart.AddItemRequest
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		if err := cartSvc.AddItem(uid, req); err != nil {
			code, msg, ec, de := cart.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.OK(c, gin.H{"added": true})
	})
	cartGroup.PUT("/items/:id", func(c *gin.Context) {
		uid, err := auth.ParseUUID(c.GetString("user_id"))
		if err != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		itemID, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		var req cart.UpdateItemRequest
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		if err := cartSvc.UpdateItem(uid, itemID, req.Quantity); err != nil {
			code, msg, ec, de := cart.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.OK(c, gin.H{"updated": true})
	})
	cartGroup.DELETE("/items/:id", func(c *gin.Context) {
		uid, err := auth.ParseUUID(c.GetString("user_id"))
		if err != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		itemID, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		if err := cartSvc.RemoveItem(uid, itemID); err != nil {
			code, msg, ec, de := cart.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
	cartGroup.DELETE("", func(c *gin.Context) {
		uid, err := auth.ParseUUID(c.GetString("user_id"))
		if err != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		if err := cartSvc.Clear(uid); err != nil {
			code, msg, ec, de := cart.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.OK(c, gin.H{"cleared": true})
	})
	ordSvc := order.NewService(db)
	ord := v1.Group("/orders", middleware.AuthJWT(cfg.JWTSecret), middleware.RequireRoles(auth.RoleCustomer))
	ord.POST("/checkout", func(c *gin.Context) {
		uid, err := auth.ParseUUID(c.GetString("user_id"))
		if err != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		var req order.CheckoutRequest
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		out, err := ordSvc.Checkout(uid, req)
		if err != nil {
			code, msg, ec, de := order.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.Created(c, out)
	})
	ord.GET("", func(c *gin.Context) {
		uid, err := auth.ParseUUID(c.GetString("user_id"))
		if err != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		out, err := ordSvc.ListCustomerOrders(uid)
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, gin.H{"items": out})
	})
	ord.GET(":id", func(c *gin.Context) {
		uid, err := auth.ParseUUID(c.GetString("user_id"))
		if err != nil {
			response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
			return
		}
		oid, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		out, err := ordSvc.GetCustomerOrder(uid, oid)
		if err != nil {
			code, msg, ec, de := order.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.OK(c, out)
	})

	paySvc, err := payment.NewService(db, cfg)
	if err != nil {
		panic(err)
	}
	payHandler := payment.NewHandler(paySvc)
	pay := v1.Group("/payments", middleware.AuthJWT(cfg.JWTSecret), middleware.RequireRoles(auth.RoleCustomer))
	pay.POST("/orders/:order_id/pay", payHandler.CreatePayment)
	pay.GET(":id/status", payHandler.GetPaymentStatus)
	v1.POST("/webhooks/payments/midtrans", payHandler.MidtransWebhook)
	v1.POST("/webhooks/payments/xendit", payHandler.XenditWebhook)

	admin := v1.Group("/admin", middleware.AuthJWT(cfg.JWTSecret), middleware.RequireRoles(auth.RoleAdmin))
	admin.GET("/customers", custHandler.AdminListCustomers)
	admin.GET("/customers/:id", custHandler.AdminGetCustomer)
	admin.GET("/customers/:id/orders", custHandler.AdminGetCustomerOrders)
	admin.POST("/categories", func(c *gin.Context) {
		var req category.UpsertCategoryRequest
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		out, err := catSvc.Create(req)
		if err != nil {
			product.HandleErr(c, err)
			return
		}
		response.Created(c, out)
	})
	admin.PUT("/categories/:id", func(c *gin.Context) {
		id, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		var req category.UpsertCategoryRequest
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		out, err := catSvc.Update(id, req)
		if err != nil {
			product.HandleErr(c, err)
			return
		}
		response.OK(c, out)
	})
	admin.DELETE("/categories/:id", func(c *gin.Context) {
		id, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		if err := catSvc.Delete(id); err != nil {
			product.HandleErr(c, err)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
	admin.POST("/products", func(c *gin.Context) {
		var req product.UpsertProductRequest
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		out, err := prodSvc.Create(req)
		if err != nil {
			product.HandleErr(c, err)
			return
		}
		response.Created(c, out)
	})
	admin.PUT("/products/:id", func(c *gin.Context) {
		id, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		var req product.UpsertProductRequest
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		out, err := prodSvc.Update(id, req)
		if err != nil {
			product.HandleErr(c, err)
			return
		}
		response.OK(c, out)
	})
	admin.DELETE("/products/:id", func(c *gin.Context) {
		id, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		if err := prodSvc.Delete(id); err != nil {
			product.HandleErr(c, err)
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
	admin.PATCH("/products/:id/publish", func(c *gin.Context) {
		id, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		if err := prodSvc.SetActive(id, true); err != nil {
			product.HandleErr(c, err)
			return
		}
		response.OK(c, gin.H{"published": true})
	})
	admin.PATCH("/products/:id/unpublish", func(c *gin.Context) {
		id, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		if err := prodSvc.SetActive(id, false); err != nil {
			product.HandleErr(c, err)
			return
		}
		response.OK(c, gin.H{"published": false})
	})
	admin.PUT("/products/:id/stock", func(c *gin.Context) {
		id, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		var req struct {
			Stock int `json:"stock"`
		}
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		if err := prodSvc.SetStock(id, req.Stock); err != nil {
			product.HandleErr(c, err)
			return
		}
		response.OK(c, gin.H{"stock": req.Stock})
	})
	admin.GET("/orders", func(c *gin.Context) {
		out, err := ordSvc.ListAdminOrders()
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, gin.H{"items": out})
	})
	admin.GET("/orders/:id", func(c *gin.Context) {
		oid, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		out, err := ordSvc.GetAdminOrder(oid)
		if err != nil {
			code, msg, ec, de := order.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.OK(c, out)
	})
	admin.PATCH("/orders/:id/status", func(c *gin.Context) {
		oid, err := auth.ParseUUID(c.Param("id"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
			return
		}
		var req struct {
			Status string `json:"status"`
		}
		if c.ShouldBindJSON(&req) != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		out, err := ordSvc.UpdateOrderStatus(oid, req.Status)
		if err != nil {
			code, msg, ec, de := order.HandleErr(err)
			response.Fail(c, code, msg, ec, de)
			return
		}
		response.OK(c, out)
	})
	admin.POST("/uploads/images", ok)
	reportSvc := report.NewService(db)
	auditSvc := audit.NewService(db)
	admin.GET("/reports/orders", func(c *gin.Context) {
		f, err := report.ParseFilter(c.Query("date_from"), c.Query("date_to"), c.Query("page"), c.Query("limit"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid date format"})
			return
		}
		out, err := reportSvc.OrderReport(f)
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, out)
	})
	admin.GET("/reports/sales", func(c *gin.Context) {
		f, err := report.ParseFilter(c.Query("date_from"), c.Query("date_to"), c.Query("page"), c.Query("limit"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid date format"})
			return
		}
		out, err := reportSvc.SalesReport(f)
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, out)
	})
	admin.GET("/reports/products", func(c *gin.Context) {
		f, err := report.ParseFilter(c.Query("date_from"), c.Query("date_to"), c.Query("page"), c.Query("limit"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid date format"})
			return
		}
		items, total, err := reportSvc.ProductSalesReport(f)
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, gin.H{"items": items, "page": f.Page, "limit": f.Limit, "total": total})
	})
	admin.GET("/reports/payments", func(c *gin.Context) {
		f, err := report.ParseFilter(c.Query("date_from"), c.Query("date_to"), c.Query("page"), c.Query("limit"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid date format"})
			return
		}
		out, err := reportSvc.PaymentReport(f)
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, out)
	})
	admin.GET("/audit-logs", func(c *gin.Context) {
		f, err := audit.ParseFilter(c.Query("actor_type"), c.Query("action"), c.Query("resource_type"), c.Query("date_from"), c.Query("date_to"), c.Query("page"), c.Query("limit"))
		if err != nil {
			response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid date format"})
			return
		}
		items, total, err := auditSvc.List(f)
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, gin.H{"items": items, "page": f.Page, "limit": f.Limit, "total": total})
	})
}

func ok(c *gin.Context) { response.OK(c, gin.H{"todo": true}) }

func (s *Server) Run() error { return s.engine.Run(":" + s.cfg.HTTPPort) }

func healthz(c *gin.Context) {
	response.OK(c, gin.H{"status": "healthy"})
}
