// Salve como internal/whatsapp/domain/models.go no protrack-api.
package domain

import (
	"context"
	"time"

	companiesDomain "github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	plansDomain "github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	subscriptionsDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RepositoryInterface interface {
	CountMessagesInPeriod(ctx context.Context, arg db.CountMessagesInPeriodParams) (int64, error)
	CreateWhatsAppMessage(ctx context.Context, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error)
	GetCompanyWhatsAppConfig(ctx context.Context, companyId pgtype.UUID) (db.CompanyWhatsappConfig, error)
	GetEligibleTemplateByName(ctx context.Context, arg db.GetEligibleTemplateByNameParams) (db.WhatsappTemplate, error)
	GetMessageByMetaID(ctx context.Context, metaMessageID pgtype.Text) (db.WhatsappMessage, error)
	ListApprovedTemplates(ctx context.Context) ([]db.WhatsappTemplate, error)
	MarkUsageSyncedWithStripe(ctx context.Context, arg db.MarkUsageSyncedWithStripeParams) error
	UpdateWhatsAppMessageStatus(ctx context.Context, arg db.UpdateWhatsAppMessageStatusParams) (db.WhatsappMessage, error)
	UpsertCompanyWhatsAppConfig(ctx context.Context, arg db.UpsertCompanyWhatsAppConfigParams) (db.CompanyWhatsappConfig, error)
	UpsertMonthlyUsage(ctx context.Context, arg db.UpsertMonthlyUsageParams) (db.WhatsappUsageMonthly, error)
}

type MetaClientInterface interface {
	SendTemplateMessage(ctx context.Context, phoneNumberID, accessToken string, req SendMessageRequest) (metaMessageID string, err error)
}

type CompaniesServiceInterface interface {
	GetCompanyByID(ctx context.Context, id uuid.UUID) (companiesDomain.CompanyResponse, error)
}

type PlansServiceInterface interface {
	GetPlanByID(ctx context.Context, planId uuid.UUID) (plansDomain.PlanResponse, error)
}

type SubscriptionsServiceInterface interface {
	GetSubscriptionByCompanyID(ctx context.Context, companyID uuid.UUID) (subscriptionsDomain.SubscriptionResponse, error)
}

type BillingClientInterface interface {
	ReportUsage(ctx context.Context, customerID string, quantity int, idempotencyKey string) error
}

type ServiceInterface interface {
	SendMessage(ctx context.Context, companyId uuid.UUID, req SendMessageRequest) (*Message, error)
	GetCompanyConfig(ctx context.Context, companyId uuid.UUID) (*CompanyWhatsAppConfig, error)
	UpsertCompanyConfig(ctx context.Context, companyId uuid.UUID, req UpsertCompanyConfigRequest) (*CompanyWhatsAppConfig, error)
	ListApprovedTemplates(ctx context.Context) ([]Template, error)
	HandleWebhookEvent(ctx context.Context, payload []byte) error
	SyncMonthlyUsage(ctx context.Context, companyId uuid.UUID, periodStart, periodEnd time.Time) error
}

type WhatsAppMode string

const (
	WhatsAppModePlatformShared WhatsAppMode = "platform_shared"
	WhatsAppModeOwnWABA        WhatsAppMode = "own_waba"
)

type MessageCategory string

const (
	CategoryMarketing      MessageCategory = "marketing"
	CategoryUtility        MessageCategory = "utility"
	CategoryAuthentication MessageCategory = "authentication"
	CategoryService        MessageCategory = "service"
)

type MessageStatus string

const (
	StatusQueued    MessageStatus = "queued"
	StatusSent      MessageStatus = "sent"
	StatusDelivered MessageStatus = "delivered"
	StatusRead      MessageStatus = "read"
	StatusFailed    MessageStatus = "failed"
)

// CompanyWhatsAppConfig define como uma empresa específica envia mensagens:
// pelo WABA central da ProTrack (PlatformShared) ou pelo próprio WABA
// conectado via Embedded Signup (OwnWABA).
type CompanyWhatsAppConfig struct {
	ID                      uuid.UUID
	CompanyID               uuid.UUID
	Mode                    WhatsAppMode
	WABAID                  *string
	PhoneNumberID           *string
	DisplayPhoneNumber      *string
	MonthlyMessageAllowance int
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Template representa um item do catálogo fechado de mensagens.
// IsPlatformSharedEligible controla a whitelist do modo platform_shared —
// é a trava central que impede uso livre/conversacional no número compartilhado.
type Template struct {
	ID                       uuid.UUID
	MetaTemplateName         string
	Category                 MessageCategory
	LanguageCode             string
	BodyText                 string
	Variables                []string
	IsPlatformSharedEligible bool
	MetaApprovalStatus       string // pending | approved | rejected | paused
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// Message é o registro de cada disparo, atualizado pelo handler de webhook
// conforme os status chegam (sent -> delivered -> read, ou failed).
type Message struct {
	ID                 uuid.UUID
	CompanyID          uuid.UUID
	TemplateID         *uuid.UUID
	Category           MessageCategory
	RecipientPhone     string
	MetaMessageID      *string
	Status             MessageStatus
	FailureCode        *int32
	FailureReason      *string
	EstimatedCostCents *int32
	SentAt             *time.Time
	DeliveredAt        *time.Time
	CreatedAt          time.Time
}

// SendMessageRequest é o input do serviço de disparo. Se a empresa estiver em
// modo PlatformShared, TemplateName precisa bater com um template com
// IsPlatformSharedEligible = true — essa validação acontece na camada de service,
// não aqui, pra manter o domain livre de I/O.
type SendMessageRequest struct {
	TemplateName   string            `json:"template_name"`
	LanguageCode   string            `json:"language_code"`
	RecipientPhone string            `json:"recipient_phone"`
	Category       MessageCategory   `json:"category"`
	Variables      map[string]string `json:"variables"`
}

type UpsertCompanyConfigRequest struct {
	Mode                    WhatsAppMode
	WABAID                  *string
	PhoneNumberId           *string
	DisplayPhoneNumber      *string
	AccessToken             *string
	MonthlyMessageAllowance int
	IsActive                bool
}
