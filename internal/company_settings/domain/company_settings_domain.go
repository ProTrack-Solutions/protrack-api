package domain

import (
	"context"
	"time"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var DefaultSettings = map[enums.CompanySettingsKey]any{
	enums.IsWhatsappActive:    true,
	enums.IsExcessUsage:       false,
	enums.LowStock:            2,
	enums.MediumStock:         3,
	enums.NormalStock:         5,
	enums.SaleOverdueTemplate: "venda_vencida",
}

type RepositoryInterface interface {
	GetCompanySetting(ctx context.Context, arg db.GetCompanySettingParams) (db.CompanySetting, error)
	ListCompanySettings(ctx context.Context, companyId pgtype.UUID) ([]db.CompanySetting, error)
	UpsertCompanySetting(ctx context.Context, arg db.UpsertCompanySettingParams) (db.CompanySetting, error)
	DeleteCompanySetting(ctx context.Context, arg db.DeleteCompanySettingParams) error
	DeleteAllCompanySettings(ctx context.Context, companyId pgtype.UUID) error
}

type ServiceInterface interface {
	GetCompanySetting(ctx context.Context, companyId uuid.UUID, key enums.CompanySettingsKey) (CompanySettingResponse, error)
	ListCompanySettings(ctx context.Context, companyId uuid.UUID) ([]CompanySettingResponse, error)
	UpsertCompanySetting(ctx context.Context, componyId uuid.UUID, req UpsertCompanySettingRequest) (CompanySettingResponse, error)
	DeleteCompanySetting(ctx context.Context, companyId uuid.UUID, key string) error
	DeleteAllCompanySettings(ctx context.Context, companyId uuid.UUID) error
	SetDefaultSettings(ctx context.Context, tx db.DBTX, companyId uuid.UUID) error
	RestorDefaultSettings(ctx context.Context, companyId uuid.UUID) error
}

type CompanySettingResponse struct {
	ID        uuid.UUID                `json:"id"`
	CompanyID uuid.UUID                `json:"company_id"`
	Key       enums.CompanySettingsKey `json:"key"`
	Value     any                      `json:"value"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

type UpsertCompanySettingRequest struct {
	Key   enums.CompanySettingsKey `json:"key"`
	Value any                      `json:"value"`
}
