package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	fiscalInvoices := r.Group("/fiscal-invoices").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		fiscalInvoices.POST("", h.EmitFiscalInvoice)
		fiscalInvoices.GET("", h.ListFiscalInvoicesBySale)
		fiscalInvoices.GET("/:id", h.GetFiscalInvoiceByID)
		fiscalInvoices.POST("/:id/cancel", h.CancelFiscalInvoice)
	}

	companyCertificates := r.Group("/company-certificates").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		companyCertificates.POST("", h.UploadCertificate)
		companyCertificates.DELETE("", h.DeleteCompanyCertificate)
	}

	// Rota de webhook: SEM AuthMiddleware, autenticada pelo segredo
	// compartilhado no header Authorization (validado dentro do handler).
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/plugnotas", h.HandlePlugNotasWebhook)
	}
}
