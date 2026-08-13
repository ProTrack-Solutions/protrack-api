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

func (r *Repository) WithTx(tx db.DBTX) *Repository {
	return &Repository{
		db: tx,
	}
}

func (r *Repository) queries() *db.Queries {
	return db.New(r.db)
}

// ===== whatsapp_bot_configs =====

func (r *Repository) CreateWhatsAppBotConfig(ctx context.Context, arg db.CreateWhatsAppBotConfigParams) (db.WhatsappBotConfig, error) {
	return r.queries().CreateWhatsAppBotConfig(ctx, arg)
}

func (r *Repository) GetWhatsAppBotConfigByCompanyID(ctx context.Context, companyID pgtype.UUID) (db.WhatsappBotConfig, error) {
	return r.queries().GetWhatsAppBotConfigByCompanyID(ctx, companyID)
}

func (r *Repository) GetWhatsAppBotConfigByInstanceName(ctx context.Context, instanceName string) (db.WhatsappBotConfig, error) {
	return r.queries().GetWhatsAppBotConfigByInstanceName(ctx, instanceName)
}

func (r *Repository) UpdateWhatsAppBotConfig(ctx context.Context, arg db.UpdateWhatsAppBotConfigParams) (db.WhatsappBotConfig, error) {
	return r.queries().UpdateWhatsAppBotConfig(ctx, arg)
}

func (r *Repository) UpdateWhatsAppBotInstanceName(ctx context.Context, arg db.UpdateWhatsAppBotInstanceNameParams) (db.WhatsappBotConfig, error) {
	return r.queries().UpdateWhatsAppBotInstanceName(ctx, arg)
}

func (r *Repository) SetWhatsAppBotActive(ctx context.Context, arg db.SetWhatsAppBotActiveParams) error {
	return r.queries().SetWhatsAppBotActive(ctx, arg)
}

func (r *Repository) DeleteWhatsAppBotConfig(ctx context.Context, id pgtype.UUID) error {
	return r.queries().DeleteWhatsAppBotConfig(ctx, id)
}

// ===== whatsapp_bot_menu_options =====

func (r *Repository) CreateWhatsAppBotMenuOption(ctx context.Context, arg db.CreateWhatsAppBotMenuOptionParams) (db.WhatsappBotMenuOption, error) {
	return r.queries().CreateWhatsAppBotMenuOption(ctx, arg)
}

func (r *Repository) ListWhatsAppBotMenuOptions(ctx context.Context, botConfigID pgtype.UUID) ([]db.WhatsappBotMenuOption, error) {
	return r.queries().ListWhatsAppBotMenuOptions(ctx, botConfigID)
}

func (r *Repository) GetWhatsAppBotMenuOptionByKey(ctx context.Context, arg db.GetWhatsAppBotMenuOptionByKeyParams) (db.WhatsappBotMenuOption, error) {
	return r.queries().GetWhatsAppBotMenuOptionByKey(ctx, arg)
}

func (r *Repository) UpdateWhatsAppBotMenuOption(ctx context.Context, arg db.UpdateWhatsAppBotMenuOptionParams) (db.WhatsappBotMenuOption, error) {
	return r.queries().UpdateWhatsAppBotMenuOption(ctx, arg)
}

func (r *Repository) DeleteWhatsAppBotMenuOption(ctx context.Context, id pgtype.UUID) error {
	return r.queries().DeleteWhatsAppBotMenuOption(ctx, id)
}

func (r *Repository) DeleteAllMenuOptionsForBotConfig(ctx context.Context, botConfigID pgtype.UUID) error {
	return r.queries().DeleteAllMenuOptionsForBotConfig(ctx, botConfigID)
}

// ===== consulta composta (config + opções de uma vez) =====

func (r *Repository) GetWhatsAppBotConfigWithOptionsByInstance(ctx context.Context, instanceName string) ([]db.GetWhatsAppBotConfigWithOptionsByInstanceRow, error) {
	return r.queries().GetWhatsAppBotConfigWithOptionsByInstance(ctx, instanceName)
}
