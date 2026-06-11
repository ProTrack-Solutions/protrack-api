package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	cashFlow := r.Group("/cash-flow").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		cashFlow.GET("/summary", h.CashFlowSummary)
		cashFlow.GET("/history-projection", h.GetCashFlowHistoryProjections)
		cashFlow.GET("/inflow-category", h.GetCashInFlowByCategory)
		cashFlow.GET("/outflow-category", h.GetCashOutFlowByCategory)
		cashFlow.GET("/summary-month", h.GetCashFlowPeriod)
	}
}
