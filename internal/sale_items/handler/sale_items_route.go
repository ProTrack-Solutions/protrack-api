package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	saleItems := r.Group("/sale-items").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		saleItems.DELETE("/:id", h.DeleteSaleItem)
		saleItems.DELETE("/sale/:saleId", h.DeleteItemsBySale)
	}
}
