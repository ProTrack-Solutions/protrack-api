package repository

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (r *Repository) CreateProductsCategories(ctx context.Context, arg db.CreateProductCategoryParams) (db.ProductCategory, error) {
	return r.q.CreateProductCategory(ctx, arg)
}

func (r *Repository) DeleteProductCategory(ctx context.Context, arg db.DeleteProductCategoryParams) error {
	return r.q.DeleteProductCategory(ctx, arg)
}

func (r *Repository) GetProductCategoryById(ctx context.Context, id pgtype.UUID) (db.ProductCategory, error) {
	return r.q.GetProductCategoryById(ctx, id)
}

func (r *Repository) ListProductCategoryByCompanyId(ctx context.Context, companyId pgtype.UUID) ([]db.ProductCategory, error) {
	return r.q.ListProductCategoryByCompanyId(ctx, companyId)
}

func (r *Repository) SetProductCategoryStatus(ctx context.Context, arg db.SetProductCategoryStatusParams) (int64, error) {
	return r.q.SetProductCategoryStatus(ctx, arg)
}

func (r *Repository) UpdateProductCategory(ctx context.Context, arg db.UpdateProductCategoryParams) (db.ProductCategory, error) {
	return r.q.UpdateProductCategory(ctx, arg)
}
