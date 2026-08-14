package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/crypto"
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"
	"github.com/google/uuid"
)

type Service struct {
	repo            domain.RepositoryInterface
	metaClient      domain.MetaClientInterface
	PhoneNumberID   string
	MetaAccessToken string
	MetaSecretKey   string
}

func NewService(repo domain.RepositoryInterface, metaClient domain.MetaClientInterface, cfg *config.Config) *Service {
	return &Service{
		repo:            repo,
		metaClient:      metaClient,
		PhoneNumberID:   cfg.PhoneNumberID,
		MetaAccessToken: cfg.MetaAccessToken,
		MetaSecretKey:   cfg.MetaSecretKey,
	}
}

func estimateCostCents(category domain.MessageCategory) int {
	switch category {
	case domain.CategoryMarketing:
		return 250 // valor placeholder, em centavos
	case domain.CategoryUtility, domain.CategoryAuthentication:
		return 50
	default:
		return 0
	}
}

func (s *Service) SendMessage(ctx context.Context, req domain.SendMessageRequest) (*domain.Message, error) {
	configCompany, err := s.repo.GetCompanyWhatsAppConfig(ctx, pgconv.OptionalUUIDToPgType(req.CompanyID))
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
		CompanyID:          pgconv.OptionalUUIDToPgType(req.CompanyID),
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
