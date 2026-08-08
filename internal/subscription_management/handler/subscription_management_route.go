package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	authorized := r.Group("/subscription-management").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		authorized.GET("/subscription", h.GetSubscriptionDetails)
		authorized.POST("/payment-methods", h.AddPaymentMethod)
		authorized.PUT("/payment-methods/:id/default", h.UpdateDefaultPaymentMethod)
		authorized.PUT("/subscription/cancel", h.CancelSubscription)
	}
}
