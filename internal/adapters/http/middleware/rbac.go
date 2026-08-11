package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole permite acesso apenas se a role no contexto (setada pelo
// AuthMiddleware) estiver entre as roles permitidas. Deve ser usado
// SEMPRE depois do AuthMiddleware na cadeia de middlewares.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		roleAny, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "role not found in context"})
			c.Abort()
			return
		}

		role, ok := roleAny.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid role type in context"})
			c.Abort()
			return
		}

		if _, ok := allowed[role]; !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}
