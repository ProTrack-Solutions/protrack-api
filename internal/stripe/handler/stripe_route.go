package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	stripe := r.Group("/stripe")
	{
		stripe.POST("/webhook", h.SyncSubscriptionWebhook)
	}

	authorized := stripe.Group("").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		authorized.POST("/payment-methods", h.AddPaymentMethod)
		authorized.PUT("/payment-methods/:id/default", h.UpdateDefaultPaymentMethod)
		authorized.PUT("/subscription/cancel", h.CancelSubscription)
	}
}
