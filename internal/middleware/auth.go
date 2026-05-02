package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go-be-mono-commerce/internal/auth"
	"go-be-mono-commerce/pkg/response"
)

func JWT(secret string, allowedRoles ...string) gin.HandlerFunc {
	roleAllowed := map[string]bool{}
	for _, r := range allowedRoles { roleAllowed[r] = true }
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") { response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil); c.Abort(); return }
		claims, err := auth.Parse(secret, strings.TrimPrefix(h, "Bearer "))
		if err != nil { response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil); c.Abort(); return }
		if len(roleAllowed) > 0 && !roleAllowed[claims.Role] { response.Fail(c, 403, "Forbidden", "FORBIDDEN", nil); c.Abort(); return }
		c.Set("user_id", claims.UserID); c.Set("role", claims.Role); c.Next()
	}
}
