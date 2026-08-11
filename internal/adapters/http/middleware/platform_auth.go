package middleware

import (
	"fmt"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/gin-gonic/gin"
)

func PlatformAuthMiddleware(jwtManager *jwt.JWTManager, blacklist *cache.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("platform_access_token")
		if err != nil || tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		tokenId := fmt.Sprintf("%s:%d", claims.Subject, claims.ExpiresAt.Unix())
		isBlacklisted, err := blacklist.IsTokenBlacklisted(c.Request.Context(), tokenId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			c.Abort()
			return
		}
		if isBlacklisted {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
			c.Abort()
			return
		}

		c.Set("sub", claims.Subject)
		c.Set("role", claims.Role)
		c.Next()
	}
}
