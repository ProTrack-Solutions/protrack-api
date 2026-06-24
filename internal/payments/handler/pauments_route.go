package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	payments := r.Group("/payments").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		payments.POST("", h.NewPayment)
	}
}
