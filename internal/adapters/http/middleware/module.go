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

// GrantedModuleKey guarda no contexto por qual módulo o acesso foi liberado
// ("ADMIN" para administradores), para o handler ajustar o que expõe.
const GrantedModuleKey = "granted_module"

// RequireAnyModule libera se o departamento tiver qualquer um dos módulos, na
// ordem informada, e grava em GrantedModuleKey o primeiro que liberou.
// ADMIN sempre passa. Deve vir depois do AuthMiddleware.
func RequireAnyModule(queries domain.ServiceInterface, moduleCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role == "ADMIN" {
			c.Set(GrantedModuleKey, "ADMIN")
			c.Next()
			return
		}

		departmentID, ok := c.Get("department_id")
		depID, okType := departmentID.(uuid.UUID)
		if !ok || !okType || depID == uuid.Nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "user has no department assigned"})
			c.Abort()
			return
		}

		for _, code := range moduleCodes {
			hasAccess, err := queries.DepartmentHasModule(c.Request.Context(), depID, code)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check module access"})
				c.Abort()
				return
			}
			if hasAccess {
				c.Set(GrantedModuleKey, code)
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient module permissions"})
		c.Abort()
	}
}
