package repository

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	db db.DBTX
}

func NewRepository(dbtx db.DBTX) *Repository {
	return &Repository{
		db: dbtx,
	}
}

func (r *Repository) CreateCompany(ctx context.Context, arg db.CreateCompanyParams) (db.Company, error) {
	q := db.New(r.db)
	return q.CreateCompany(ctx, arg)
}

func (r *Repository) DeleteCompany(ctx context.Context, arg db.DeleteCompanyParams) error {
	q := db.New(r.db)
	return q.DeleteCompany(ctx, arg)
}

func (r *Repository) GetCompanyByDocument(ctx context.Context, document pgtype.Text) (db.Company, error) {
	q := db.New(r.db)
	return q.GetCompanyByDocument(ctx, document)
}

func (r *Repository) GetCompanyByID(ctx context.Context, id pgtype.UUID) (db.Company, error) {
	q := db.New(r.db)
	return q.GetCompanyByID(ctx, id)
}

func (r *Repository) ListCompanies(ctx context.Context) ([]db.Company, error) {
	q := db.New(r.db)
	return q.ListCompanies(ctx)
}

func (r *Repository) SetCompanyStatus(ctx context.Context, arg db.SetCompanyStatusParams) (int64, error) {
	q := db.New(r.db)
	return q.SetCompanyStatus(ctx, arg)
}

func (r *Repository) UpdateCompany(ctx context.Context, arg db.UpdateCompanyParams) (db.Company, error) {
	q := db.New(r.db)
	return q.UpdateCompany(ctx, arg)
}
