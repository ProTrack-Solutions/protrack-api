package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	purchases := r.Group("/purchases")
	purchases.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	purchases.Use(middleware.RequireModule("purchases", h.queries))
	{
		purchases.GET("/orders", h.ListOrders)
		purchases.POST("/orders", h.CreateOrder)
		purchases.GET("/orders/:id", h.GetOrder)
		purchases.PUT("/orders/:id", h.UpdateOrder)
		purchases.POST("/orders/:id/send", h.SendOrder)
		purchases.POST("/orders/:id/close", h.CloseOrder)
		purchases.POST("/orders/:id/cancel", h.CancelOrder)
		purchases.POST("/orders/:id/receipts", h.Receive)
		purchases.POST("/orders/:id/email", h.EmailOrder)
		purchases.POST("/receipts/:id/reverse", h.ReverseReceipt)

		purchases.GET("/catalog/products", h.SearchCatalog)
		purchases.POST("/catalog/products", h.CreateVariations)
		purchases.GET("/catalog/models", h.ModelVariations)
		purchases.GET("/catalog/categories", h.ListCategories)
	}
}
