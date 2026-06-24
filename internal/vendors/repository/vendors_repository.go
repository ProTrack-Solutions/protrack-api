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

func (r *Repository) CreateVendors(ctx context.Context, arg db.CreateVendorsParams) error {
	return r.queries().CreateVendors(ctx, arg)
}

func (r *Repository) GetVendorsById(ctx context.Context, arg db.GetVendorsByIdParams) (db.Vendor, error) {
	return r.queries().GetVendorsById(ctx, db.GetVendorsByIdParams{ID: arg.ID, CompanyID: arg.CompanyID})
}

func (r *Repository) ListVendors(ctx context.Context, companyId pgtype.UUID) ([]db.Vendor, error) {
	return r.queries().ListVendors(ctx, companyId)
}

func (r *Repository) ListVendorsIsActive(ctx context.Context, companyId pgtype.UUID) ([]db.Vendor, error) {
	return r.queries().ListVendorsIsActive(ctx, companyId)
}

func (r *Repository) ToggleVendorsActive(ctx context.Context, arg db.ToggleVendorsActiveParams) error {
	return r.queries().ToggleVendorsActive(ctx, db.ToggleVendorsActiveParams{ID: arg.ID, IsActive: arg.IsActive})
}

func (r *Repository) UpdateVendors(ctx context.Context, arg db.UpdateVendorsParams) error {
	return r.queries().UpdateVendors(ctx, arg)
}
