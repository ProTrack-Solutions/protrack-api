package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/handler"
	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var errService = errors.New("service error")

func init() {
	gin.SetMode(gin.TestMode)
}

// newTestHandler monta um handler.Handler real com um ServiceInterface mockado.
// jwt/blacklist não são usados pelos métodos do handler diretamente (apenas
// pelo middleware de autenticação, montado fora dele), então podem ser nil aqui.
func newTestHandler(mockSvc *mocks.MockServiceInterface) *handler.Handler {
	return handler.NewHandler(mockSvc, nil, nil)
}

// withCompanyID injeta "company_id" no contexto antes do handler real rodar,
// simulando o que o AuthMiddleware faria em produção.
func withCompanyID(companyID uuid.UUID, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("company_id", companyID)
		next(c)
	}
}

func sampleSettingResponse(companyID uuid.UUID, key enums.CompanySettingsKey, value any) domain.CompanySettingResponse {
	now := time.Now().UTC()
	return domain.CompanySettingResponse{
		ID:        uuid.New(),
		CompanyID: companyID,
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ---------------------------------------------------------------------------
// ListCompanySettings
// ---------------------------------------------------------------------------

func TestHTTP_ListCompanySettings_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	expected := []domain.CompanySettingResponse{
		sampleSettingResponse(companyID, enums.IsWhatsappActive, true),
		sampleSettingResponse(companyID, enums.LowStock, float64(2)),
	}

	mockSvc.EXPECT().
		ListCompanySettings(gomock.Any(), companyID).
		Return(expected, nil)

	r := gin.New()
	r.GET("/company-settings", withCompanyID(companyID, h.ListCompanySettings))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/company-settings", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var resp []domain.CompanySettingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("falha ao decodificar resposta: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("esperava 2 configs, obteve %d", len(resp))
	}
}

func TestHTTP_ListCompanySettings_MissingCompanyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	r := gin.New()
	r.GET("/company-settings", h.ListCompanySettings)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/company-settings", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

func TestHTTP_ListCompanySettings_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	mockSvc.EXPECT().
		ListCompanySettings(gomock.Any(), companyID).
		Return(nil, errService)

	r := gin.New()
	r.GET("/company-settings", withCompanyID(companyID, h.ListCompanySettings))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/company-settings", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// UpsertCompanySetting
// ---------------------------------------------------------------------------

func TestHTTP_UpsertCompanySetting_Created(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	req := domain.UpsertCompanySettingRequest{
		Key:   enums.LowStock,
		Value: float64(3),
	}

	now := time.Now().UTC()

	mockSvc.EXPECT().
		UpsertCompanySetting(gomock.Any(), companyID, req).
		Return(domain.CompanySettingResponse{
			ID:        uuid.New(),
			CompanyID: companyID,
			Key:       enums.LowStock,
			Value:     float64(3),
			CreatedAt: now,
			UpdatedAt: now, // igual a CreatedAt -> handler deve responder 201
		}, nil)

	r := gin.New()
	r.POST("/company-settings", withCompanyID(companyID, h.UpsertCompanySetting))

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/company-settings", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusCreated {
		t.Fatalf("esperava 201, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_UpsertCompanySetting_UpdatedOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	req := domain.UpsertCompanySettingRequest{
		Key:   enums.LowStock,
		Value: float64(3),
	}

	created := time.Now().Add(-time.Hour).UTC()
	updated := time.Now().UTC()

	mockSvc.EXPECT().
		UpsertCompanySetting(gomock.Any(), companyID, req).
		Return(domain.CompanySettingResponse{
			ID:        uuid.New(),
			CompanyID: companyID,
			Key:       enums.LowStock,
			Value:     float64(3),
			CreatedAt: created,
			UpdatedAt: updated, // diferente de CreatedAt -> handler deve responder 200
		}, nil)

	r := gin.New()
	r.POST("/company-settings", withCompanyID(companyID, h.UpsertCompanySetting))

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/company-settings", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_UpsertCompanySetting_MissingCompanyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	r := gin.New()
	r.POST("/company-settings", h.UpsertCompanySetting)

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/company-settings", bytes.NewReader([]byte(`{}`)))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

func TestHTTP_UpsertCompanySetting_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	r := gin.New()
	r.POST("/company-settings", withCompanyID(companyID, h.UpsertCompanySetting))

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/company-settings", bytes.NewReader([]byte(`{invalido`)))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

func TestHTTP_UpsertCompanySetting_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	mockSvc.EXPECT().
		UpsertCompanySetting(gomock.Any(), companyID, gomock.Any()).
		Return(domain.CompanySettingResponse{}, errService)

	r := gin.New()
	r.POST("/company-settings", withCompanyID(companyID, h.UpsertCompanySetting))

	body, _ := json.Marshal(domain.UpsertCompanySettingRequest{Key: enums.LowStock, Value: float64(3)})
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/company-settings", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// RestorDefaultSettings
// ---------------------------------------------------------------------------

func TestHTTP_RestorDefaultSettings_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	mockSvc.EXPECT().
		RestorDefaultSettings(gomock.Any(), companyID).
		Return(nil)

	r := gin.New()
	r.PUT("/company-settings", withCompanyID(companyID, h.RestorDefaultSettings))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/company-settings", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_RestorDefaultSettings_MissingCompanyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	r := gin.New()
	r.PUT("/company-settings", h.RestorDefaultSettings)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/company-settings", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

func TestHTTP_RestorDefaultSettings_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	mockSvc.EXPECT().
		RestorDefaultSettings(gomock.Any(), companyID).
		Return(errService)

	r := gin.New()
	r.PUT("/company-settings", withCompanyID(companyID, h.RestorDefaultSettings))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/company-settings", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d — body: %s", w.Code, w.Body.String())
	}
}
