package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	paymentHistory := r.Group("/payment-history")
	paymentHistory.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		// operação do dia a dia (registrar/consultar pagamento de uma venda) -> aberto
		paymentHistory.POST("", h.CreatePaymentHistory)
		paymentHistory.GET("", h.ListPaymentHistory)
		paymentHistory.GET("/customer/:customerId", h.GetPaymentsByCustomer)
		paymentHistory.GET("/sale/:saleId", h.GetPaymentsBySale)

		// total recebido no período = dado financeiro agregado da empresa -> ADMIN
		admin := paymentHistory.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.GET("/total", h.GetTotalReceivedByPeriod)
		}
		// paymentHistory.GET("/report", h.ExportExcel)
	}
}
