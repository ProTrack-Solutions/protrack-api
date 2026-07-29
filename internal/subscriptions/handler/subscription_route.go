package handler

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	subscription := r.Group("/subscription")
	{
		subscription.GET("/:id", h.GetSubscriptionById)
		subscription.PUT("/plan/:id", h.UpdateSubscriptionPlan)
		subscription.PUT("/method/:id", h.UpdateSubscriptionMethod)
		subscription.PUT("/status/:id", h.UpdateSubscriptionStatus)
		subscription.PUT("/cancel/:id", h.CancelSubscription)
	}
}
