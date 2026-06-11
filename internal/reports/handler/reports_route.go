package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	reports := r.Group("/reports").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		reports.GET("", h.GenerateReports)
	}
}
