package handler

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/gin-gonic/gin"
)

// salesOnly: o acesso às leituras compartilhadas veio só pelo módulo Vendas.
// Vendedor enxerga preço de venda e estoque, mas não custo nem margem.
func salesOnly(c *gin.Context) bool {
	return c.GetString(middleware.GrantedModuleKey) == "sales"
}

func hideCostInList(c *gin.Context, resp *domain.ProductPaginatedResponse) {
	if !salesOnly(c) {
		return
	}
	resp.TotalValueInStock = 0
	for i := range resp.Data {
		resp.Data[i].CostPrice = 0
	}
}

func hideCostInProduct(c *gin.Context, p *domain.ProductResponse) {
	if salesOnly(c) {
		p.CostPrice = 0
	}
}

func hideCostInStock(c *gin.Context, resp *domain.StockEntriesResponse) {
	if !salesOnly(c) {
		return
	}
	resp.TotalValueInStock = 0
	for i := range resp.Data {
		resp.Data[i].MinCostPrice = 0
		resp.Data[i].MaxCostPrice = 0
	}
}

func hideCostInVariations(c *gin.Context, vs []domain.GridVariation) {
	if !salesOnly(c) {
		return
	}
	for i := range vs {
		vs[i].CostPrice = 0
	}
}
