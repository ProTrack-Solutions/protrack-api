package repository

import (
	"context"

	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
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

func (r *Repository) CreateBillCategories(ctx context.Context, arg db.CreateBillCategoriesParams) error {
	return r.queries().CreateBillCategories(ctx, arg)
}

func (r *Repository) DeleteBillCategories(ctx context.Context, id pgtype.UUID) error {
	return r.queries().DeleteBillCategories(ctx, id)
}

func (r *Repository) GetBillCategoriesById(ctx context.Context, id pgtype.UUID) (db.BillCategory, error) {
	return r.queries().GetBillCategoriesById(ctx, id)
}

func (r *Repository) ListBillCategories(ctx context.Context, companyId pgtype.UUID) ([]db.BillCategory, error) {
	return r.queries().ListBillCategories(ctx, companyId)
}

func (r *Repository) ListBillCategoriesActive(ctx context.Context, companyId pgtype.UUID) ([]db.BillCategory, error) {
	return r.queries().ListBillCategoriesActive(ctx, companyId)
}

func (r *Repository) ToggleBillCategoriesActive(ctx context.Context, arg db.ToggleBillCategoriesActiveParams) error {
	return r.queries().ToggleBillCategoriesActive(ctx, arg)
}
