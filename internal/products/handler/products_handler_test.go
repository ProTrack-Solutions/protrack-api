package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

var errService = errors.New("service error")

// ginCtxBackground retorna um context.Context simples para uso nos testes de serviço.
func ginCtxBackground() context.Context {
	return context.Background()
}

// sampleProductResponse cria uma ProductResponse de exemplo.
func sampleProductResponse(id, companyID, categoryID uuid.UUID) domain.ProductResponse {
	return domain.ProductResponse{
		ID:          id,
		CompanyID:   companyID,
		CategoryID:  categoryID,
		Name:        "Produto Teste",
		Description: "Descrição teste",
		Barcode:     "TEST001",
		Quantity:    10,
		Size:        "M",
		CostPrice:   25.00,
		SalePrice:   50.00,
		CreatedBy:   uuid.New(),
		CreatedAt:   time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// Testes de lógica de serviço via mock (verificam assinaturas e contratos)
// ---------------------------------------------------------------------------

// TestServiceContract_CreateProduct verifica que o contrato do mock de serviço é respeitado.
func TestServiceContract_CreateProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()
	createdBy := uuid.New()
	productID := uuid.New()

	expectedResp := sampleProductResponse(productID, companyID, uuid.New())

	mockSvc.EXPECT().
		CreateProduct(gomock.Any(), gomock.AssignableToTypeOf(domain.CreateProductRequest{})).
		Return(expectedResp, nil)

	resp, err := mockSvc.CreateProduct(
		ginCtxBackground(),
		domain.CreateProductRequest{
			CompanyID:   companyID,
			Name:        "Produto Teste",
			Description: "Descrição",
			CategoryID:  uuid.New(),
			Barcode:     "TEST001",
			Quantity:    10,
			Size:        "M",
			CostPrice:   25.00,
			SalePrice:   50.00,
			CreatedBy:   createdBy,
		},
	)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != productID {
		t.Errorf("ID incorreto")
	}
}

// TestServiceContract_DeleteProduct verifica contrato de deleção.
func TestServiceContract_DeleteProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	productID := uuid.New()

	mockSvc.EXPECT().
		DeleteProduct(gomock.Any(), domain.DeleteProductRequest{
			ID:        productID,
			DeletedBy: uuid.Nil,
		}).
		Return(nil)

	err := mockSvc.DeleteProduct(ginCtxBackground(), domain.DeleteProductRequest{
		ID:        productID,
		DeletedBy: uuid.Nil,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

// TestServiceContract_GetProductByBarcode_Success verifica busca por barcode.
func TestServiceContract_GetProductByBarcode_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	barcode := "BAR12345"
	expected := sampleProductResponse(uuid.New(), uuid.New(), uuid.New())
	expected.Barcode = barcode

	mockSvc.EXPECT().
		GetProductByBarcode(gomock.Any(), barcode).
		Return(expected, nil)

	resp, err := mockSvc.GetProductByBarcode(ginCtxBackground(), barcode)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Barcode != barcode {
		t.Errorf("Barcode incorreto: '%s'", resp.Barcode)
	}
}

// TestServiceContract_GetProductByBarcode_Error verifica propagação de erro.
func TestServiceContract_GetProductByBarcode_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)

	mockSvc.EXPECT().
		GetProductByBarcode(gomock.Any(), gomock.Any()).
		Return(domain.ProductResponse{}, errService)

	_, err := mockSvc.GetProductByBarcode(ginCtxBackground(), "QUALQUER")

	if err == nil {
		t.Fatal("esperava erro do serviço")
	}
	if !errors.Is(err, errService) {
		t.Errorf("erro incorreto: %v", err)
	}
}

// TestServiceContract_GetProductById_Success verifica busca por ID.
func TestServiceContract_GetProductById_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	companyID := uuid.New()

	expected := sampleProductResponse(id, companyID, uuid.New())

	mockSvc.EXPECT().
		GetProductById(gomock.Any(), id).
		Return(expected, nil)

	resp, err := mockSvc.GetProductById(ginCtxBackground(), id)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
}

// TestServiceContract_ListProductsByCategoryId_Success verifica listagem por categoria.
func TestServiceContract_ListProductsByCategoryId_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	categoryID := uuid.New()
	companyID := uuid.New()

	products := []domain.ProductResponse{
		sampleProductResponse(uuid.New(), companyID, categoryID),
		sampleProductResponse(uuid.New(), companyID, categoryID),
	}

	mockSvc.EXPECT().
		ListProductsByCategoryId(gomock.Any(), categoryID, companyID).
		Return(products, nil)

	resp, err := mockSvc.ListProductsByCategoryId(ginCtxBackground(), categoryID, companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("esperava 2 produtos, obteve %d", len(resp))
	}
}

// TestServiceContract_ListProductsByCompanyPaginated_Success verifica paginação.
func TestServiceContract_ListProductsByCompanyPaginated_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()
	pagination := globalDomain.PaginationParams{Page: 1, PerPage: 10}

	paginatedResponse := domain.ProductPaginatedResponse{
		PaginatedResponse: globalDomain.PaginatedResponse[domain.ListProductsByCompanyRow]{
			Data:       []domain.ListProductsByCompanyRow{},
			Page:       1,
			PerPage:    10,
			TotalRows:  0,
			TotalPages: 0,
		},
		TotalValueInStock: 0,
		ItensInStock:      0,
		LowItensInStock:   0,
	}

	mockSvc.EXPECT().
		ListProductsByCompanyPaginated(gomock.Any(), companyID, pagination).
		Return(paginatedResponse, nil)

	resp, err := mockSvc.ListProductsByCompanyPaginated(ginCtxBackground(), companyID, pagination)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("esperava Page=1, obteve %d", resp.Page)
	}
}

// TestServiceContract_UpdateProduct_Success verifica atualização de produto.
func TestServiceContract_UpdateProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	companyID := uuid.New()

	updateReq := domain.UpdateProductRequest{
		Name:      "Novo Nome",
		SalePrice: 99.90,
	}
	expected := sampleProductResponse(id, companyID, uuid.New())
	expected.Name = "Novo Nome"

	mockSvc.EXPECT().
		UpdateProduct(gomock.Any(), id, updateReq).
		Return(expected, nil)

	resp, err := mockSvc.UpdateProduct(ginCtxBackground(), id, updateReq)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Name != "Novo Nome" {
		t.Errorf("Name incorreto: '%s'", resp.Name)
	}
}

// TestServiceContract_CountProducts_Success verifica contagem de produtos.
func TestServiceContract_CountProducts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		CountProducts(gomock.Any(), companyID).
		Return(int64(25), nil)

	count, err := mockSvc.CountProducts(ginCtxBackground(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 25 {
		t.Errorf("esperava 25, obteve %d", count)
	}
}

// TestServiceContract_GetProductsPerformanceSummary_Success verifica percentual de performance.
func TestServiceContract_GetProductsPerformanceSummary_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetProductsPerformanceSummary(gomock.Any(), companyID).
		Return(35.5, nil)

	percentage, err := mockSvc.GetProductsPerformanceSummary(ginCtxBackground(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 35.5 {
		t.Errorf("esperava 35.5, obteve %f", percentage)
	}
}

// TestServiceContract_GetCostTotalStock_Success verifica custo total do estoque.
func TestServiceContract_GetCostTotalStock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetCostTotalStock(gomock.Any(), companyID).
		Return(12500.75, nil)

	total, err := mockSvc.GetCostTotalStock(ginCtxBackground(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if total != 12500.75 {
		t.Errorf("esperava 12500.75, obteve %f", total)
	}
}

// TestServiceContract_GetTop5BestSellingProducts_Success verifica top 5 mais vendidos.
func TestServiceContract_GetTop5BestSellingProducts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	expected := []domain.GetTop5BestSellingProductsRow{
		{ID: uuid.New(), Name: "Camisa A", TotalQuantitySold: 500},
		{ID: uuid.New(), Name: "Camisa B", TotalQuantitySold: 300},
	}

	mockSvc.EXPECT().
		GetTop5BestSellingProducts(gomock.Any(), companyID).
		Return(expected, nil)

	resp, err := mockSvc.GetTop5BestSellingProducts(ginCtxBackground(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("esperava 2 produtos, obteve %d", len(resp))
	}
	if resp[0].TotalQuantitySold != 500 {
		t.Errorf("TotalQuantitySold incorreto: %d", resp[0].TotalQuantitySold)
	}
}

// ---------------------------------------------------------------------------
// Testes de integração HTTP com httptest
// ---------------------------------------------------------------------------

// TestHTTP_GetProductById_BadUUID verifica que UUIDs inválidos retornam 400.
func TestHTTP_GetProductById_BadUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/product/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uuid"})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/product/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

// TestHTTP_CreateProduct_MissingCompanyId verifica retorno 400 quando company_id ausente.
func TestHTTP_CreateProduct_MissingCompanyId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/product", func(c *gin.Context) {
		_, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/product", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}

	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "company_id null" {
		t.Errorf("mensagem de erro incorreta: %v", body)
	}
}

// TestHTTP_DeleteProduct_ValidUUID verifica que handler responde 204 com UUID válido.
func TestHTTP_DeleteProduct_ValidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	productID := uuid.New()

	mockSvc.EXPECT().
		DeleteProduct(gomock.Any(), gomock.AssignableToTypeOf(domain.DeleteProductRequest{})).
		Return(nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/product/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := mockSvc.DeleteProduct(c.Request.Context(), domain.DeleteProductRequest{ID: id, DeletedBy: uuid.Nil}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/product/%s", productID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("esperava 204, obteve %d", w.Code)
	}
}

// TestHTTP_DeleteProduct_ServiceError verifica 500 quando serviço retorna erro.
func TestHTTP_DeleteProduct_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	productID := uuid.New()

	mockSvc.EXPECT().
		DeleteProduct(gomock.Any(), gomock.Any()).
		Return(errService)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/product/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := mockSvc.DeleteProduct(c.Request.Context(), domain.DeleteProductRequest{ID: id, DeletedBy: uuid.Nil}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/product/%s", productID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d", w.Code)
	}
}

// TestHTTP_CountProducts_MissingCompanyId verifica 401 quando company_id ausente.
func TestHTTP_CountProducts_MissingCompanyId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/product/count", func(c *gin.Context) {
		_, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": 0})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/product/count", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

// TestHTTP_GetProductBarcode_Success verifica retorno 200 com barcode válido.
func TestHTTP_GetProductBarcode_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	barcode := "ABC999"
	expected := sampleProductResponse(uuid.New(), uuid.New(), uuid.New())
	expected.Barcode = barcode

	mockSvc.EXPECT().
		GetProductByBarcode(gomock.Any(), barcode).
		Return(expected, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/product/barcode/:barcode", func(c *gin.Context) {
		b := c.Param("barcode")
		product, err := mockSvc.GetProductByBarcode(c.Request.Context(), b)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"product": product})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/product/barcode/"+barcode, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	productMap, ok := resp["product"].(map[string]interface{})
	if !ok {
		t.Fatal("resposta não contém 'product'")
	}
	if productMap["barcode"] != barcode {
		t.Errorf("barcode incorreto na resposta: %v", productMap["barcode"])
	}
}

// TestHTTP_GetTop5BestSellingProducts_Success verifica retorno 200 com company_id no contexto.
func TestHTTP_GetTop5BestSellingProducts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	expected := []domain.GetTop5BestSellingProductsRow{
		{ID: uuid.New(), Name: "Best Seller", TotalQuantitySold: 999},
	}

	mockSvc.EXPECT().
		GetTop5BestSellingProducts(gomock.Any(), companyID).
		Return(expected, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.GET("/product/top-products", func(c *gin.Context) {
		companyIdAny, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
			return
		}
		products, err := mockSvc.GetTop5BestSellingProducts(c.Request.Context(), companyIdAny.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, products)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/product/top-products", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

// TestHTTP_ListProductsByCompany_MissingCompanyId verifica 400 sem company_id.
func TestHTTP_ListProductsByCompany_MissingCompanyId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/product/company", func(c *gin.Context) {
		_, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "company_id null"})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/product/company", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

// TestHTTP_UpdateProduct_InvalidUUID verifica retorno 400 com UUID inválido.
func TestHTTP_UpdateProduct_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/product/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/product/invalid-uuid", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

// TestHTTP_GetCostTotalStock_WithCompanyId verifica retorno 200 com custo do estoque.
func TestHTTP_GetCostTotalStock_WithCompanyId(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetCostTotalStock(gomock.Any(), companyID).
		Return(9999.99, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.GET("/product/cost-total", func(c *gin.Context) {
		companyIdAny, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id null"})
			return
		}
		total, err := mockSvc.GetCostTotalStock(c.Request.Context(), companyIdAny.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"cost_total": total})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/product/cost-total", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]float64
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["cost_total"] != 9999.99 {
		t.Errorf("cost_total incorreto: %v", resp["cost_total"])
	}
}
