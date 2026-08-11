package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	subscription := r.Group("/subscription").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	// gerenciamento de assinatura/plano da empresa -> ADMIN
	subscription.Use(middleware.RequireRole("ADMIN"))
	{
		subscription.GET("/:id", h.GetSubscriptionById)
		subscription.PUT("/plan/:id", h.UpdateSubscriptionPlan)
		subscription.PUT("/method/:id", h.UpdateSubscriptionMethod)
		subscription.PUT("/cancel/:id", h.CancelSubscription)
	}
}
