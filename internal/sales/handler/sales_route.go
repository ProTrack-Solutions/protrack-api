package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	sales := r.Group("/sales").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		sales.POST("", h.CreateSale)
		sales.DELETE("/:id", h.DeleteSale)
		sales.GET("/:id", h.GetSaleById)
		sales.GET("/list/", h.ListSales)
		sales.GET("/list/company", h.ListSalesByCompanyAndStatus)
		sales.GET("/count", h.CountSales)
		sales.GET("/percentage", h.GetSalesPerformanceSummary)
		sales.GET("/total-amount", h.GetTotalAmountSummary)
		sales.GET("/total-pending", h.GetTotalAmountIsPending)
		sales.GET("/total-overdue", h.GetTotalAmountIsOverdue)
		sales.GET("/count/pending-overdue", h.ContSalesPendingAndOverdue)
		sales.GET("/complete", h.ListSalesWithDetails)
		sales.GET("/complete/pending-overdue", h.ListSalesWithDetailsPendingOverdue)
		sales.GET("/real-profit", h.GetRealProfitItem)
		sales.GET("/top5-products", h.GetTop5RealProfitItem)
		sales.GET("/performance-mounts", h.GetPerformanceMonth)
		sales.GET("/investment-categories", h.GetTotalInvestmentCategory)
		sales.GET("/margin-distribution", h.MarginDistribution)
		sales.PUT("/:saleId", h.UpdateSale)
		sales.GET("/stock-turnover", h.GetInventoryTurnover)
	}
}
