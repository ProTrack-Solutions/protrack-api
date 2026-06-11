package handler

import (
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	category := r.Group("/products-categories").Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	{
		category.POST("", h.CreateProductCategory)
		category.DELETE("/:id", h.DeleteProductCategory)
		category.GET("/:id", h.GetProductCategoryById)
		category.GET("/list/company", h.ListProductCategoryByCompanyId)
		category.PUT("/status/", h.SetProductCategoryStatus)
		category.PUT("/:id", h.UpdateProductCategory)
	}
}
