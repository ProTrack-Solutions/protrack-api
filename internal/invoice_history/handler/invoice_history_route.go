package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	invoice := r.Group("/invoice-history").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	// faturas da assinatura da empresa no ProTrack (subscription_id/mp_payment_id) ->
	// gerenciamento de assinatura/plano -> ADMIN
	invoice.Use(middleware.RequireRole("ADMIN"))
	{
		invoice.GET("/:id", h.GetInvoiceById)
		invoice.GET("/mp/:mp_payment_id", h.GetInvoiceByMpPaymentId)
		invoice.GET("/company", h.ListInvoiceByCompany)
	}
}
