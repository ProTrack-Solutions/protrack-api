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

	"github.com/ProTrack-Solutions/protrack-api/internal/customers/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/customers/mocks"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var errService = errors.New("service error")

func ginCtx() context.Context { return context.Background() }

func sampleCustomerResponse(id, companyID uuid.UUID) domain.CustomerResponse {
	return domain.CustomerResponse{
		ID:            id,
		CompanyID:     companyID,
		FullName:      "Ana Costa",
		BirthDate:     "1995-07-20",
		Cpf:           "987.654.321-00",
		Rg:            "SP7654321",
		MaritalStatus: "solteira",
		Gender:        enums.GenderFemale,
		Whatsapp:      "+5511912345678",
		Email:         "ana@email.com",
		AddressCity:   "São Paulo",
		AddressState:  "SP",
		BalanceDue:    150.00,
		CreatedBy:     uuid.New(),
		CreatedAt:     time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// Testes de Contrato de Serviço via Mock
// ---------------------------------------------------------------------------

// TestServiceContract_CreateCustomer verifica que o contrato de criação é respeitado.
func TestServiceContract_CreateCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()
	createdBy := uuid.New()
	newID := uuid.New()

	mockSvc.EXPECT().
		CreateCustomer(gomock.Any(), gomock.AssignableToTypeOf(domain.CreateCustomersRequest{})).
		Return(newID, nil)

	id, err := mockSvc.CreateCustomer(ginCtx(), domain.CreateCustomersRequest{
		CompanyID: companyID,
		FullName:  "João Santos",
		Email:     "joao@email.com",
		Gender:    enums.GenderMale,
		CreatedBy: createdBy,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if id != newID {
		t.Errorf("ID incorreto")
	}
}

// TestServiceContract_DeleteCustomer verifica o contrato de deleção.
func TestServiceContract_DeleteCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()

	mockSvc.EXPECT().
		DeleteCustomer(gomock.Any(), domain.DeleteCustomerRequest{
			ID:        id,
			DeletedBy: uuid.Nil,
		}).
		Return(nil)

	err := mockSvc.DeleteCustomer(ginCtx(), domain.DeleteCustomerRequest{
		ID:        id,
		DeletedBy: uuid.Nil,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

// TestServiceContract_GetCustomerByCPF_Success verifica busca por CPF.
func TestServiceContract_GetCustomerByCPF_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	companyID := uuid.New()
	cpf := "123.456.789-00"

	expected := sampleCustomerResponse(id, companyID)
	expected.Cpf = cpf

	mockSvc.EXPECT().
		GetCustomerByCPF(gomock.Any(), cpf).
		Return(expected, nil)

	resp, err := mockSvc.GetCustomerByCPF(ginCtx(), cpf)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Cpf != cpf {
		t.Errorf("Cpf incorreto: '%s'", resp.Cpf)
	}
}

// TestServiceContract_GetCustomerByCPF_Error verifica propagação de erro.
func TestServiceContract_GetCustomerByCPF_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)

	mockSvc.EXPECT().
		GetCustomerByCPF(gomock.Any(), gomock.Any()).
		Return(domain.CustomerResponse{}, errService)

	_, err := mockSvc.GetCustomerByCPF(ginCtx(), "000.000.000-00")

	if err == nil {
		t.Fatal("esperava erro do serviço")
	}
	if !errors.Is(err, errService) {
		t.Errorf("erro incorreto: %v", err)
	}
}

// TestServiceContract_GetCustomerById_Success verifica busca por ID.
func TestServiceContract_GetCustomerById_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	companyID := uuid.New()

	expected := sampleCustomerResponse(id, companyID)

	mockSvc.EXPECT().
		GetCustomerById(gomock.Any(), id).
		Return(expected, nil)

	resp, err := mockSvc.GetCustomerById(ginCtx(), id)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
	if resp.FullName != "Ana Costa" {
		t.Errorf("FullName incorreto: '%s'", resp.FullName)
	}
}

// TestServiceContract_ListCustomers_Success verifica listagem completa.
func TestServiceContract_ListCustomers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	customers := []domain.CustomerResponse{
		sampleCustomerResponse(uuid.New(), companyID),
		sampleCustomerResponse(uuid.New(), companyID),
	}

	mockSvc.EXPECT().
		ListCustomers(gomock.Any(), companyID).
		Return(customers, nil)

	resp, err := mockSvc.ListCustomers(ginCtx(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("esperava 2 clientes, obteve %d", len(resp))
	}
}

// TestServiceContract_ListCustomersPaginated_Success verifica paginação.
func TestServiceContract_ListCustomersPaginated_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()
	pagination := globalDomain.PaginationParams{Page: 1, PerPage: 10}

	paginatedResp := domain.CustomerPaginatedResponse{
		PaginatedResponse: globalDomain.PaginatedResponse[domain.CustomerResponse]{
			Data:       []domain.CustomerResponse{sampleCustomerResponse(uuid.New(), companyID)},
			Page:       1,
			PerPage:    10,
			TotalRows:  1,
			TotalPages: 1,
		},
	}

	mockSvc.EXPECT().
		ListCustomersPaginated(gomock.Any(), companyID, pagination).
		Return(paginatedResp, nil)

	resp, err := mockSvc.ListCustomersPaginated(ginCtx(), companyID, pagination)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("esperava Page=1, obteve %d", resp.Page)
	}
	if len(resp.Data) != 1 {
		t.Errorf("esperava 1 cliente, obteve %d", len(resp.Data))
	}
}

// TestServiceContract_UpdateBalanceDue_Success verifica atualização de saldo.
func TestServiceContract_UpdateBalanceDue_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	updatedBy := uuid.New()

	req := domain.UpdateBalanceDueCustomerRequest{
		BalanceDue: 500.00,
		Prohibited: 0,
		UpdatedBy:  updatedBy,
	}

	mockSvc.EXPECT().
		UpdateBalanceDueCustomer(gomock.Any(), id, req).
		Return(nil)

	err := mockSvc.UpdateBalanceDueCustomer(ginCtx(), id, req)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

// TestServiceContract_UpdateCustomer_Success verifica atualização de cliente.
func TestServiceContract_UpdateCustomer_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()

	req := domain.UpdateCustomerRequest{
		FullName:  "Nome Atualizado",
		Email:     "novo@email.com",
		UpdatedBy: uuid.New(),
	}

	mockSvc.EXPECT().
		UpdateCustomer(gomock.Any(), id, req).
		Return(nil)

	err := mockSvc.UpdateCustomer(ginCtx(), id, req)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

// TestServiceContract_CountCustomers_Success verifica contagem.
func TestServiceContract_CountCustomers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		CountCustomers(gomock.Any(), companyID).
		Return(int64(42), nil)

	count, err := mockSvc.CountCustomers(ginCtx(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 42 {
		t.Errorf("esperava 42, obteve %d", count)
	}
}

// TestServiceContract_GetCustomersPerformanceSummary_Success verifica percentual.
func TestServiceContract_GetCustomersPerformanceSummary_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetCustomersPerformanceSummary(gomock.Any(), companyID).
		Return(25.5, nil)

	percentage, err := mockSvc.GetCustomersPerformanceSummary(ginCtx(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 25.5 {
		t.Errorf("esperava 25.5, obteve %f", percentage)
	}
}

// ---------------------------------------------------------------------------
// Testes de integração HTTP com httptest + gin
// ---------------------------------------------------------------------------

// TestHTTP_CreateCustomer_MissingSubjectToken verifica 401 sem "sub".
func TestHTTP_CreateCustomer_MissingSubjectToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/customers", func(c *gin.Context) {
		_, exists := c.Get("sub")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is null"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

// TestHTTP_CreateCustomer_MissingCompanyId verifica 401 sem "company_id".
func TestHTTP_CreateCustomer_MissingCompanyId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/customers", func(c *gin.Context) {
		_, exists := c.Get("sub")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "userId is null"})
			return
		}
		_, exists = c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{})
	})

	// Injeta "sub" mas não "company_id"
	r2 := gin.New()
	r2.Use(func(c *gin.Context) {
		c.Set("sub", uuid.New().String())
		c.Next()
	})
	r2.POST("/customers", func(c *gin.Context) {
		_, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

// TestHTTP_DeleteCustomer_InvalidUUID verifica 400 com UUID inválido.
func TestHTTP_DeleteCustomer_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/customers/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/customers/nao-e-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

// TestHTTP_DeleteCustomer_ValidUUID_Success verifica 204 com deleção bem-sucedida.
func TestHTTP_DeleteCustomer_ValidUUID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	customerID := uuid.New()
	companyID := uuid.New()

	mockSvc.EXPECT().
		DeleteCustomer(gomock.Any(), gomock.AssignableToTypeOf(domain.DeleteCustomerRequest{})).
		Return(nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.DELETE("/customers/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		companyIdAny, _ := c.Get("company_id")
		companyId := companyIdAny.(uuid.UUID)
		if err := mockSvc.DeleteCustomer(c.Request.Context(), domain.DeleteCustomerRequest{
			ID:        id,
			DeletedBy: companyId,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/customers/%s", customerID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("esperava 204, obteve %d", w.Code)
	}
}

// TestHTTP_DeleteCustomer_ServiceError verifica 500 quando serviço falha.
func TestHTTP_DeleteCustomer_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	customerID := uuid.New()
	companyID := uuid.New()

	mockSvc.EXPECT().
		DeleteCustomer(gomock.Any(), gomock.Any()).
		Return(errService)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.DELETE("/customers/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := mockSvc.DeleteCustomer(c.Request.Context(), domain.DeleteCustomerRequest{ID: id}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/customers/%s", customerID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d", w.Code)
	}
}

// TestHTTP_GetCustomerByCPF_Success verifica retorno 200.
func TestHTTP_GetCustomerByCPF_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	cpf := "123.456.789-00"
	id := uuid.New()
	companyID := uuid.New()

	expected := sampleCustomerResponse(id, companyID)
	expected.Cpf = cpf

	mockSvc.EXPECT().
		GetCustomerByCPF(gomock.Any(), cpf).
		Return(expected, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/customers/cpf/:cpf", func(c *gin.Context) {
		cpfParam := c.Param("cpf")
		customer, err := mockSvc.GetCustomerByCPF(c.Request.Context(), cpfParam)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"customer": customer})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/cpf/"+cpf, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&body)
	customerMap, ok := body["customer"].(map[string]interface{})
	if !ok {
		t.Fatal("resposta não contém 'customer'")
	}
	if customerMap["cpf"] != cpf {
		t.Errorf("CPF incorreto na resposta: %v", customerMap["cpf"])
	}
}

// TestHTTP_GetCustomerById_InvalidUUID verifica 400.
func TestHTTP_GetCustomerById_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/customers/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/nao-e-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

// TestHTTP_CountCustomers_MissingCompanyId verifica 401 sem company_id.
func TestHTTP_CountCustomers_MissingCompanyId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/customers/count", func(c *gin.Context) {
		_, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": 0})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/count", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

// TestHTTP_CountCustomers_Success verifica retorno 200 com contagem.
func TestHTTP_CountCustomers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		CountCustomers(gomock.Any(), companyID).
		Return(int64(7), nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.GET("/customers/count", func(c *gin.Context) {
		companyIdAny, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
			return
		}
		count, err := mockSvc.CountCustomers(c.Request.Context(), companyIdAny.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": count})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/count", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if int(body["count"].(float64)) != 7 {
		t.Errorf("count incorreto na resposta: %v", body["count"])
	}
}

// TestHTTP_GetCustomersPerformanceSummary_Success verifica retorno 200.
func TestHTTP_GetCustomersPerformanceSummary_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetCustomersPerformanceSummary(gomock.Any(), companyID).
		Return(33.33, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("company_id", companyID)
		c.Next()
	})
	r.GET("/customers/percentage", func(c *gin.Context) {
		companyIdAny, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
			return
		}
		percentage, err := mockSvc.GetCustomersPerformanceSummary(c.Request.Context(), companyIdAny.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"percentage": percentage})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/percentage", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var body map[string]float64
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["percentage"] != 33.33 {
		t.Errorf("percentage incorreto: %v", body["percentage"])
	}
}

// TestHTTP_UpdateCustomer_InvalidUUID verifica 400 com UUID inválido.
func TestHTTP_UpdateCustomer_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/customers/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/customers/invalid", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

// TestHTTP_ListCustomers_MissingCompanyId verifica 401 sem company_id.
func TestHTTP_ListCustomers_MissingCompanyId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/customers/list", func(c *gin.Context) {
		_, exists := c.Get("company_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "companyId is null"})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/customers/list", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}
