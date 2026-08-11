package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	announcements := r.Group("/announcements")
	announcements.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		// leitura: ADMIN e USER podem ver os avisos da empresa
		announcements.GET("", h.ListAnnoucements)

		// criar/remover aviso afeta a empresa toda -> só ADMIN
		admin := announcements.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.POST("", h.CreateAnnoucements)
			admin.DELETE("/:id", h.DeleteAnnoucements)
		}
	}
}
