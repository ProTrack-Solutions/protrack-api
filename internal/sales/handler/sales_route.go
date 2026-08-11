package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	sales := r.Group("/sales")
	sales.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		// operação do dia a dia de vendas -> aberto
		sales.POST("", h.CreateSale)
		sales.DELETE("/:id", h.DeleteSale)
		sales.GET("/:id", h.GetSaleById)
		sales.GET("/list/", h.ListSales)
		sales.GET("/list/company", h.ListSalesByCompanyAndStatus)
		sales.GET("/count", h.CountSales)
		sales.GET("/percentage", h.GetSalesPerformanceSummary)
		sales.GET("/count/pending-overdue", h.ContSalesPendingAndOverdue)
		sales.GET("/complete", h.ListSalesWithDetails)
		sales.GET("/complete/pending-overdue", h.ListSalesWithDetailsPendingOverdue)
		sales.GET("/performance-mounts", h.GetPerformanceMonth)
		sales.PUT("/:saleId", h.UpdateSale)
		sales.GET("/stock-turnover", h.GetInventoryTurnover)

		// totais em R$, lucro real e margem = dado financeiro sensível da empresa -> ADMIN
		admin := sales.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.GET("/total-amount", h.GetTotalAmountSummary)
			admin.GET("/total-pending", h.GetTotalAmountIsPending)
			admin.GET("/total-overdue", h.GetTotalAmountIsOverdue)
			admin.GET("/real-profit", h.GetRealProfitItem)
			admin.GET("/top5-products", h.GetTop5RealProfitItem)
			admin.GET("/investment-categories", h.GetTotalInvestmentCategory)
			admin.GET("/margin-distribution", h.MarginDistribution)
		}
	}
}
