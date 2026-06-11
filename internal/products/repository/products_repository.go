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

func (r *Repository) WithTx(tx db.DBTX) *Repository {
	return &Repository{
		db: tx,
	}
}

func (r *Repository) queries() *db.Queries {
	return db.New(r.db)
}

func (r *Repository) CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.Product, error) {
	return r.queries().CreateProduct(ctx, arg)
}

func (r *Repository) DeleteProduct(ctx context.Context, arg db.DeleteProductParams) error {
	return r.queries().DeleteProduct(ctx, arg)
}

func (r *Repository) GetProductByBarcode(ctx context.Context, barcode pgtype.Text) (db.Product, error) {
	return r.queries().GetProductByBarcode(ctx, barcode)
}

func (r *Repository) GetProductById(ctx context.Context, id pgtype.UUID) (db.Product, error) {
	return r.queries().GetProductById(ctx, id)
}

func (r *Repository) ListProductsByCategoryId(ctx context.Context, arg db.ListProductsByCategoryIdParams) ([]db.Product, error) {
	return r.queries().ListProductsByCategoryId(ctx, arg)
}

func (r *Repository) ListProductsByCompany(ctx context.Context, categoryID pgtype.UUID) ([]db.ListProductsByCompanyRow, error) {
	return r.queries().ListProductsByCompany(ctx, categoryID)
}

func (r *Repository) UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (db.Product, error) {
	return r.queries().UpdateProduct(ctx, arg)
}

func (r *Repository) DecrementStock(ctx context.Context, arg db.DecrementStockParams) error {
	return r.queries().DecrementStock(ctx, arg)
}

func (r *Repository) CountProducts(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	return r.queries().CountProducts(ctx, companyId)
}

func (r *Repository) GetProductsPerformanceSummary(ctx context.Context, companyId pgtype.UUID) (db.GetProductsPerformanceSummaryRow, error) {
	return r.queries().GetProductsPerformanceSummary(ctx, companyId)
}

func (r *Repository) GetCostTotalStock(ctx context.Context, companyId pgtype.UUID) (float64, error) {
	return r.queries().GetCostTotalStock(ctx, companyId)
}

func (r *Repository) GetTop5BestSellingProducts(ctx context.Context, companyId pgtype.UUID) ([]db.GetTop5BestSellingProductsRow, error) {
	return r.queries().GetTop5BestSellingProducts(ctx, companyId)
}

func (r *Repository) GetInventoryReport(ctx context.Context, arg db.GetInventoryReportParams) ([]db.GetInventoryReportRow, error) {
	return r.queries().GetInventoryReport(ctx, arg)
}

func (r *Repository) ListProductsByDate(ctx context.Context, arg db.ListProductsByDateParams) ([]db.ListProductsByDateRow, error) {
	return r.queries().ListProductsByDate(ctx, arg)
}

func (r *Repository) ListProductBuCategoryIdAndDate(ctx context.Context, arg db.ListProductsByCategoryAndDateParams) ([]db.ListProductsByCategoryAndDateRow, error) {
	return r.queries().ListProductsByCategoryAndDate(ctx, arg)
}
