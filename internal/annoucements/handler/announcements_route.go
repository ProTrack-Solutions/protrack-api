package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	announcements := r.Group("/announcements").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		announcements.POST("", h.CreateAnnoucements)
		announcements.GET("", h.ListAnnoucements)
		announcements.DELETE("/:id", h.DeleteAnnoucements)
	}
}
