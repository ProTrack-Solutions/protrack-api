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

func (r *Repository) CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.Product, error) {
	product, err := r.queries().CreateProduct(ctx, arg)
	return db.Product(product), err
}

func (r *Repository) DeleteProduct(ctx context.Context, arg db.DeleteProductParams) (int64, error) {
	return r.queries().DeleteProduct(ctx, arg)
}

func (r *Repository) GetProductByBarcode(ctx context.Context, arg db.GetProductByBarcodeParams) (db.Product, error) {
	return r.queries().GetProductByBarcode(ctx, arg)
}

func (r *Repository) GetProductById(ctx context.Context, id pgtype.UUID) (db.Product, error) {
	return r.queries().GetProductById(ctx, id)
}

func (r *Repository) ListProductsByCategoryId(ctx context.Context, arg db.ListProductsByCategoryIdParams) ([]db.Product, error) {
	return r.queries().ListProductsByCategoryId(ctx, arg)
}

func (r *Repository) ListProductsByCompany(ctx context.Context, companyId pgtype.UUID) ([]db.ListProductsByCompanyRow, error) {
	return r.queries().ListProductsByCompany(ctx, companyId)
}

func (r *Repository) UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (db.Product, error) {
	return r.queries().UpdateProduct(ctx, arg)
}

func (r *Repository) AdjustStock(ctx context.Context, arg db.AdjustStockParams) (db.AdjustStockRow, error) {
	return r.queries().AdjustStock(ctx, arg)
}

func (r *Repository) DecrementStock(ctx context.Context, arg db.DecrementStockParams) error {
	return r.queries().DecrementStock(ctx, arg)
}

func (r *Repository) CreateStockMovement(ctx context.Context, arg db.CreateStockMovementParams) error {
	return r.queries().CreateStockMovement(ctx, arg)
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

func (r *Repository) CountProductsByCompany(ctx context.Context, companyID pgtype.UUID) (int64, error) {
	return r.queries().CountProductsByCompany(ctx, companyID)
}

func (r *Repository) ListProductsByCompanyPaginated(ctx context.Context, arg db.ListProductsByCompanyPaginatedParams) ([]db.ListProductsByCompanyPaginatedRow, error) {
	return r.queries().ListProductsByCompanyPaginated(ctx, arg)
}

func (r *Repository) CountLowStockProductsByCompany(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	return r.queries().CountLowStockProductsByCompany(ctx, companyId)
}

func (r *Repository) GetGeneralTotalStockValue(ctx context.Context, companyId pgtype.UUID) (float64, error) {
	return r.queries().GetGeneralTotalStockValue(ctx, companyId)
}

func (r *Repository) GetGlobalTotalStockQuantity(ctx context.Context, companyId pgtype.UUID) (int32, error) {
	return r.queries().GetGlobalTotalStockQuantity(ctx, companyId)
}

// Grade de produtos

func (r *Repository) ListStockEntries(ctx context.Context, arg db.ListStockEntriesParams) ([]db.ListStockEntriesRow, error) {
	return r.queries().ListStockEntries(ctx, arg)
}

func (r *Repository) ListGridVariations(ctx context.Context, arg db.ListGridVariationsParams) ([]db.ListGridVariationsRow, error) {
	return r.queries().ListGridVariations(ctx, arg)
}

func (r *Repository) LockGridProducts(ctx context.Context, arg db.LockGridProductsParams) ([]db.LockGridProductsRow, error) {
	return r.queries().LockGridProducts(ctx, arg)
}

func (r *Repository) CreateGridProduct(ctx context.Context, arg db.CreateGridProductParams) (pgtype.UUID, error) {
	return r.queries().CreateGridProduct(ctx, arg)
}

func (r *Repository) UpdateGridVariation(ctx context.Context, arg db.UpdateGridVariationParams) error {
	return r.queries().UpdateGridVariation(ctx, arg)
}

func (r *Repository) AssignProductToModel(ctx context.Context, arg db.AssignProductToModelParams) error {
	return r.queries().AssignProductToModel(ctx, arg)
}

func (r *Repository) RemoveGridVariation(ctx context.Context, arg db.RemoveGridVariationParams) (int64, error) {
	return r.queries().RemoveGridVariation(ctx, arg)
}

func (r *Repository) GridCategoryExists(ctx context.Context, arg db.GridCategoryExistsParams) (bool, error) {
	return r.queries().GridCategoryExists(ctx, arg)
}

func (r *Repository) GridBarcodeExists(ctx context.Context, arg db.GridBarcodeExistsParams) (bool, error) {
	return r.queries().GridBarcodeExists(ctx, arg)
}

func (r *Repository) CountActiveProducts(ctx context.Context, companyID pgtype.UUID) (int32, error) {
	return r.queries().CountActiveProducts(ctx, companyID)
}
