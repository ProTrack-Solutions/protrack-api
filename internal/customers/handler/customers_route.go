package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	auth := middleware.AuthMiddleware(h.jwtManager, h.blacklist)

	// O PDV e as trocas escolhem o cliente da venda: a listagem também vale para Vendas
	read := r.Group("/customers").Use(auth)
	read.Use(middleware.RequireAnyModule(h.queries, "customers", "sales"))
	{
		read.GET("/list", h.ListCustomers)
	}

	customers := r.Group("/customers").Use(auth)
	customers.Use(middleware.RequireModule("customers", h.queries))
	{
		customers.POST("", h.CreateCustomer)
		customers.DELETE("/:id", h.DeleteCustomer)
		customers.GET("/cpf/:cpf", h.GetCustomerByCPF)
		customers.GET("/:id", h.GetCustomerById)
		customers.PUT("/balanceDue/:id", h.UpdateBalanceDueCustomer)
		customers.PUT("/:id", h.UpdateCustomer)
		customers.GET("/count", h.CountCustomers)
		customers.GET("/percentage", h.GetCustomersPerformanceSummary)
	}
}
