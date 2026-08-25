package middleware

import (
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// RequireModule permite acesso apenas se o departamento do usuário no contexto
// (setado pelo AuthMiddleware) tiver o módulo informado liberado.
// ADMIN sempre passa direto, independente de departamento.
// Deve ser usado SEMPRE depois do AuthMiddleware na cadeia de middlewares.
func RequireModule(moduleCode string, queries domain.ServiceInterface) gin.HandlerFunc {
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

		if role == "ADMIN" {
			c.Next()
			return
		}

		departmentIDAny, exists := c.Get("department_id")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "department not found in context"})
			c.Abort()
			return
		}

		log.Debug().Any("departmentIDAny", departmentIDAny).Msg("departmentIDAny")

		departmentID, ok := departmentIDAny.(uuid.UUID)
		if !ok || departmentID == uuid.Nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "user has no department assigned"})
			c.Abort()
			return
		}

		hasAccess, err := queries.DepartmentHasModule(c.Request.Context(), departmentID, moduleCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check module access"})
			c.Abort()
			return
		}
		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient module permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}
