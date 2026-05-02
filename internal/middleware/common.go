package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CORS(origin string) gin.HandlerFunc {
	return cors.New(cors.Config{AllowOrigins: []string{origin}, AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Authorization", "Content-Type"}})
}
func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		s := time.Now()
		c.Next()
		log.Info("http", zap.String("path", c.FullPath()), zap.String("method", c.Request.Method), zap.Int("status", c.Writer.Status()), zap.Duration("latency", time.Since(s)))
	}
}
