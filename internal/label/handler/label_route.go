package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	label := r.Group("/label").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	label.Use(middleware.RequireModule("inventory", h.queries))
	{
		label.POST("/download", h.DownloadLabels)
	}
}
