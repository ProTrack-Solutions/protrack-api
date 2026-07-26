package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	paymentMethods := r.Group("/subscription-payment-methods").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		paymentMethods.POST("", h.CreateSubscriptionPaymentMethodHandler)
		paymentMethods.GET("", h.GetSubscriptionPaymentMethodByCompanyId)
		paymentMethods.GET("/:id", h.GetSubscriptionPaymentMethodById)
		paymentMethods.PUT("/:id", h.UpdateSubscriptionPaymentMethodHandler)
		paymentMethods.PATCH("/:id", h.SetDefaultSubscriptionPaymentMethod)
		paymentMethods.DELETE("/:id", h.DeleteSubscriptionPaymentMethod)
	}
}
