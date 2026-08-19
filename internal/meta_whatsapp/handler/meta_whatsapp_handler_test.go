package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/handler"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/mocks"
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

// newTestHandler monta um handler.Handler real com um ServiceInterface mockado
// e um DiscordLogger "mudo" (webhook vazio faz o Send() retornar sem I/O de rede).
func newTestHandler(mockSvc *mocks.MockServiceInterface) *handler.Handler {
	cfg := &config.Config{
		MetaWebhookVerifyToken: "verify-token-123",
		MetaAppSecret:          "app-secret",
	}
	discordLogger := discord.NewDiscordLogger(&config.Config{})
	return handler.NewHandler(mockSvc, cfg, discordLogger, nil, nil)
}

// withCompanyID injeta "company_id" no contexto antes do handler real rodar,
// simulando o que o AuthMiddleware faria em produção.
func withCompanyID(companyID uuid.UUID, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("company_id", companyID)
		next(c)
	}
}

func sampleMessage(companyID uuid.UUID) *domain.Message {
	return &domain.Message{
		ID:             uuid.New(),
		CompanyID:      companyID,
		Category:       domain.CategoryUtility,
		RecipientPhone: "+5511988887777",
		Status:         domain.StatusSent,
	}
}

// ---------------------------------------------------------------------------
// SendMessage
// ---------------------------------------------------------------------------

func TestHTTP_SendMessage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	req := domain.SendMessageRequest{
		TemplateName:   "order_confirmation",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
	}
	expected := sampleMessage(companyID)

	mockSvc.EXPECT().
		SendMessage(gomock.Any(), companyID, req).
		Return(expected, nil)

	r := gin.New()
	r.POST("/send-message", withCompanyID(companyID, h.SendMessage))

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/send-message", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}

	var resp domain.Message
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("falha ao decodificar resposta: %v", err)
	}
	if resp.ID != expected.ID {
		t.Errorf("ID incorreto na resposta")
	}
}

func TestHTTP_SendMessage_MissingCompanyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	r := gin.New()
	r.POST("/send-message", h.SendMessage)

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/send-message", bytes.NewReader([]byte(`{}`)))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

func TestHTTP_SendMessage_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	r := gin.New()
	r.POST("/send-message", withCompanyID(companyID, h.SendMessage))

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/send-message", bytes.NewReader([]byte(`{invalido`)))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

func TestHTTP_SendMessage_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	mockSvc.EXPECT().
		SendMessage(gomock.Any(), companyID, gomock.Any()).
		Return(&domain.Message{}, errService)

	r := gin.New()
	r.POST("/send-message", withCompanyID(companyID, h.SendMessage))

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/send-message", bytes.NewReader([]byte(`{"template_name":"x","language_code":"pt_BR","recipient_phone":"+5511988887777"}`)))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetCompanyConfig
// ---------------------------------------------------------------------------

func TestHTTP_GetCompanyConfig_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	expected := &domain.CompanyWhatsAppConfig{
		ID:        uuid.New(),
		CompanyID: companyID,
		Mode:      domain.WhatsAppModePlatformShared,
		IsActive:  true,
	}

	mockSvc.EXPECT().
		GetCompanyConfig(gomock.Any(), companyID).
		Return(expected, nil)

	r := gin.New()
	r.GET("/config", withCompanyID(companyID, h.GetCompanyConfig))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/config", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_GetCompanyConfig_MissingCompanyID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	r := gin.New()
	r.GET("/config", h.GetCompanyConfig)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/config", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperava 401, obteve %d", w.Code)
	}
}

func TestHTTP_GetCompanyConfig_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	mockSvc.EXPECT().
		GetCompanyConfig(gomock.Any(), companyID).
		Return(&domain.CompanyWhatsAppConfig{}, errService)

	r := gin.New()
	r.GET("/config", withCompanyID(companyID, h.GetCompanyConfig))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/config", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// UpsertCompanyConfig
// ---------------------------------------------------------------------------

func TestHTTP_UpsertCompanyConfig_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	reqBody := domain.UpsertCompanyConfigRequest{
		Mode:                    domain.WhatsAppModePlatformShared,
		MonthlyMessageAllowance: 500,
		IsActive:                true,
	}
	expected := &domain.CompanyWhatsAppConfig{ID: uuid.New(), CompanyID: companyID}

	mockSvc.EXPECT().
		UpsertCompanyConfig(gomock.Any(), companyID, reqBody).
		Return(expected, nil)

	r := gin.New()
	r.POST("/upsert-config", withCompanyID(companyID, h.UpsertCompanyConfig))

	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/upsert-config", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_UpsertCompanyConfig_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	r := gin.New()
	r.POST("/upsert-config", withCompanyID(companyID, h.UpsertCompanyConfig))

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/upsert-config", bytes.NewReader([]byte(`{invalido`)))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperava 400, obteve %d", w.Code)
	}
}

func TestHTTP_UpsertCompanyConfig_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)
	companyID := uuid.New()

	mockSvc.EXPECT().
		UpsertCompanyConfig(gomock.Any(), companyID, gomock.Any()).
		Return(&domain.CompanyWhatsAppConfig{}, errService)

	r := gin.New()
	r.POST("/upsert-config", withCompanyID(companyID, h.UpsertCompanyConfig))

	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/upsert-config", bytes.NewReader([]byte(`{"mode":"own_waba"}`)))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// ListApprovedTemplates
// ---------------------------------------------------------------------------

func TestHTTP_ListApprovedTemplates_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	mockSvc.EXPECT().
		ListApprovedTemplates(gomock.Any()).
		Return([]domain.Template{{MetaTemplateName: "template_1"}}, nil)

	r := gin.New()
	r.GET("/list-templates", h.ListApprovedTemplates)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/list-templates", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_ListApprovedTemplates_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	mockSvc.EXPECT().
		ListApprovedTemplates(gomock.Any()).
		Return(nil, errService)

	r := gin.New()
	r.GET("/list-templates", h.ListApprovedTemplates)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/list-templates", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperava 500, obteve %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// VerifyWebhook
// ---------------------------------------------------------------------------

func TestHTTP_VerifyWebhook_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	r := gin.New()
	r.GET("/webhook", h.VerifyWebhook)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=verify-token-123&hub.challenge=desafio-123", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", w.Code)
	}
	if w.Body.String() != "desafio-123" {
		t.Errorf("esperava o challenge no corpo, obteve: %s", w.Body.String())
	}
}

func TestHTTP_VerifyWebhook_WrongToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	r := gin.New()
	r.GET("/webhook", h.VerifyWebhook)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=subscribe&hub.verify_token=token-errado&hub.challenge=x", nil))

	if w.Code != http.StatusForbidden {
		t.Errorf("esperava 403, obteve %d", w.Code)
	}
}

func TestHTTP_VerifyWebhook_WrongMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	r := gin.New()
	r.GET("/webhook", h.VerifyWebhook)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhook?hub.mode=unsubscribe&hub.verify_token=verify-token-123&hub.challenge=x", nil))

	if w.Code != http.StatusForbidden {
		t.Errorf("esperava 403, obteve %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// HandleWebhookEvent
// ---------------------------------------------------------------------------

func TestHTTP_HandleWebhookEvent_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	payload := []byte(`{"object":"whatsapp_business_account","entry":[]}`)

	mockSvc.EXPECT().
		HandleWebhookEvent(gomock.Any(), payload).
		Return(nil)

	r := gin.New()
	r.POST("/webhook", h.HandleWebhookEvent)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload)))

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200, obteve %d", w.Code)
	}
}

// A Meta exige 200 mesmo quando o processamento falha internamente, senão
// ela reenvia o mesmo evento repetidamente — por isso o handler sempre
// responde 200 e só loga o erro via Discord.
func TestHTTP_HandleWebhookEvent_ServiceError_StillReturns200(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockServiceInterface(ctrl)
	h := newTestHandler(mockSvc)

	payload := []byte(`{"object":"whatsapp_business_account","entry":[]}`)

	mockSvc.EXPECT().
		HandleWebhookEvent(gomock.Any(), payload).
		Return(errService)

	r := gin.New()
	r.POST("/webhook", h.HandleWebhookEvent)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload)))

	if w.Code != http.StatusOK {
		t.Errorf("esperava 200 mesmo com erro do service, obteve %d", w.Code)
	}
}
