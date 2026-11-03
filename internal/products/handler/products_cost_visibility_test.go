package handler

import (
	"testing"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/http/middleware"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/gin-gonic/gin"
)

func ctxGranted(module string) *gin.Context {
	c, _ := gin.CreateTestContext(nil)
	c.Set(middleware.GrantedModuleKey, module)
	return c
}

func TestHideCost_SalesOnlyLosesCost(t *testing.T) {
	stock := domain.StockEntriesResponse{TotalValueInStock: 900, Data: []domain.StockEntry{{MinCostPrice: 10, MaxCostPrice: 12, MinSalePrice: 30}}}
	hideCostInStock(ctxGranted("sales"), &stock)
	if stock.TotalValueInStock != 0 || stock.Data[0].MinCostPrice != 0 || stock.Data[0].MaxCostPrice != 0 || stock.Data[0].MinSalePrice != 30 {
		t.Fatalf("vendedor não deveria ver custo: %+v", stock)
	}

	vs := []domain.GridVariation{{CostPrice: 10, SalePrice: 30}}
	hideCostInVariations(ctxGranted("sales"), vs)
	if vs[0].CostPrice != 0 || vs[0].SalePrice != 30 {
		t.Fatalf("variação: %+v", vs[0])
	}

	p := domain.ProductResponse{CostPrice: 10, SalePrice: 30}
	hideCostInProduct(ctxGranted("sales"), &p)
	if p.CostPrice != 0 {
		t.Fatal("produto por código não deveria ter custo")
	}
}

func TestHideCost_InventoryAndAdminKeepCost(t *testing.T) {
	for _, module := range []string{"inventory", "ADMIN"} {
		vs := []domain.GridVariation{{CostPrice: 10}}
		hideCostInVariations(ctxGranted(module), vs)
		if vs[0].CostPrice != 10 {
			t.Fatalf("%s deveria ver custo", module)
		}
	}
}
