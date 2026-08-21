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

func (r *Repository) GetCompanySetting(ctx context.Context, arg db.GetCompanySettingParams) (db.CompanySetting, error) {
	return r.queries().GetCompanySetting(ctx, arg)
}

func (r *Repository) ListCompanySettings(ctx context.Context, companyId pgtype.UUID) ([]db.CompanySetting, error) {
	return r.queries().ListCompanySettings(ctx, companyId)
}

func (r *Repository) UpsertCompanySetting(ctx context.Context, arg db.UpsertCompanySettingParams) (db.CompanySetting, error) {
	return r.queries().UpsertCompanySetting(ctx, arg)
}

func (r *Repository) DeleteCompanySetting(ctx context.Context, arg db.DeleteCompanySettingParams) error {
	return r.queries().DeleteCompanySetting(ctx, arg)
}

func (r *Repository) DeleteAllCompanySettings(ctx context.Context, companyId pgtype.UUID) error {
	return r.queries().DeleteAllCompanySettings(ctx, companyId)
}
