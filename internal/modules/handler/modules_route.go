package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	modules := r.Group("/modules").Use(middleware.RequireRole("SUPER_ADMIN"))
	{
		modules.GET("/list", h.ListModules)
		modules.GET("/:code", h.GetModule)
	}
}
