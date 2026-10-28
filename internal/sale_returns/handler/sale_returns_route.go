package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	returns := r.Group("/returns")
	returns.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	returns.Use(middleware.RequireModule("sales", h.queries))
	{
		returns.GET("", h.List)
		returns.POST("", h.Create)
		returns.GET("/:id", h.Get)
		returns.GET("/sales", h.SearchSales)
		returns.GET("/sales/:id", h.ReturnableSale)
		returns.GET("/customers/:id/credit", h.CustomerCredit)
	}
}
