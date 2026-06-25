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

func (r *Repository) CreateSales(ctx context.Context, arg db.CreateSaleParams) (pgtype.UUID, error) {
	q := db.New(r.db)
	return q.CreateSale(ctx, arg)
}

func (r *Repository) DeleteSales(ctx context.Context, arg db.DeleteSaleParams) error {
	q := db.New(r.db)
	return q.DeleteSale(ctx, arg)
}

func (r *Repository) GetSaleById(ctx context.Context, arg db.GetSaleByIdParams) (db.GetSaleByIdRow, error) {
	q := db.New(r.db)
	return q.GetSaleById(ctx, arg)
}

func (r *Repository) ListSales(ctx context.Context, companyId pgtype.UUID) ([]db.ListSalesRow, error) {
	q := db.New(r.db)
	return q.ListSales(ctx, companyId)
}

func (r *Repository) UpdateSaleStatus(ctx context.Context, arg db.UpdateSaleStatusParams) error {
	q := db.New(r.db)
	return q.UpdateSaleStatus(ctx, arg)
}

func (r *Repository) ListSalesByCompanyAndStatus(ctx context.Context, arg db.ListSalesByCompanyAndStatusParams) ([]db.ListSalesByCompanyAndStatusRow, error) {
	q := db.New(r.db)
	return q.ListSalesByCompanyAndStatus(ctx, arg)
}

func (r *Repository) CountSales(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	q := db.New(r.db)
	return q.CountSales(ctx, companyId)
}

func (r *Repository) GetSalesPerformanceSummary(ctx context.Context, companyId pgtype.UUID) (db.GetSalesPerformanceSummaryRow, error) {
	q := db.New(r.db)
	return q.GetSalesPerformanceSummary(ctx, companyId)
}

func (r *Repository) GetTotalAmountSummary(ctx context.Context, companyId pgtype.UUID) (db.GetTotalAmountSummaryRow, error) {
	q := db.New(r.db)
	return q.GetTotalAmountSummary(ctx, companyId)
}

func (r *Repository) GetTotalAmountByStatus(ctx context.Context, arg db.GetTotalAmountByStatusParams) (float64, error) {
	q := db.New(r.db)
	return q.GetTotalAmountByStatus(ctx, arg)
}

func (r *Repository) GetSaleByIdWhatsapp(ctx context.Context, id pgtype.UUID) (db.GetSaleByIdWhatsappRow, error) {
	q := db.New(r.db)
	return q.GetSaleByIdWhatsapp(ctx, id)
}

func (r *Repository) UpdateOverdueSalesAndAccounts(ctx context.Context) ([]db.UpdateOverdueSalesAndAccountsGlobalRow, error) {
	q := db.New(r.db)
	return q.UpdateOverdueSalesAndAccountsGlobal(ctx)
}

func (r *Repository) GetSaleByIdJust(ctx context.Context, saleId pgtype.UUID) (db.GetSaleByIdJustRow, error) {
	q := db.New(r.db)
	return q.GetSaleByIdJust(ctx, saleId)
}

func (r *Repository) ContSalesPendingAndOverdue(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	return r.queries().ContSalesPendingAndOverdue(ctx, companyId)
}

func (r *Repository) ListSalesWithDetails(ctx context.Context, companyID pgtype.UUID) ([]db.ListSalesWithDetailsRow, error) {
	return r.queries().ListSalesWithDetails(ctx, companyID)
}

func (r *Repository) ListSalesWithDetailsPendingOverdue(ctx context.Context, companyID pgtype.UUID) ([]db.ListSalesWithDetailsPendingOverdueRow, error) {
	return r.queries().ListSalesWithDetailsPendingOverdue(ctx, companyID)
}

func (r *Repository) GetPendingSalesDetailedReport(ctx context.Context, arg db.GetPendingSalesDetailedReportParams) ([]db.GetPendingSalesDetailedReportRow, error) {
	return r.queries().GetPendingSalesDetailedReport(ctx, arg)
}

func (r *Repository) ListSalesWithDetailsPaginate(ctx context.Context, arg db.ListSalesWithDetailsPaginateParams) ([]db.ListSalesWithDetailsPaginateRow, error) {
	return r.queries().ListSalesWithDetailsPaginate(ctx, arg)
}

func (r *Repository) CountSalesByCompany(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	return r.queries().CountSalesByCompany(ctx, companyId)
}
