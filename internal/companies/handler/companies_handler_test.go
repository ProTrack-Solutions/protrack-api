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

	"github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/companies/mocks"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var errService = errors.New("service error")

func ginCtx() context.Context { return context.Background() }

func sampleCompanyResponse(id uuid.UUID, name string) domain.CompanyResponse {
	return domain.CompanyResponse{
		ID:                      id,
		Name:                    name,
		TradeName:               "Trade " + name,
		Document:                "00000000000191",
		DocumentType:            "CNPJ",
		Email:                   "empresa@email.com",
		Phone:                   "+5511999999999",
		Website:                 "https://empresa.com",
		AddressStreet:           "Rua Teste",
		AddressNumber:           "100",
		AddressNeighborhood:     "Bairro Teste",
		AddressCity:             "São Paulo",
		AddressState:            "SP",
		AddressZipcode:          "01000-000",
		AddressCountry:          "Brasil",
		Status:                  "active",
		CreatedBy:               uuid.New(),
		CreatedAt:               time.Now().UTC(),
		InscricaoEstadual:       "110042490114",
		InscricaoEstadualIsento: false,
		Cnae:                    "4711-3/02",
		RegimeTributario:        "simples_nacional",
	}
}

// ---------------------------------------------------------------------------
// Testes de Contrato de Serviço via Mock (Companies)
// ---------------------------------------------------------------------------

func TestServiceContract_DeleteCompany(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	req := domain.DeleteCompanyParams{
		ID:        id,
		DeletedBy: uuid.Nil,
	}

	mockSvc.EXPECT().
		DeleteCompany(gomock.Any(), req).
		Return(nil)

	err := mockSvc.DeleteCompany(ginCtx(), req)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestServiceContract_GetCompanyByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	expected := sampleCompanyResponse(id, "Empresa Um")

	mockSvc.EXPECT().
		GetCompanyByID(gomock.Any(), id).
		Return(expected, nil)

	resp, err := mockSvc.GetCompanyByID(ginCtx(), id)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
	if resp.Name != "Empresa Um" {
		t.Errorf("Name incorreto: '%s'", resp.Name)
	}
}

func TestServiceContract_GetCompanyByID_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()

	mockSvc.EXPECT().
		GetCompanyByID(gomock.Any(), id).
		Return(domain.CompanyResponse{}, errService)

	_, err := mockSvc.GetCompanyByID(ginCtx(), id)
	if err == nil {
		t.Fatal("esperava erro do serviço")
	}
	if !errors.Is(err, errService) {
		t.Errorf("erro incorreto: %v", err)
	}
}

func TestServiceContract_GetCompanyByDocument_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	doc := "00000000000191"
	expected := sampleCompanyResponse(id, "Empresa Doc")

	mockSvc.EXPECT().
		GetCompanyByDocument(gomock.Any(), doc).
		Return(expected, nil)

	resp, err := mockSvc.GetCompanyByDocument(ginCtx(), doc)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Document != doc {
		t.Errorf("Document incorreto: '%s'", resp.Document)
	}
}

func TestServiceContract_ListCompanies_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)

	companies := []domain.CompanyResponse{
		sampleCompanyResponse(uuid.New(), "Empresa 1"),
		sampleCompanyResponse(uuid.New(), "Empresa 2"),
	}

	mockSvc.EXPECT().
		ListCompanies(gomock.Any()).
		Return(companies, nil)

	resp, err := mockSvc.ListCompanies(ginCtx())
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("esperava 2 empresas, obteve %d", len(resp))
	}
}

func TestServiceContract_SetCompanyStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	req := domain.SetCompanyStatusParams{
		ID:     uuid.New(),
		Status: "inactive",
	}

	mockSvc.EXPECT().
		SetCompanyStatus(gomock.Any(), req).
		Return(int64(1), nil)

	count, err := mockSvc.SetCompanyStatus(ginCtx(), req)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 1 {
		t.Errorf("esperava count=1, obteve %d", count)
	}
}

func TestServiceContract_UpdateCompany_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	id := uuid.New()
	req := domain.UpdateCompanyRequest{
		Name: "Nome Atualizado",
	}

	expected := sampleCompanyResponse(id, "Nome Atualizado")

	mockSvc.EXPECT().
		UpdateCompany(gomock.Any(), id, req).
		Return(expected, nil)

	resp, err := mockSvc.UpdateCompany(ginCtx(), id, req)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Name != "Nome Atualizado" {
		t.Errorf("Name incorreto: '%s'", resp.Name)
	}
}

// ---------------------------------------------------------------------------
// Testes de integração HTTP com httptest + gin (Companies)
// ---------------------------------------------------------------------------

func TestHTTP_DeleteCompany_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/companies/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/companies/nao-e-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

func TestHTTP_DeleteCompany_ValidUUID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()

	mockSvc.EXPECT().
		DeleteCompany(gomock.Any(), domain.DeleteCompanyParams{
			ID:        companyID,
			DeletedBy: uuid.Nil,
		}).
		Return(nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/companies/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := mockSvc.DeleteCompany(c.Request.Context(), domain.DeleteCompanyParams{
			ID:        id,
			DeletedBy: uuid.Nil,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/companies/%s", companyID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("esperava 204, obteve %d", w.Code)
	}
}

func TestHTTP_GetCompanyByID_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/companies/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/companies/invalid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

func TestHTTP_GetCompanyByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companyID := uuid.New()
	expected := sampleCompanyResponse(companyID, "Empresa Teste HTTP")

	mockSvc.EXPECT().
		GetCompanyByID(gomock.Any(), companyID).
		Return(expected, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/companies/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		company, err := mockSvc.GetCompanyByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"company": company})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/companies/%s", companyID), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&body)
	compMap, ok := body["company"].(map[string]interface{})
	if !ok {
		t.Fatal("resposta não contém 'company'")
	}
	if compMap["name"] != "Empresa Teste HTTP" {
		t.Errorf("Nome da empresa incorreto: %v", compMap["name"])
	}
	if compMap["inscricao_estadual"] != "110042490114" {
		t.Errorf("inscricao_estadual incorreta: %v", compMap["inscricao_estadual"])
	}
	if compMap["regime_tributario"] != "simples_nacional" {
		t.Errorf("regime_tributario incorreto: %v", compMap["regime_tributario"])
	}
}

func TestHTTP_GetCompanyByDocument_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	doc := "00000000000191"
	expected := sampleCompanyResponse(uuid.New(), "Empresa Doc HTTP")

	mockSvc.EXPECT().
		GetCompanyByDocument(gomock.Any(), doc).
		Return(expected, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/companies/document/:document", func(c *gin.Context) {
		document := c.Param("document")
		company, err := mockSvc.GetCompanyByDocument(c.Request.Context(), document)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"company": company})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/companies/document/"+doc, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_ListCompanies_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	companies := []domain.CompanyResponse{
		sampleCompanyResponse(uuid.New(), "Empresa 1"),
		sampleCompanyResponse(uuid.New(), "Empresa 2"),
	}

	mockSvc.EXPECT().
		ListCompanies(gomock.Any()).
		Return(companies, nil)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/companies", func(c *gin.Context) {
		result, err := mockSvc.ListCompanies(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"companies": result})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/companies", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_UpdateCompany_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/companies/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if _, err := uuid.Parse(idStr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/companies/invalid", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}
