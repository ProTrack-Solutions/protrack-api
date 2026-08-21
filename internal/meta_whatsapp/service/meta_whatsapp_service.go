package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/crypto"
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/google/uuid"
)

type Service struct {
	repo                         domain.RepositoryInterface
	metaClient                   domain.MetaClientInterface
	subscriptionService          domain.SubscriptionsServiceInterface
	plansService                 domain.PlansServiceInterface
	billingClient                domain.BillingClientInterface
	companiesService             domain.CompaniesServiceInterface
	PhoneNumberID                string
	MetaAccessToken              string
	MetaSecretKey                string
	StripeWhatsappOveragePriceID string
}

func NewService(
	repo domain.RepositoryInterface,
	metaClient domain.MetaClientInterface,
	subscriptionService domain.SubscriptionsServiceInterface,
	plansService domain.PlansServiceInterface,
	billingClient domain.BillingClientInterface,
	companiesService domain.CompaniesServiceInterface,
	cfg *config.Config,
) *Service {
	return &Service{
		repo:                         repo,
		metaClient:                   metaClient,
		PhoneNumberID:                cfg.PhoneNumberID,
		MetaAccessToken:              cfg.MetaAccessToken,
		MetaSecretKey:                cfg.MetaSecretKey,
		subscriptionService:          subscriptionService,
		plansService:                 plansService,
		billingClient:                billingClient,
		companiesService:             companiesService,
		StripeWhatsappOveragePriceID: cfg.StripeWhatsappOveragePriceID,
	}
}

func estimateCostCents(category domain.MessageCategory) int {
	switch category {
	case domain.CategoryMarketing:
		return 35 // valor placeholder, em centavos
	case domain.CategoryUtility, domain.CategoryAuthentication:
		return 4
	default:
		return 0
	}
}

func (s *Service) SendMessage(ctx context.Context, companyId uuid.UUID, req domain.SendMessageRequest) (*domain.Message, error) {
	configCompany, err := s.repo.GetCompanyWhatsAppConfig(ctx, pgconv.OptionalUUIDToPgType(companyId))
	if err != nil {
		return &domain.Message{}, err
	}

	var phoneNumberId string
	var accessToken string
	var templateId *uuid.UUID
	var category domain.MessageCategory

	if configCompany.Mode == db.WhatsappMode(domain.WhatsAppModePlatformShared) {
		template, err := s.repo.GetEligibleTemplateByName(ctx, db.GetEligibleTemplateByNameParams{
			MetaTemplateName: req.TemplateName,
			LanguageCode:     req.LanguageCode,
		})
		if err != nil {
			return &domain.Message{}, err
		}

		if req.TemplateName != template.MetaTemplateName {
			return &domain.Message{}, errors.New("Template not register")
		}

		phoneNumberId = s.PhoneNumberID
		accessToken = s.MetaAccessToken
		templateId = pgconv.PgUUIDToUUIDPtr(template.ID)
		category = domain.MessageCategory(template.Category)
	} else {
		phoneNumberId = configCompany.PhoneNumberID.String
		decryptAcessToken, err := crypto.Decrypt(s.MetaSecretKey, configCompany.AccessTokenEncrypted.String)
		if err != nil {
			return &domain.Message{}, err
		}
		accessToken = decryptAcessToken
		category = req.Category
	}

	metaMessageId, err := s.metaClient.SendTemplateMessage(ctx, phoneNumberId, accessToken, req)
	if err != nil {
		return &domain.Message{}, err
	}

	menssage, err := s.repo.CreateWhatsAppMessage(ctx, db.CreateWhatsAppMessageParams{
		CompanyID:          pgconv.OptionalUUIDToPgType(companyId),
		TemplateID:         pgconv.OptionalPtrUUIDToPgType(templateId),
		Category:           db.WhatsappMessageCategory(category),
		RecipientPhone:     req.RecipientPhone,
		MetaMessageID:      pgconv.ParseStringToPgText(metaMessageId),
		Status:             db.WhatsappMessageStatus(domain.StatusSent),
		EstimatedCostCents: pgconv.IntToPgInt4(estimateCostCents(category)),
	})
	if err != nil {
		return &domain.Message{}, err
	}

	return &domain.Message{
		ID:                 pgconv.PgUUIDToUUID(menssage.ID),
		CompanyID:          pgconv.PgUUIDToUUID(menssage.CompanyID),
		TemplateID:         pgconv.PgUUIDToUUIDPtr(menssage.TemplateID),
		Category:           domain.MessageCategory(menssage.Category),
		RecipientPhone:     menssage.RecipientPhone,
		MetaMessageID:      pgconv.ParsePgTextToStringPtr(menssage.MetaMessageID),
		Status:             domain.MessageStatus(menssage.Status),
		FailureCode:        pgconv.PgInt4ToIntPtr(menssage.FailureCode),
		FailureReason:      pgconv.ParsePgTextToStringPtr(menssage.FailureReason),
		EstimatedCostCents: pgconv.PgInt4ToIntPtr(menssage.EstimatedCostCents),
		SentAt:             pgconv.PgTimestamptzToTimePtr(menssage.SentAt),
		DeliveredAt:        pgconv.PgTimestamptzToTimePtr(menssage.DeliveredAt),
		CreatedAt:          menssage.CreatedAt.Time,
	}, nil
}

func (s *Service) GetCompanyConfig(ctx context.Context, companyId uuid.UUID) (*domain.CompanyWhatsAppConfig, error) {
	companyConfig, err := s.repo.GetCompanyWhatsAppConfig(ctx, pgconv.OptionalUUIDToPgType(companyId))
	if err != nil {
		return &domain.CompanyWhatsAppConfig{}, err
	}

	return &domain.CompanyWhatsAppConfig{
		ID:                      pgconv.PgUUIDToUUID(companyConfig.ID),
		CompanyID:               pgconv.PgUUIDToUUID(companyConfig.CompanyID),
		Mode:                    domain.WhatsAppMode(companyConfig.Mode),
		WABAID:                  pgconv.ParsePgTextToStringPtr(companyConfig.WabaID),
		PhoneNumberID:           pgconv.ParsePgTextToStringPtr(companyConfig.PhoneNumberID),
		DisplayPhoneNumber:      pgconv.ParsePgTextToStringPtr(companyConfig.DisplayPhoneNumber),
		MonthlyMessageAllowance: int(companyConfig.MonthlyMessageAllowance),
		IsActive:                companyConfig.IsActive,
		CreatedAt:               pgconv.PgTimestamptzToTime(companyConfig.CreatedAt),
		UpdatedAt:               pgconv.PgTimestamptzToTime(companyConfig.UpdatedAt),
	}, nil
}

func (s *Service) UpsertCompanyConfig(ctx context.Context, companyId uuid.UUID, req domain.UpsertCompanyConfigRequest) (*domain.CompanyWhatsAppConfig, error) {
	var accessTokenEncryptedPtr *string

	if req.AccessToken != nil {
		accessTokenEncrypted, err := crypto.Encrypt(s.MetaSecretKey, *req.AccessToken)
		if err != nil {
			return &domain.CompanyWhatsAppConfig{}, err
		}

		accessTokenEncryptedPtr = &accessTokenEncrypted
	}

	config, err := s.repo.UpsertCompanyWhatsAppConfig(ctx, db.UpsertCompanyWhatsAppConfigParams{
		CompanyID:               pgconv.OptionalUUIDToPgType(companyId),
		Mode:                    db.WhatsappMode(req.Mode),
		WabaID:                  pgconv.ParseStringPtrToPgText(req.WABAID),
		PhoneNumberID:           pgconv.ParseStringPtrToPgText(req.PhoneNumberId),
		DisplayPhoneNumber:      pgconv.ParseStringPtrToPgText(req.DisplayPhoneNumber),
		AccessTokenEncrypted:    pgconv.ParseStringPtrToPgText(accessTokenEncryptedPtr),
		MonthlyMessageAllowance: int32(req.MonthlyMessageAllowance),
		IsActive:                req.IsActive,
	})
	if err != nil {
		return &domain.CompanyWhatsAppConfig{}, err
	}

	if req.Mode == domain.WhatsAppModePlatformShared {
		sub, err := s.subscriptionService.GetSubscriptionByCompanyID(ctx, companyId)
		if err != nil {
			return &domain.CompanyWhatsAppConfig{}, fmt.Errorf("erro ao buscar subscription para anexar item de billing: %w", err)
		}

		if err := s.billingClient.AttachOverageItem(ctx, sub.ExternalSubscriptionID, s.StripeWhatsappOveragePriceID); err != nil {
			return &domain.CompanyWhatsAppConfig{}, fmt.Errorf("erro ao anexar item de billing do WhatsApp: %w", err)
		}
	}

	return &domain.CompanyWhatsAppConfig{
		ID:                      pgconv.PgUUIDToUUID(config.ID),
		CompanyID:               pgconv.PgUUIDToUUID(config.CompanyID),
		Mode:                    domain.WhatsAppMode(config.Mode),
		WABAID:                  pgconv.ParsePgTextToStringPtr(config.WabaID),
		PhoneNumberID:           pgconv.ParsePgTextToStringPtr(config.PhoneNumberID),
		DisplayPhoneNumber:      pgconv.ParsePgTextToStringPtr(config.DisplayPhoneNumber),
		MonthlyMessageAllowance: int(config.MonthlyMessageAllowance),
		IsActive:                config.IsActive,
		CreatedAt:               pgconv.PgTimestamptzToTime(config.CreatedAt),
		UpdatedAt:               pgconv.PgTimestamptzToTime(config.UpdatedAt),
	}, nil
}

func (s *Service) ListApprovedTemplates(ctx context.Context) ([]domain.Template, error) {
	template, err := s.repo.ListApprovedTemplates(ctx)
	if err != nil {
		return []domain.Template{}, err
	}

	var response []domain.Template

	for _, tp := range template {

		var variables []string
		if err := json.Unmarshal(tp.Variables, &variables); err != nil {
			return []domain.Template{}, err
		}
		response = append(response, domain.Template{
			ID:                       pgconv.PgUUIDToUUID(tp.ID),
			MetaTemplateName:         tp.MetaTemplateName,
			Category:                 domain.MessageCategory(tp.Category),
			LanguageCode:             tp.LanguageCode,
			BodyText:                 tp.BodyText,
			IsPlatformSharedEligible: tp.IsPlatformSharedEligible,
			MetaApprovalStatus:       tp.MetaApprovalStatus,
			CreatedAt:                pgconv.PgTimestamptzToTime(tp.CreatedAt),
			UpdatedAt:                pgconv.PgTimestamptzToTime(tp.UpdatedAt),
			Variables:                variables,
		})

	}

	return response, nil
}

func (s *Service) HandleWebhookEvent(ctx context.Context, payload []byte) error {
	var webhookPayload domain.WebhookPayload

	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		return err
	}

	for _, entry := range webhookPayload.Entry {
		for _, change := range entry.Changes {
			for _, status := range change.Value.Statuses {
				var failureCode int
				var failureReason string

				if len(status.Errors) > 0 {

					failureCode = status.Errors[0].Code
					failureReason = status.Errors[0].Message

				}

				if _, err := s.repo.UpdateWhatsAppMessageStatus(ctx, db.UpdateWhatsAppMessageStatusParams{
					MetaMessageID: pgconv.ParseStringToPgText(status.Id),
					Status:        db.WhatsappMessageStatus(status.Status),
					FailureCode:   pgconv.IntToPgInt4(failureCode),
					FailureReason: pgconv.ParseStringToPgText(failureReason),
				}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *Service) SyncMonthlyUsage(ctx context.Context, companyId uuid.UUID, periodStart, periodEnd time.Time) error {
	countMessage, err := s.repo.CountMessagesInPeriod(ctx, db.CountMessagesInPeriodParams{
		CompanyID:   pgconv.OptionalUUIDToPgType(companyId),
		CreatedAt:   pgconv.TimeToPgTimestamptz(periodStart),
		CreatedAt_2: pgconv.TimeToPgTimestamptz(periodEnd),
	})
	if err != nil {
		return err
	}

	sub, err := s.subscriptionService.GetSubscriptionByCompanyID(ctx, companyId)
	if err != nil {
		return err
	}

	plan, err := s.plansService.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return err
	}

	var quantityMessage int

	for _, ft := range plan.Features {
		if ft.FeatureKey == "max_whatsapp_integration" {
			quantityMessage = int(ft.LimitValue)
		}
	}

	restMessage := int(countMessage) - quantityMessage

	overAllowance := 0
	if restMessage > 0 {
		overAllowance = restMessage
	}

	_, err = s.repo.UpsertMonthlyUsage(ctx, db.UpsertMonthlyUsageParams{
		CompanyID:             pgconv.OptionalUUIDToPgType(companyId),
		BillingPeriod:         pgconv.StringToPgDate(periodStart.String()),
		MessagesSent:          int32(countMessage),
		MessagesOverAllowance: int32(overAllowance),
	})
	if err != nil {
		return err
	}

	idempotencyKey := fmt.Sprintf("whatsapp-overage-%s-%s", companyId, periodStart.Format("2006-01-02"))

	company, err := s.companiesService.GetCompanyByID(ctx, companyId)
	if err != nil {
		return err
	}

	if overAllowance > 0 {
		if err = s.billingClient.ReportUsage(ctx, company.ExternalCompanyId, overAllowance, idempotencyKey); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) CountMessagesInPeriod(ctx context.Context, companyId uuid.UUID, periodStart, periodEnd time.Time) (*int64, error) {
	countMessage, err := s.repo.CountMessagesInPeriod(ctx, db.CountMessagesInPeriodParams{
		CompanyID:   pgconv.OptionalUUIDToPgType(companyId),
		CreatedAt:   pgconv.TimeToPgTimestamptz(periodStart),
		CreatedAt_2: pgconv.TimeToPgTimestamptz(periodEnd),
	})
	if err != nil {
		return nil, err
	}

	return &countMessage, err
}

func (s *Service) GetEligibleTemplateByName(ctx context.Context, key enums.CompanySettingsKey, LanguageCode string) (domain.Template, error) {
	template, err := s.repo.GetEligibleTemplateByName(ctx, db.GetEligibleTemplateByNameParams{
		MetaTemplateName: string(key),
		LanguageCode:     LanguageCode,
	})
	if err != nil {
		return domain.Template{}, err
	}

	var variables []string
	if err := json.Unmarshal(template.Variables, &variables); err != nil {
		return domain.Template{}, err
	}

	return domain.Template{
		ID:                       pgconv.PgUUIDToUUID(template.ID),
		MetaTemplateName:         template.MetaTemplateName,
		Category:                 domain.MessageCategory(template.Category),
		LanguageCode:             template.LanguageCode,
		BodyText:                 template.BodyText,
		Variables:                variables,
		IsPlatformSharedEligible: template.IsPlatformSharedEligible,
		MetaApprovalStatus:       template.MetaApprovalStatus,
		CreatedAt:                pgconv.PgTimestamptzToTime(template.CreatedAt),
		UpdatedAt:                pgconv.PgTimestamptzToTime(template.UpdatedAt),
	}, nil
}
