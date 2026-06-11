package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	payments := r.Group("/payments").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		payments.POST("", h.NewPayment)
	}
}
