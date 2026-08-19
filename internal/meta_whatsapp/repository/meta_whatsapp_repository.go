package repository

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	db db.DBTX
}

func NewRepository(db db.DBTX) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) queries() *db.Queries {
	return db.New(r.db)
}

func (r *Repository) WithTx(tx db.DBTX) *Repository {
	return &Repository{
		db: tx,
	}
}

func (r *Repository) CountMessagesInPeriod(ctx context.Context, arg db.CountMessagesInPeriodParams) (int64, error) {
	return r.queries().CountMessagesInPeriod(ctx, arg)
}

func (r *Repository) CreateWhatsAppMessage(ctx context.Context, arg db.CreateWhatsAppMessageParams) (db.WhatsappMessage, error) {
	return r.queries().CreateWhatsAppMessage(ctx, arg)
}

func (r *Repository) GetCompanyWhatsAppConfig(ctx context.Context, companyId pgtype.UUID) (db.CompanyWhatsappConfig, error) {
	return r.queries().GetCompanyWhatsAppConfig(ctx, companyId)
}

func (r *Repository) GetEligibleTemplateByName(ctx context.Context, arg db.GetEligibleTemplateByNameParams) (db.WhatsappTemplate, error) {
	return r.queries().GetEligibleTemplateByName(ctx, arg)
}

func (r *Repository) GetMessageByMetaID(ctx context.Context, metaMessageID pgtype.Text) (db.WhatsappMessage, error) {
	return r.queries().GetMessageByMetaID(ctx, metaMessageID)
}

func (r *Repository) ListApprovedTemplates(ctx context.Context) ([]db.WhatsappTemplate, error) {
	return r.queries().ListApprovedTemplates(ctx)
}

func (r *Repository) MarkUsageSyncedWithStripe(ctx context.Context, arg db.MarkUsageSyncedWithStripeParams) error {
	return r.queries().MarkUsageSyncedWithStripe(ctx, arg)
}

func (r *Repository) UpdateWhatsAppMessageStatus(ctx context.Context, arg db.UpdateWhatsAppMessageStatusParams) (db.WhatsappMessage, error) {
	return r.queries().UpdateWhatsAppMessageStatus(ctx, arg)
}

func (r *Repository) UpsertCompanyWhatsAppConfig(ctx context.Context, arg db.UpsertCompanyWhatsAppConfigParams) (db.CompanyWhatsappConfig, error) {
	return r.queries().UpsertCompanyWhatsAppConfig(ctx, arg)
}

func (r *Repository) UpsertMonthlyUsage(ctx context.Context, arg db.UpsertMonthlyUsageParams) (db.WhatsappUsageMonthly, error) {
	return r.queries().UpsertMonthlyUsage(ctx, arg)
}
