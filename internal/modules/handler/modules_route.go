package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	modules := r.Group("/modules")
	modules.Use(middleware.AuthMiddleware(h.jwt, h.blackList))

	admin := modules.Use(middleware.RequireRole("ADMIN"))
	{
		admin.GET("/list", h.ListModules)
		admin.GET("/:code", h.GetModule)
	}
}
