package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoute(r *gin.RouterGroup) {
	auth := middleware.AuthMiddleware(h.jwtManager, h.blacklist)

	// Leituras que o PDV e as trocas também usam: liberadas para Estoque ou
	// Vendas. Quem entra só por Vendas não vê custo (ver hideCostIfSalesOnly).
	read := r.Group("/product")
	read.Use(auth)
	read.Use(middleware.RequireAnyModule(h.queries, "inventory", "sales"))
	{
		read.GET("/company", h.ListProductsByCompany)
		read.GET("/barcode/:barcode", h.GetProductByBarcode)
		read.GET("/stock", h.ListStockEntries)
		read.GET("/models", h.ListModelVariations)
	}

	product := r.Group("/product")
	product.Use(auth)
	product.Use(middleware.RequireModule("inventory", h.queries))
	{
		// operação do dia a dia de catálogo/estoque -> aberto
		product.POST("", h.CreateProduct)
		product.DELETE("/:id", h.DeleteProduct)
		product.GET("/category/:categoryId", h.ListProductsByCategoryId)
		product.GET("/count", h.CountProducts)
		product.GET("/percentage", h.GetProductsPerformanceSummary)
		product.GET("/top-products", h.GetTop5BestSellingProducts)
		product.GET("/:id", h.GetProductById)
		product.PUT("/:id", h.UpdateProduct)
		product.POST("/:id/stock-movements", h.CreateStockMovement)

		// grade de produtos (modelo identificado por ?name=, que pode ter "/")
		product.POST("/models", h.CreateModel)
		product.PUT("/models", h.UpdateModel)
		product.POST("/models/variations", h.AddVariations)
		product.DELETE("/models/variations/:id", h.RemoveVariation)
		product.POST("/models/stock-movements", h.GridStockMovement)
		product.POST("/models/group", h.GroupProducts)

		// custo total do estoque revela custo de compra/margem da empresa -> ADMIN
		admin := product.Group("")
		admin.Use(middleware.RequireRole("ADMIN"))
		{
			admin.GET("/cost-total", h.GetCostTotalStock)
		}
	}
}
