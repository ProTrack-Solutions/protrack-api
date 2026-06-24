package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	accountsReceivable := r.Group("/accounts-receivable").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		accountsReceivable.GET("", h.GetCustomerDebtSummary)
		accountsReceivable.GET("/list", h.ListOverdueReceivables)
		accountsReceivable.GET("/customer/:customerId", h.GetPendingReceivablesByCustomer)
		accountsReceivable.GET("/sale/:saleId", h.GetReceivablesBySale)
		accountsReceivable.GET("/total-pending", h.GetTotalOpenAmountByCompany)
		accountsReceivable.GET("/total-overdue", h.GetTotalOverdueAmountByCompany)
		accountsReceivable.GET("/total-pending-overdue", h.GetTotalPendingAndOverdue)
	}
}
