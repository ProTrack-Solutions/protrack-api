package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var errService = errors.New("service error")

func ginCtx() context.Context { return context.Background() }

func sampleGetSaleByIdRow(id, companyID uuid.UUID) domain.GetSaleByIdRow {
	return domain.GetSaleByIdRow{
		ID:             id,
		CustomerID:     uuid.New(),
		CompanyID:      companyID,
		SaleAt:         time.Now().UTC(),
		DiscountAmount: 0,
		Subtotal:       100.0,
		TotalAmount:    100.0,
		DueDays:        30,
		PaymentMethod:  "paid",
		Status:         "paid",
		CreatedAt:      time.Now().UTC(),
		CreatedBy:      uuid.New(),
		CustomerName:   "Cliente Venda",
	}
}

// ---------------------------------------------------------------------------
// Testes de Contrato de Serviço via Mock (Sales)
// ---------------------------------------------------------------------------

func TestServiceContract_DeleteSale(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	req := domain.DeleteSaleRequest{
		DeletedBy: uuid.New(),
		ID:        id,
		CompanyID: uuid.New(),
	}

	mockSvc.EXPECT().
		DeleteSale(gomock.Any(), id, req).
		Return(nil)

	err := mockSvc.DeleteSale(ginCtx(), id, req)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestServiceContract_GetSaleById_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	companyID := uuid.New()
	req := domain.GetSaleByIdRequest{
		ID:        id,
		CompanyID: companyID,
	}

	expected := sampleGetSaleByIdRow(id, companyID)

	mockSvc.EXPECT().
		GetSaleById(gomock.Any(), req).
		Return(expected, nil)

	resp, err := mockSvc.GetSaleById(ginCtx(), req)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
}

func TestServiceContract_GetSaleById_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	req := domain.GetSaleByIdRequest{
		ID:        uuid.New(),
		CompanyID: uuid.New(),
	}

	mockSvc.EXPECT().
		GetSaleById(gomock.Any(), req).
		Return(domain.GetSaleByIdRow{}, errService)

	_, err := mockSvc.GetSaleById(ginCtx(), req)
	if err == nil {
		t.Fatal("esperava erro do serviço")
	}
	if !errors.Is(err, errService) {
		t.Errorf("erro incorreto: %v", err)
	}
}

func TestServiceContract_ListSales_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	salesList := []domain.ListSalesRow{
		{ID: uuid.New(), TotalAmount: 150.0, Status: "paid"},
		{ID: uuid.New(), TotalAmount: 200.0, Status: "pending"},
	}

	mockSvc.EXPECT().
		ListSales(gomock.Any(), companyID).
		Return(salesList, nil)

	resp, err := mockSvc.ListSales(ginCtx(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("esperava 2 vendas, obteve %d", len(resp))
	}
}

func TestServiceContract_CountSales_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		CountSales(gomock.Any(), companyID).
		Return(int64(15), nil)

	count, err := mockSvc.CountSales(ginCtx(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 15 {
		t.Errorf("esperava 15, obteve %d", count)
	}
}

func TestServiceContract_GetSalesPerformanceSummary_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetSalesPerformanceSummary(gomock.Any(), companyID).
		Return(50.0, nil)

	perc, err := mockSvc.GetSalesPerformanceSummary(ginCtx(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if perc != 50.0 {
		t.Errorf("esperava 50.0, obteve %f", perc)
	}
}

func TestServiceContract_GetTotalAmountIsPending_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetTotalAmountIsPending(gomock.Any(), companyID).
		Return(300.50, nil)

	total, err := mockSvc.GetTotalAmountIsPending(ginCtx(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if total != 300.50 {
		t.Errorf("esperava 300.50, obteve %f", total)
	}
}

// ---------------------------------------------------------------------------
// Testes de integração HTTP com httptest + gin (Sales)
// ---------------------------------------------------------------------------

func TestHTTP_DeleteSale_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/sales/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/sales/nao-e-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

func TestHTTP_DeleteSale_ValidUUID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	saleID := uuid.New()
	companyID := uuid.New()
	userID := uuid.New()

	mockSvc.EXPECT().
		DeleteSale(gomock.Any(), saleID, domain.DeleteSaleRequest{
			CompanyID: companyID,
			DeletedBy: userID,
		}).
		Return(nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Set("sub", userID.String())
		c.Next()
	})
	r.DELETE("/sales/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		companyIdAny, _ := c.Get("company_id")
		userIdAny, _ := c.Get("sub")

		companyId := companyIdAny.(uuid.UUID)
		userId, _ := uuid.Parse(userIdAny.(string))

		var deleteReq domain.DeleteSaleRequest
		deleteReq.CompanyID = companyId
		deleteReq.DeletedBy = userId

		if err := mockSvc.DeleteSale(c.Request.Context(), id, deleteReq); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/sales/%s", saleID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("esperava 204, obteve %d", w.Code)
	}
}

func TestHTTP_GetSaleById_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/sales/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sales/invalid-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

func TestHTTP_GetSaleById_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	saleID := uuid.New()
	companyID := uuid.New()

	expected := sampleGetSaleByIdRow(saleID, companyID)

	mockSvc.EXPECT().
		GetSaleById(gomock.Any(), domain.GetSaleByIdRequest{
			ID:        saleID,
			CompanyID: companyID,
		}).
		Return(expected, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.GET("/sales/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		companyIdAny, _ := c.Get("company_id")
		companyId := companyIdAny.(uuid.UUID)

		sale, err := mockSvc.GetSaleById(c.Request.Context(), domain.GetSaleByIdRequest{
			ID:        id,
			CompanyID: companyId,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sale": sale})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/sales/%s", saleID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&body)
	saleMap, ok := body["sale"].(map[string]interface{})
	if !ok {
		t.Fatal("resposta não contém 'sale'")
	}
	if saleMap["id"] != saleID.String() {
		t.Errorf("ID da venda incorreto: %v", saleMap["id"])
	}
}

func TestHTTP_CountSales_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		CountSales(gomock.Any(), companyID).
		Return(int64(42), nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.GET("/sales/count", func(c *gin.Context) {
		companyIdAny, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
			return
		}
		count, err := mockSvc.CountSales(c.Request.Context(), companyIdAny.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": count})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sales/count", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if int(body["count"].(float64)) != 42 {
		t.Errorf("count incorreto na resposta: %v", body["count"])
	}
}

func TestHTTP_GetSalesPerformanceSummary_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetSalesPerformanceSummary(gomock.Any(), companyID).
		Return(75.5, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.GET("/sales/percentage", func(c *gin.Context) {
		companyIdAny, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "company_id is null"})
			return
		}
		percentage, err := mockSvc.GetSalesPerformanceSummary(c.Request.Context(), companyIdAny.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"percentage": percentage})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sales/percentage", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var body map[string]float64
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["percentage"] != 75.5 {
		t.Errorf("percentage incorreto: %v", body["percentage"])
	}
}

func TestHTTP_ListSalesWithDetails_HeaderBinding(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		ListSalesWithDetailsPaginate(gomock.Any(), companyID, gomock.Any()).
		Return(domain.SaleResponsePaginate{
			PaginatedResponse: globalDomain.PaginatedResponse[domain.ListSalesWithInstallmentsResponse]{
				Page:       1,
				PerPage:    10,
				TotalRows:  0,
				TotalPages: 0,
			},
		}, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.GET("/sales/complete", func(c *gin.Context) {
		companyIdAny, _ := c.Get("company_id")
		companyId := companyIdAny.(uuid.UUID)

		var pagination globalDomain.PaginationParams
		if err := c.ShouldBindHeader(&pagination); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sales, err := mockSvc.ListSalesWithDetailsPaginate(c.Request.Context(), companyId, pagination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, sales)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sales/complete", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}
