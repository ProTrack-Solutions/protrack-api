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
		protected.PUT("", h.UpdateOwnProfile)

		protected.PUT("/password", h.UpdatePasswordHash)

		admin := protected.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.DELETE("/:id", h.DeleteUser)
			admin.PUT("/:id", h.UpdateUser)
		}

		superadmin := protected.Group("")
		superadmin.Use(middleware.RequireRole("SUPER_ADMIN"))
		{
			superadmin.GET("/list", h.ListUsers)
			superadmin.GET("/count", h.CountUsers)
		}
	}
}
