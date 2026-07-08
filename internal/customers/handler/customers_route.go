package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	customers := r.Group("/customers").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		customers.POST("", h.CreateCustomer)
		customers.DELETE("/:id", h.DeleteCustomer)
		customers.GET("/cpf/:cpf", h.GetCustomerByCPF)
		customers.GET("/:id", h.GetCustomerById)
		customers.GET("/list", h.ListCustomers)
		customers.PUT("/balanceDue/:id", h.UpdateBalanceDueCustomer)
		customers.PUT("/:id", h.UpdateCustomer)
		customers.GET("/count", h.CountCustomers)
		customers.GET("/percentage", h.GetCustomersPerformanceSummary)
	}
}
