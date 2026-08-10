package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	protected := r.Group("/user")
	protected.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{

		protected.GET("/:id", h.GetUserById)
		protected.PUT("/:id", h.UpdateUser)
		protected.DELETE("/:id", h.DeleteUser)
		protected.PUT("/password", h.UpdatePasswordHash)
	}
}
