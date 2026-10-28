package middleware

import (
	"net/http"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/demo"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func AuthMiddleware(jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			log.Error().Err(err).Msg("invalid token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Só tokens do web e do app; o do painel da plataforma tem middleware próprio
		aud := claims.AudienceValue()
		if !jwt.IsUserAudience(aud) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		isBlacklisted, err := blacklist.IsTokenBlacklisted(c.Request.Context(), claims.TokenKey())
		if err != nil {
			log.Error().Err(err).Msg("failed to check token blacklist")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		if isBlacklisted {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
			c.Abort()
			return
		}

		// Sessão única por usuário em cada aplicação: um login mais recente no
		// web (ou no app) invalida os tokens do login anterior. O usuário demo é
		// compartilhado pelos visitantes e fica de fora.
		if !demo.IsDemoCompany(claims.CompanyId) {
			active, err := blacklist.GetSession(c.Request.Context(), claims.Subject, aud)
			if err != nil {
				log.Error().Err(err).Msg("failed to check session")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				c.Abort()
				return
			}
			if claims.SessionId == "" || active != claims.SessionId {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "sua conta foi acessada em outro dispositivo; entre novamente",
					"code":  "SESSION_REPLACED",
				})
				c.Abort()
				return
			}
		}

		if demo.IsDemoCompany(claims.CompanyId) && demo.IsBlockedAction(c.Request.Method, c.FullPath()) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "ação indisponível no modo demonstração",
				"code":  "DEMO_READONLY",
			})
			c.Abort()
			return
		}

		c.Set("sub", claims.Subject)
		c.Set("company_id", claims.CompanyId)
		c.Set("role", claims.Role)
		c.Set("department_id", claims.DepartmentId)

		c.Next()
	}
}
