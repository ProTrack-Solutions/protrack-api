package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		protected.GET("/me", h.GetUserFromContext)
		protected.POST("/logout", h.Logout)
	}

	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)

	}
}
