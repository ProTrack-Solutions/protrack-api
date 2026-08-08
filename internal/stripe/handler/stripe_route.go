package handler

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	stripe := r.Group("/stripe")
	{
		stripe.POST("/webhook", h.SyncSubscriptionWebhook)
	}
}
