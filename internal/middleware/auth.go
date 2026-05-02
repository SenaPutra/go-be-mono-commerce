package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"go-be-mono-commerce/internal/auth"
	"go-be-mono-commerce/pkg/response"
)

func AuthJWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			response.Error(c, errors.New("UNAUTHORIZED"))
			c.Abort()
			return
		}
		claims, err := auth.Parse(secret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			response.Error(c, errors.New("UNAUTHORIZED"))
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	roleAllowed := map[string]bool{}
	for _, r := range allowedRoles {
		roleAllowed[r] = true
	}
	return func(c *gin.Context) {
		roleVal, ok := c.Get("role")
		if !ok || !roleAllowed[roleVal.(string)] {
			response.Error(c, errors.New("FORBIDDEN"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func JWT(secret string, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		AuthJWT(secret)(c)
		if c.IsAborted() {
			return
		}
		RequireRoles(allowedRoles...)(c)
	}
}
