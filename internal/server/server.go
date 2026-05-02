package server

import (
	"github.com/gin-gonic/gin"
	"go-be-mono-commerce/internal/config"
	"go-be-mono-commerce/internal/database"
	"go-be-mono-commerce/internal/middleware"
	"go-be-mono-commerce/internal/payment"
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
	v1.GET("/products", func(c *gin.Context) { response.OK(c, gin.H{"items": []any{}}) })
	v1.GET("/products/:slug", func(c *gin.Context) { response.OK(c, gin.H{"slug": c.Param("slug")}) })
	v1.GET("/categories", func(c *gin.Context) { response.OK(c, gin.H{"items": []any{}}) })

	auth := v1.Group("/auth")
	auth.POST("/customer/register", ok)
	auth.POST("/customer/login", ok)
	auth.POST("/admin/login", ok)
	auth.POST("/forgot-password", ok)
	auth.POST("/reset-password", ok)
	auth.GET("/me", middleware.JWT(cfg.JWTSecret, "customer", "admin"), ok)

	cust := v1.Group("/customers/me", middleware.JWT(cfg.JWTSecret, "customer"))
	cust.GET("", ok)
	cust.PUT("", ok)
	cust.GET("/addresses", ok)
	cust.POST("/addresses", ok)
	cust.PUT("/addresses/:id", ok)
	cust.DELETE("/addresses/:id", ok)
	cust.GET("/orders", ok)
	cust.GET("/orders/:id", ok)
	cart := v1.Group("/cart", middleware.JWT(cfg.JWTSecret, "customer"))
	cart.GET("", ok)
	cart.POST("/items", ok)
	cart.PUT("/items/:id", ok)
	cart.DELETE("/items/:id", ok)
	cart.DELETE("", ok)
	ord := v1.Group("/orders", middleware.JWT(cfg.JWTSecret, "customer"))
	ord.POST("/checkout", ok)
	ord.GET("", ok)
	ord.GET(":id", ok)

	paySvc, err := payment.NewService(db, cfg.PaymentProvider)
	if err != nil {
		panic(err)
	}
	payHandler := payment.NewHandler(paySvc)
	pay := v1.Group("/payments", middleware.JWT(cfg.JWTSecret, "customer"))
	pay.POST("/orders/:order_id/pay", payHandler.CreatePayment)
	pay.GET(":id/status", payHandler.GetPaymentStatus)
	v1.POST("/webhooks/payments/midtrans", payHandler.MidtransWebhook)
	v1.POST("/webhooks/payments/xendit", payHandler.XenditWebhook)

	admin := v1.Group("/admin", middleware.JWT(cfg.JWTSecret, "admin"))
	admin.GET("/customers", ok)
	admin.GET("/customers/:id", ok)
	admin.GET("/customers/:id/orders", ok)
	admin.POST("/products", ok)
	admin.PUT("/products/:id", ok)
	admin.DELETE("/products/:id", ok)
	admin.PATCH("/products/:id/publish", ok)
	admin.PATCH("/products/:id/unpublish", ok)
	admin.PUT("/products/:id/stock", ok)
	admin.POST("/categories", ok)
	admin.PUT("/categories/:id", ok)
	admin.DELETE("/categories/:id", ok)
	admin.GET("/orders", ok)
	admin.GET("/orders/:id", ok)
	admin.PATCH("/orders/:id/status", ok)
	admin.POST("/uploads/images", ok)
	admin.GET("/reports/orders", ok)
	admin.GET("/reports/sales", ok)
	admin.GET("/reports/products", ok)
	admin.GET("/reports/payments", ok)
	admin.GET("/audit-logs", ok)
}

func ok(c *gin.Context) { response.OK(c, gin.H{"todo": true}) }

func (s *Server) Run() error { return s.engine.Run(":" + s.cfg.HTTPPort) }

func healthz(c *gin.Context) {
	response.OK(c, gin.H{"status": "healthy"})
}
