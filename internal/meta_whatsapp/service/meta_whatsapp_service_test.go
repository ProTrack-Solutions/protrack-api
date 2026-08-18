package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/crypto"
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	companiesDomain "github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/service"
	plansFeaturesDomain "github.com/ProTrack-Solutions/protrack-api/internal/plan_features/domain"
	plansDomain "github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	subscriptionsDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers e Fixtures
// ---------------------------------------------------------------------------

const testSecretKey = "test-secret-key"

var errDatabase = errors.New("database error")

// deps agrupa todos os mocks de dependência do service, pra reduzir boilerplate
// na assinatura dos testes.
type deps struct {
	repo            *mocks.MockRepositoryInterface
	metaClient      *mocks.MockMetaClientInterface
	subscriptionSvc *mocks.MockSubscriptionsServiceInterface
	plansSvc        *mocks.MockPlansServiceInterface
	billingClient   *mocks.MockBillingClientInterface
	companiesSvc    *mocks.MockCompaniesServiceInterface
}

func newDeps(ctrl *gomock.Controller) *deps {
	return &deps{
		repo:            mocks.NewMockRepositoryInterface(ctrl),
		metaClient:      mocks.NewMockMetaClientInterface(ctrl),
		subscriptionSvc: mocks.NewMockSubscriptionsServiceInterface(ctrl),
		plansSvc:        mocks.NewMockPlansServiceInterface(ctrl),
		billingClient:   mocks.NewMockBillingClientInterface(ctrl),
		companiesSvc:    mocks.NewMockCompaniesServiceInterface(ctrl),
	}
}

// newSvc cria um Service com todos os mocks injetados e as credenciais Meta
// usadas nos testes (phoneNumberID/accessToken do modo platform_shared).
func newSvc(d *deps) *service.Service {
	return service.NewServiceWithDeps(
		d.repo, d.metaClient, d.subscriptionSvc, d.plansSvc, d.billingClient, d.companiesSvc,
		"platform-phone-id", "platform-access-token", testSecretKey, "price_whatsapp_overage",
	)
}

func buildCompanyConfig(id, companyID uuid.UUID, mode db.WhatsappMode) db.CompanyWhatsappConfig {
	return db.CompanyWhatsappConfig{
		ID:                      pgconv.ParseUUIDToPgType(id),
		CompanyID:               pgconv.ParseUUIDToPgType(companyID),
		Mode:                    mode,
		WabaID:                  pgtype.Text{String: "waba-123", Valid: true},
		PhoneNumberID:           pgtype.Text{String: "own-phone-id", Valid: true},
		DisplayPhoneNumber:      pgtype.Text{String: "+5511999999999", Valid: true},
		AccessTokenEncrypted:    pgtype.Text{Valid: false},
		MonthlyMessageAllowance: 1000,
		IsActive:                true,
		CreatedAt:               pgtype.Timestamptz{Time: time.Now(), Valid: true},
		UpdatedAt:               pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

// ---------------------------------------------------------------------------
// SendMessage
// ---------------------------------------------------------------------------

func TestSendMessage_PlatformShared_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	templateID := uuid.New()
	messageID := uuid.New()

	companyConfig := buildCompanyConfig(uuid.New(), companyID, db.WhatsappModePlatformShared)

	req := domain.SendMessageRequest{
		TemplateName:   "order_confirmation",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
	}

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), pgconv.OptionalUUIDToPgType(companyID)).
		Return(companyConfig, nil)

	d.repo.EXPECT().
		GetEligibleTemplateByName(gomock.Any(), db.GetEligibleTemplateByNameParams{
			MetaTemplateName: req.TemplateName,
			LanguageCode:     req.LanguageCode,
		}).
		Return(db.WhatsappTemplate{
			ID:               pgconv.ParseUUIDToPgType(templateID),
			MetaTemplateName: req.TemplateName,
			Category:         db.WhatsappMessageCategoryUtility,
		}, nil)

	d.metaClient.EXPECT().
		SendTemplateMessage(gomock.Any(), "platform-phone-id", "platform-access-token", req).
		Return("wamid.123", nil)

	d.repo.EXPECT().
		CreateWhatsAppMessage(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
			if arg.Category != db.WhatsappMessageCategoryUtility {
				t.Errorf("categoria incorreta: %v", arg.Category)
			}
			if pgconv.PgUUIDToUUID(arg.TemplateID) != templateID {
				t.Errorf("templateID incorreto")
			}
			if arg.MetaMessageID.String != "wamid.123" {
				t.Errorf("metaMessageID incorreto: %s", arg.MetaMessageID.String)
			}
			return db.WhatsappMessage{
				ID:                 pgconv.ParseUUIDToPgType(messageID),
				CompanyID:          pgconv.ParseUUIDToPgType(companyID),
				TemplateID:         arg.TemplateID,
				Category:           arg.Category,
				RecipientPhone:     arg.RecipientPhone,
				MetaMessageID:      arg.MetaMessageID,
				Status:             arg.Status,
				EstimatedCostCents: arg.EstimatedCostCents,
				CreatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil
		})

	msg, err := svc.SendMessage(context.Background(), companyID, req)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if msg.ID != messageID {
		t.Errorf("ID incorreto")
	}
	if msg.Status != domain.StatusSent {
		t.Errorf("status incorreto: %s", msg.Status)
	}
	if msg.Category != domain.CategoryUtility {
		t.Errorf("categoria incorreta: %s", msg.Category)
	}
}

func TestSendMessage_GetCompanyConfigError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(db.CompanyWhatsappConfig{}, errDatabase)

	_, err := svc.SendMessage(context.Background(), uuid.New(), domain.SendMessageRequest{})

	if err == nil {
		t.Fatal("esperava erro ao buscar config da empresa")
	}
}

func TestSendMessage_PlatformShared_TemplateNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	companyConfig := buildCompanyConfig(uuid.New(), companyID, db.WhatsappModePlatformShared)

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(companyConfig, nil)

	d.repo.EXPECT().
		GetEligibleTemplateByName(gomock.Any(), gomock.Any()).
		Return(db.WhatsappTemplate{}, errors.New("not found"))

	_, err := svc.SendMessage(context.Background(), companyID, domain.SendMessageRequest{
		TemplateName: "inexistente",
		LanguageCode: "pt_BR",
	})

	if err == nil {
		t.Fatal("esperava erro ao buscar template elegível")
	}
}

func TestSendMessage_PlatformShared_TemplateNameMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	companyConfig := buildCompanyConfig(uuid.New(), companyID, db.WhatsappModePlatformShared)

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(companyConfig, nil)

	// Repositório devolve um template com nome diferente do solicitado.
	d.repo.EXPECT().
		GetEligibleTemplateByName(gomock.Any(), gomock.Any()).
		Return(db.WhatsappTemplate{MetaTemplateName: "outro_template"}, nil)

	_, err := svc.SendMessage(context.Background(), companyID, domain.SendMessageRequest{
		TemplateName: "template_solicitado",
		LanguageCode: "pt_BR",
	})

	if err == nil {
		t.Fatal("esperava erro de template não registrado")
	}
}

func TestSendMessage_OwnWABA_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	messageID := uuid.New()

	plaintextToken := "own-waba-access-token"
	encryptedToken, err := crypto.Encrypt(testSecretKey, plaintextToken)
	if err != nil {
		t.Fatalf("falha ao preparar fixture: %v", err)
	}

	companyConfig := buildCompanyConfig(uuid.New(), companyID, db.WhatsappModeOwnWaba)
	companyConfig.AccessTokenEncrypted = pgtype.Text{String: encryptedToken, Valid: true}

	req := domain.SendMessageRequest{
		TemplateName:   "custom_template",
		LanguageCode:   "pt_BR",
		RecipientPhone: "+5511988887777",
		Category:       domain.CategoryMarketing,
	}

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(companyConfig, nil)

	d.metaClient.EXPECT().
		SendTemplateMessage(gomock.Any(), "own-phone-id", plaintextToken, req).
		Return("wamid.own", nil)

	d.repo.EXPECT().
		CreateWhatsAppMessage(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
			if arg.Category != db.WhatsappMessageCategoryMarketing {
				t.Errorf("categoria incorreta: %v", arg.Category)
			}
			if arg.TemplateID.Valid {
				t.Errorf("templateID deveria ser nulo no modo own_waba")
			}
			return db.WhatsappMessage{
				ID:                 pgconv.ParseUUIDToPgType(messageID),
				CompanyID:          pgconv.ParseUUIDToPgType(companyID),
				Category:           arg.Category,
				RecipientPhone:     arg.RecipientPhone,
				MetaMessageID:      arg.MetaMessageID,
				Status:             arg.Status,
				EstimatedCostCents: arg.EstimatedCostCents,
				CreatedAt:          pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil
		})

	msg, err := svc.SendMessage(context.Background(), companyID, req)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if msg.ID != messageID {
		t.Errorf("ID incorreto")
	}
	if msg.TemplateID != nil {
		t.Errorf("TemplateID deveria ser nil, obteve %v", msg.TemplateID)
	}
}

func TestSendMessage_OwnWABA_DecryptError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	companyConfig := buildCompanyConfig(uuid.New(), companyID, db.WhatsappModeOwnWaba)
	companyConfig.AccessTokenEncrypted = pgtype.Text{String: "nao-e-base64-valido!!", Valid: true}

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(companyConfig, nil)

	_, err := svc.SendMessage(context.Background(), companyID, domain.SendMessageRequest{
		TemplateName: "custom_template",
		LanguageCode: "pt_BR",
	})

	if err == nil {
		t.Fatal("esperava erro ao descriptografar o access token")
	}
}

func TestSendMessage_MetaClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	companyConfig := buildCompanyConfig(uuid.New(), companyID, db.WhatsappModePlatformShared)

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(companyConfig, nil)

	d.repo.EXPECT().
		GetEligibleTemplateByName(gomock.Any(), gomock.Any()).
		Return(db.WhatsappTemplate{MetaTemplateName: "meu_template"}, nil)

	d.metaClient.EXPECT().
		SendTemplateMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("", errors.New("falha na Graph API"))

	_, err := svc.SendMessage(context.Background(), companyID, domain.SendMessageRequest{
		TemplateName: "meu_template",
		LanguageCode: "pt_BR",
	})

	if err == nil {
		t.Fatal("esperava erro do metaClient")
	}
}

func TestSendMessage_CreateWhatsAppMessageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	companyConfig := buildCompanyConfig(uuid.New(), companyID, db.WhatsappModePlatformShared)

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(companyConfig, nil)

	d.repo.EXPECT().
		GetEligibleTemplateByName(gomock.Any(), gomock.Any()).
		Return(db.WhatsappTemplate{MetaTemplateName: "meu_template"}, nil)

	d.metaClient.EXPECT().
		SendTemplateMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("wamid.x", nil)

	d.repo.EXPECT().
		CreateWhatsAppMessage(gomock.Any(), gomock.Any()).
		Return(db.WhatsappMessage{}, errDatabase)

	_, err := svc.SendMessage(context.Background(), companyID, domain.SendMessageRequest{
		TemplateName: "meu_template",
		LanguageCode: "pt_BR",
	})

	if err == nil {
		t.Fatal("esperava erro ao criar mensagem")
	}
}

// ---------------------------------------------------------------------------
// GetCompanyConfig
// ---------------------------------------------------------------------------

func TestGetCompanyConfig_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	configID := uuid.New()
	companyConfig := buildCompanyConfig(configID, companyID, db.WhatsappModeOwnWaba)

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), pgconv.OptionalUUIDToPgType(companyID)).
		Return(companyConfig, nil)

	resp, err := svc.GetCompanyConfig(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != configID {
		t.Errorf("ID incorreto")
	}
	if resp.Mode != domain.WhatsAppModeOwnWABA {
		t.Errorf("Mode incorreto: %s", resp.Mode)
	}
	if resp.WABAID == nil || *resp.WABAID != "waba-123" {
		t.Errorf("WABAID incorreto")
	}
	if resp.MonthlyMessageAllowance != 1000 {
		t.Errorf("MonthlyMessageAllowance incorreto: %d", resp.MonthlyMessageAllowance)
	}
}

func TestGetCompanyConfig_NullableFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(db.CompanyWhatsappConfig{
			Mode:               db.WhatsappModePlatformShared,
			WabaID:             pgtype.Text{Valid: false},
			PhoneNumberID:      pgtype.Text{Valid: false},
			DisplayPhoneNumber: pgtype.Text{Valid: false},
		}, nil)

	resp, err := svc.GetCompanyConfig(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.WABAID != nil {
		t.Errorf("WABAID deveria ser nil")
	}
	if resp.PhoneNumberID != nil {
		t.Errorf("PhoneNumberID deveria ser nil")
	}
	if resp.DisplayPhoneNumber != nil {
		t.Errorf("DisplayPhoneNumber deveria ser nil")
	}
}

func TestGetCompanyConfig_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		GetCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(db.CompanyWhatsappConfig{}, errDatabase)

	_, err := svc.GetCompanyConfig(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// UpsertCompanyConfig
// ---------------------------------------------------------------------------

func TestUpsertCompanyConfig_Success_WithoutAccessToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	waba := "waba-999"

	d.repo.EXPECT().
		UpsertCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpsertCompanyWhatsAppConfigParams) (db.CompanyWhatsappConfig, error) {
			if arg.AccessTokenEncrypted.Valid {
				t.Errorf("AccessTokenEncrypted deveria ser inválido quando não informado")
			}
			if arg.WabaID.String != waba {
				t.Errorf("WabaID incorreto: %s", arg.WabaID.String)
			}
			return db.CompanyWhatsappConfig{
				ID:        pgconv.ParseUUIDToPgType(uuid.New()),
				CompanyID: pgconv.ParseUUIDToPgType(companyID),
				Mode:      arg.Mode,
				WabaID:    arg.WabaID,
			}, nil
		})

	// Modo own_waba não deve disparar a integração de billing/subscription
	// (nenhum EXPECT registrado em d.subscriptionSvc/d.billingClient).
	resp, err := svc.UpsertCompanyConfig(context.Background(), companyID, domain.UpsertCompanyConfigRequest{
		Mode:   domain.WhatsAppModeOwnWABA,
		WABAID: &waba,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.WABAID == nil || *resp.WABAID != waba {
		t.Errorf("WABAID incorreto na resposta")
	}
}

func TestUpsertCompanyConfig_Success_WithAccessToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	plaintextToken := "novo-access-token"

	d.repo.EXPECT().
		UpsertCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.UpsertCompanyWhatsAppConfigParams) (db.CompanyWhatsappConfig, error) {
			if !arg.AccessTokenEncrypted.Valid {
				t.Fatal("AccessTokenEncrypted deveria estar presente")
			}
			decrypted, err := crypto.Decrypt(testSecretKey, arg.AccessTokenEncrypted.String)
			if err != nil {
				t.Fatalf("falha ao descriptografar o token gerado pelo service: %v", err)
			}
			if decrypted != plaintextToken {
				t.Errorf("token criptografado incorretamente: %s", decrypted)
			}
			return db.CompanyWhatsappConfig{
				ID:        pgconv.ParseUUIDToPgType(uuid.New()),
				CompanyID: pgconv.ParseUUIDToPgType(companyID),
				Mode:      arg.Mode,
			}, nil
		})

	_, err := svc.UpsertCompanyConfig(context.Background(), companyID, domain.UpsertCompanyConfigRequest{
		Mode:        domain.WhatsAppModeOwnWABA,
		AccessToken: &plaintextToken,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestUpsertCompanyConfig_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		UpsertCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(db.CompanyWhatsappConfig{}, errDatabase)

	_, err := svc.UpsertCompanyConfig(context.Background(), uuid.New(), domain.UpsertCompanyConfigRequest{
		Mode: domain.WhatsAppModePlatformShared,
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestUpsertCompanyConfig_PlatformShared_AttachesOverageItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	externalSubID := "sub_stripe_123"

	d.repo.EXPECT().
		UpsertCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(db.CompanyWhatsappConfig{
			ID:        pgconv.ParseUUIDToPgType(uuid.New()),
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
			Mode:      db.WhatsappModePlatformShared,
		}, nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), companyID).
		Return(subscriptionsDomain.SubscriptionResponse{ExternalSubscriptionID: externalSubID}, nil)

	d.billingClient.EXPECT().
		AttachOverageItem(gomock.Any(), externalSubID, "price_whatsapp_overage").
		Return(nil)

	resp, err := svc.UpsertCompanyConfig(context.Background(), companyID, domain.UpsertCompanyConfigRequest{
		Mode: domain.WhatsAppModePlatformShared,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Mode != domain.WhatsAppModePlatformShared {
		t.Errorf("Mode incorreto: %s", resp.Mode)
	}
}

func TestUpsertCompanyConfig_PlatformShared_SubscriptionServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		UpsertCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(db.CompanyWhatsappConfig{Mode: db.WhatsappModePlatformShared}, nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{}, errors.New("assinatura não encontrada"))

	_, err := svc.UpsertCompanyConfig(context.Background(), uuid.New(), domain.UpsertCompanyConfigRequest{
		Mode: domain.WhatsAppModePlatformShared,
	})

	if err == nil {
		t.Fatal("esperava erro ao buscar subscription")
	}
}

func TestUpsertCompanyConfig_PlatformShared_AttachOverageItemError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		UpsertCompanyWhatsAppConfig(gomock.Any(), gomock.Any()).
		Return(db.CompanyWhatsappConfig{Mode: db.WhatsappModePlatformShared}, nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{ExternalSubscriptionID: "sub_x"}, nil)

	d.billingClient.EXPECT().
		AttachOverageItem(gomock.Any(), "sub_x", "price_whatsapp_overage").
		Return(errors.New("falha ao anexar item no Stripe"))

	_, err := svc.UpsertCompanyConfig(context.Background(), uuid.New(), domain.UpsertCompanyConfigRequest{
		Mode: domain.WhatsAppModePlatformShared,
	})

	if err == nil {
		t.Fatal("esperava erro ao anexar item de billing")
	}
}

// ---------------------------------------------------------------------------
// ListApprovedTemplates
// ---------------------------------------------------------------------------

func TestListApprovedTemplates_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		ListApprovedTemplates(gomock.Any()).
		Return([]db.WhatsappTemplate{
			{
				ID:                       pgconv.ParseUUIDToPgType(uuid.New()),
				MetaTemplateName:         "template_1",
				Category:                 db.WhatsappMessageCategoryUtility,
				LanguageCode:             "pt_BR",
				BodyText:                 "Olá {{1}}",
				Variables:                []byte(`["1"]`),
				IsPlatformSharedEligible: true,
				MetaApprovalStatus:       "approved",
			},
		}, nil)

	resp, err := svc.ListApprovedTemplates(context.Background())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("esperava 1 template, obteve %d", len(resp))
	}
	if resp[0].MetaTemplateName != "template_1" {
		t.Errorf("nome incorreto: %s", resp[0].MetaTemplateName)
	}
	if len(resp[0].Variables) != 1 || resp[0].Variables[0] != "1" {
		t.Errorf("variables incorretas: %v", resp[0].Variables)
	}
}

func TestListApprovedTemplates_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		ListApprovedTemplates(gomock.Any()).
		Return([]db.WhatsappTemplate{}, nil)

	resp, err := svc.ListApprovedTemplates(context.Background())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("esperava lista vazia, obteve %d", len(resp))
	}
}

func TestListApprovedTemplates_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		ListApprovedTemplates(gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListApprovedTemplates(context.Background())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestListApprovedTemplates_InvalidVariablesJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		ListApprovedTemplates(gomock.Any()).
		Return([]db.WhatsappTemplate{
			{MetaTemplateName: "quebrado", Variables: []byte(`nao-e-json`)},
		}, nil)

	_, err := svc.ListApprovedTemplates(context.Background())

	if err == nil {
		t.Fatal("esperava erro ao decodificar variables")
	}
}

// ---------------------------------------------------------------------------
// HandleWebhookEvent
// ---------------------------------------------------------------------------

func TestHandleWebhookEvent_Success_NoErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	payload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "entry-1",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"statuses": [{
						"id": "wamid.abc",
						"status": "delivered",
						"timestamp": "123456",
						"recipient_id": "5511999999999"
					}]
				}
			}]
		}]
	}`)

	d.repo.EXPECT().
		UpdateWhatsAppMessageStatus(gomock.Any(), db.UpdateWhatsAppMessageStatusParams{
			MetaMessageID: pgconv.ParseStringToPgText("wamid.abc"),
			Status:        db.WhatsappMessageStatusDelivered,
			FailureCode:   pgconv.IntToPgInt4(0),
			FailureReason: pgconv.ParseStringToPgText(""),
		}).
		Return(db.WhatsappMessage{}, nil)

	err := svc.HandleWebhookEvent(context.Background(), payload)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestHandleWebhookEvent_Success_WithFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	payload := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "entry-1",
			"changes": [{
				"field": "messages",
				"value": {
					"statuses": [{
						"id": "wamid.failed",
						"status": "failed",
						"errors": [{"code": 131047, "title": "Erro", "message": "Mensagem falhou"}]
					}]
				}
			}]
		}]
	}`)

	d.repo.EXPECT().
		UpdateWhatsAppMessageStatus(gomock.Any(), db.UpdateWhatsAppMessageStatusParams{
			MetaMessageID: pgconv.ParseStringToPgText("wamid.failed"),
			Status:        db.WhatsappMessageStatusFailed,
			FailureCode:   pgconv.IntToPgInt4(131047),
			FailureReason: pgconv.ParseStringToPgText("Mensagem falhou"),
		}).
		Return(db.WhatsappMessage{}, nil)

	err := svc.HandleWebhookEvent(context.Background(), payload)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestHandleWebhookEvent_MultipleStatuses(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	payload := []byte(`{
		"entry": [{
			"changes": [{
				"value": {
					"statuses": [
						{"id": "wamid.1", "status": "sent"},
						{"id": "wamid.2", "status": "read"}
					]
				}
			}]
		}]
	}`)

	d.repo.EXPECT().
		UpdateWhatsAppMessageStatus(gomock.Any(), gomock.Any()).
		Return(db.WhatsappMessage{}, nil).
		Times(2)

	err := svc.HandleWebhookEvent(context.Background(), payload)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestHandleWebhookEvent_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	err := svc.HandleWebhookEvent(context.Background(), []byte(`{invalido`))

	if err == nil {
		t.Fatal("esperava erro de JSON inválido")
	}
}

func TestHandleWebhookEvent_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	payload := []byte(`{
		"entry": [{
			"changes": [{
				"value": {
					"statuses": [{"id": "wamid.err", "status": "sent"}]
				}
			}]
		}]
	}`)

	d.repo.EXPECT().
		UpdateWhatsAppMessageStatus(gomock.Any(), gomock.Any()).
		Return(db.WhatsappMessage{}, errDatabase)

	err := svc.HandleWebhookEvent(context.Background(), payload)

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestHandleWebhookEvent_NoStatuses_NoRepositoryCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	// Nenhum EXPECT registrado em d.repo: se UpdateWhatsAppMessageStatus for
	// chamado, o teste falha automaticamente.
	err := svc.HandleWebhookEvent(context.Background(), []byte(`{"object":"whatsapp_business_account","entry":[]}`))

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SyncMonthlyUsage
// ---------------------------------------------------------------------------

func buildPlanResponse(planID uuid.UUID, featureKey string, limit int64) plansDomain.PlanResponse {
	return plansDomain.PlanResponse{
		ID: planID,
		Features: []plansFeaturesDomain.PlanFeatureResponse{
			{FeatureKey: featureKey, LimitValue: limit},
		},
	}
}

func TestSyncMonthlyUsage_WithinAllowance_DoesNotReportUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	planID := uuid.New()
	periodStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(50), nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), companyID).
		Return(subscriptionsDomain.SubscriptionResponse{PlanID: planID}, nil)

	d.plansSvc.EXPECT().
		GetPlanByID(gomock.Any(), planID).
		Return(buildPlanResponse(planID, "whatsapp_integration", 1000), nil)

	d.repo.EXPECT().
		UpsertMonthlyUsage(gomock.Any(), db.UpsertMonthlyUsageParams{
			CompanyID:             pgconv.OptionalUUIDToPgType(companyID),
			BillingPeriod:         pgconv.StringToPgDate(periodStart.String()),
			MessagesSent:          50,
			MessagesOverAllowance: 0,
		}).
		Return(db.WhatsappUsageMonthly{}, nil)

	// GetCompanyByID é chamado incondicionalmente pelo service.
	d.companiesSvc.EXPECT().
		GetCompanyByID(gomock.Any(), companyID).
		Return(companiesDomain.CompanyResponse{ExternalCompanyId: "cus_123"}, nil)

	// billingClient não deve ser chamado: nenhum EXPECT registrado.

	err := svc.SyncMonthlyUsage(context.Background(), companyID, periodStart, periodEnd)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestSyncMonthlyUsage_OverAllowance_ReportsUsage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	planID := uuid.New()
	periodStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(1200), nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), companyID).
		Return(subscriptionsDomain.SubscriptionResponse{PlanID: planID}, nil)

	d.plansSvc.EXPECT().
		GetPlanByID(gomock.Any(), planID).
		Return(buildPlanResponse(planID, "whatsapp_integration", 1000), nil)

	d.repo.EXPECT().
		UpsertMonthlyUsage(gomock.Any(), db.UpsertMonthlyUsageParams{
			CompanyID:             pgconv.OptionalUUIDToPgType(companyID),
			BillingPeriod:         pgconv.StringToPgDate(periodStart.String()),
			MessagesSent:          1200,
			MessagesOverAllowance: 200,
		}).
		Return(db.WhatsappUsageMonthly{}, nil)

	d.companiesSvc.EXPECT().
		GetCompanyByID(gomock.Any(), companyID).
		Return(companiesDomain.CompanyResponse{ExternalCompanyId: "cus_123"}, nil)

	expectedIdempotencyKey := "whatsapp-overage-" + companyID.String() + "-" + periodStart.Format("2006-01-02")

	d.billingClient.EXPECT().
		ReportUsage(gomock.Any(), "cus_123", 200, expectedIdempotencyKey).
		Return(nil)

	err := svc.SyncMonthlyUsage(context.Background(), companyID, periodStart, periodEnd)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestSyncMonthlyUsage_CountMessagesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	err := svc.SyncMonthlyUsage(context.Background(), uuid.New(), time.Now(), time.Now())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestSyncMonthlyUsage_SubscriptionServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(10), nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{}, errors.New("assinatura não encontrada"))

	err := svc.SyncMonthlyUsage(context.Background(), uuid.New(), time.Now(), time.Now())

	if err == nil {
		t.Fatal("esperava erro do serviço de assinaturas")
	}
}

func TestSyncMonthlyUsage_PlansServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(10), nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{PlanID: uuid.New()}, nil)

	d.plansSvc.EXPECT().
		GetPlanByID(gomock.Any(), gomock.Any()).
		Return(plansDomain.PlanResponse{}, errors.New("plano não encontrado"))

	err := svc.SyncMonthlyUsage(context.Background(), uuid.New(), time.Now(), time.Now())

	if err == nil {
		t.Fatal("esperava erro do serviço de planos")
	}
}

func TestSyncMonthlyUsage_UpsertMonthlyUsageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	planID := uuid.New()

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(10), nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{PlanID: planID}, nil)

	d.plansSvc.EXPECT().
		GetPlanByID(gomock.Any(), gomock.Any()).
		Return(buildPlanResponse(planID, "whatsapp_integration", 1000), nil)

	d.repo.EXPECT().
		UpsertMonthlyUsage(gomock.Any(), gomock.Any()).
		Return(db.WhatsappUsageMonthly{}, errDatabase)

	err := svc.SyncMonthlyUsage(context.Background(), uuid.New(), time.Now(), time.Now())

	if err == nil {
		t.Fatal("esperava erro ao salvar uso mensal")
	}
}

func TestSyncMonthlyUsage_CompaniesServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	planID := uuid.New()

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(10), nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{PlanID: planID}, nil)

	d.plansSvc.EXPECT().
		GetPlanByID(gomock.Any(), gomock.Any()).
		Return(buildPlanResponse(planID, "whatsapp_integration", 1000), nil)

	d.repo.EXPECT().
		UpsertMonthlyUsage(gomock.Any(), gomock.Any()).
		Return(db.WhatsappUsageMonthly{}, nil)

	d.companiesSvc.EXPECT().
		GetCompanyByID(gomock.Any(), gomock.Any()).
		Return(companiesDomain.CompanyResponse{}, errors.New("empresa não encontrada"))

	err := svc.SyncMonthlyUsage(context.Background(), uuid.New(), time.Now(), time.Now())

	if err == nil {
		t.Fatal("esperava erro do serviço de empresas")
	}
}

func TestSyncMonthlyUsage_BillingClientError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	planID := uuid.New()

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(2000), nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{PlanID: planID}, nil)

	d.plansSvc.EXPECT().
		GetPlanByID(gomock.Any(), gomock.Any()).
		Return(buildPlanResponse(planID, "whatsapp_integration", 1000), nil)

	d.repo.EXPECT().
		UpsertMonthlyUsage(gomock.Any(), gomock.Any()).
		Return(db.WhatsappUsageMonthly{}, nil)

	d.companiesSvc.EXPECT().
		GetCompanyByID(gomock.Any(), gomock.Any()).
		Return(companiesDomain.CompanyResponse{ExternalCompanyId: "cus_123"}, nil)

	d.billingClient.EXPECT().
		ReportUsage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("falha ao reportar uso pro Stripe"))

	err := svc.SyncMonthlyUsage(context.Background(), companyID, time.Now(), time.Now())

	if err == nil {
		t.Fatal("esperava erro do billingClient")
	}
}

func TestSyncMonthlyUsage_FeatureNotFound_UsesFullCountAsOverage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(d)

	companyID := uuid.New()
	planID := uuid.New()
	periodStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	d.repo.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any()).
		Return(int64(30), nil)

	d.subscriptionSvc.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{PlanID: planID}, nil)

	// Plano sem a feature "whatsapp_integration": quantityMessage fica 0.
	d.plansSvc.EXPECT().
		GetPlanByID(gomock.Any(), gomock.Any()).
		Return(buildPlanResponse(planID, "other_feature", 500), nil)

	d.repo.EXPECT().
		UpsertMonthlyUsage(gomock.Any(), db.UpsertMonthlyUsageParams{
			CompanyID:             pgconv.OptionalUUIDToPgType(companyID),
			BillingPeriod:         pgconv.StringToPgDate(periodStart.String()),
			MessagesSent:          30,
			MessagesOverAllowance: 30,
		}).
		Return(db.WhatsappUsageMonthly{}, nil)

	d.companiesSvc.EXPECT().
		GetCompanyByID(gomock.Any(), gomock.Any()).
		Return(companiesDomain.CompanyResponse{ExternalCompanyId: "cus_123"}, nil)

	d.billingClient.EXPECT().
		ReportUsage(gomock.Any(), "cus_123", 30, gomock.Any()).
		Return(nil)

	err := svc.SyncMonthlyUsage(context.Background(), companyID, periodStart, periodEnd)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}
