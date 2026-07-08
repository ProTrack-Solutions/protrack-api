package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	reports := r.Group("/reports").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		reports.GET("", h.GenerateReports)
	}
}
