package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{

		protected.GET("/:id", h.GetUserById)
		protected.PUT("/:id", h.UpdateUser)
		protected.DELETE("/:id", h.DeleteUser)
	}

	users := r.Group("/users")
	{
		users.POST("", h.CreateUser)
		users.PUT("/password", h.UpdatePasswordHash)
		users.GET("/email/:email", h.GetUserByEmail)
	}
}
