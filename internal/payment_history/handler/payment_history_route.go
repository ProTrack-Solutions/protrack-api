package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	paymentHistory := r.Group("/payment-history")
	paymentHistory.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		// operação do dia a dia (registrar/consultar pagamento de uma venda) -> módulo sales
		sales := paymentHistory.Group("")
		sales.Use(middleware.RequireModule("sales", h.queries))
		{
			sales.POST("", h.CreatePaymentHistory)
			sales.GET("", h.ListPaymentHistory)
			sales.GET("/customer/:customerId", h.GetPaymentsByCustomer)
			sales.GET("/sale/:saleId", h.GetPaymentsBySale)
		}

		// total recebido no período = dado financeiro agregado da empresa -> ADMIN + módulo financial
		admin := paymentHistory.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		admin.Use(middleware.RequireModule("financial", h.queries))
		{
			admin.GET("/total", h.GetTotalReceivedByPeriod)
		}
		// paymentHistory.GET("/report", h.ExportExcel)
	}
}
