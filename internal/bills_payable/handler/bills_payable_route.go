package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	billsPayable := r.Group("/bills_payable").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		billsPayable.POST("", h.CreateBillPayable)
		billsPayable.GET("/:id", h.GetBillsPayableById)
		billsPayable.GET("/status/:status", h.GetBillsByStatus)
		billsPayable.GET("/overdue", h.GetOverdueBills)
		billsPayable.GET("/list", h.ListBillsPayable)
		billsPayable.PUT("/pay/:id", h.PayBill)
		billsPayable.PUT("/schedule/:id", h.ScheduleBill)
		billsPayable.PUT("/:id", h.UpdateBillPayable)
		billsPayable.GET("/summary", h.GetBillsPayableSummary)
	}
}
