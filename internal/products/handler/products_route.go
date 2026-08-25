package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	product := r.Group("/product")
	product.Use(middleware.AuthMiddleware(h.jwtManager, h.blacklist))
	product.Use(middleware.RequireModule("inventory", h.queries))
	{
		// operação do dia a dia de catálogo/estoque -> aberto
		product.POST("", h.CreateProduct)
		product.DELETE("/:id", h.DeleteProduct)
		product.GET("/barcode/:barcode", h.GetProductByBarcode)
		product.GET("/category/:categoryId", h.ListProductsByCategoryId)
		product.GET("/company", h.ListProductsByCompany)
		product.GET("/count", h.CountProducts)
		product.GET("/percentage", h.GetProductsPerformanceSummary)
		product.GET("/top-products", h.GetTop5BestSellingProducts)
		product.GET("/:id", h.GetProductById)
		product.PUT("/:id", h.UpdateProduct)

		// custo total do estoque revela custo de compra/margem da empresa -> ADMIN
		admin := product.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.GET("/cost-total", h.GetCostTotalStock)
		}
	}
}
