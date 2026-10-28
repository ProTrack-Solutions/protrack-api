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

func (r *Repository) ReportCompanyHeader(ctx context.Context, companyId pgtype.UUID) (db.ReportCompanyHeaderRow, error) {
	return r.queries().ReportCompanyHeader(ctx, companyId)
}

func (r *Repository) ReportSales(ctx context.Context, arg db.ReportSalesParams) ([]db.ReportSalesRow, error) {
	return r.queries().ReportSales(ctx, arg)
}

func (r *Repository) ReportSalesByDay(ctx context.Context, arg db.ReportSalesByDayParams) ([]db.ReportSalesByDayRow, error) {
	return r.queries().ReportSalesByDay(ctx, arg)
}

func (r *Repository) ReportSalesByPaymentMethod(ctx context.Context, arg db.ReportSalesByPaymentMethodParams) ([]db.ReportSalesByPaymentMethodRow, error) {
	return r.queries().ReportSalesByPaymentMethod(ctx, arg)
}

func (r *Repository) ReportSalesBySeller(ctx context.Context, arg db.ReportSalesBySellerParams) ([]db.ReportSalesBySellerRow, error) {
	return r.queries().ReportSalesBySeller(ctx, arg)
}

func (r *Repository) ReportSalesByCustomer(ctx context.Context, arg db.ReportSalesByCustomerParams) ([]db.ReportSalesByCustomerRow, error) {
	return r.queries().ReportSalesByCustomer(ctx, arg)
}

func (r *Repository) ReportSalesByProduct(ctx context.Context, arg db.ReportSalesByProductParams) ([]db.ReportSalesByProductRow, error) {
	return r.queries().ReportSalesByProduct(ctx, arg)
}

func (r *Repository) ReportSalesByCategory(ctx context.Context, arg db.ReportSalesByCategoryParams) ([]db.ReportSalesByCategoryRow, error) {
	return r.queries().ReportSalesByCategory(ctx, arg)
}

func (r *Repository) ReportInventory(ctx context.Context, companyId pgtype.UUID) ([]db.ReportInventoryRow, error) {
	return r.queries().ReportInventory(ctx, companyId)
}

func (r *Repository) ReportStockMovements(ctx context.Context, arg db.ReportStockMovementsParams) ([]db.ReportStockMovementsRow, error) {
	return r.queries().ReportStockMovements(ctx, arg)
}

func (r *Repository) ReportPaymentsReceived(ctx context.Context, arg db.ReportPaymentsReceivedParams) ([]db.ReportPaymentsReceivedRow, error) {
	return r.queries().ReportPaymentsReceived(ctx, arg)
}

func (r *Repository) ReportOpenInstallments(ctx context.Context, arg db.ReportOpenInstallmentsParams) ([]db.ReportOpenInstallmentsRow, error) {
	return r.queries().ReportOpenInstallments(ctx, arg)
}

func (r *Repository) ReportBillsPayable(ctx context.Context, arg db.ReportBillsPayableParams) ([]db.ReportBillsPayableRow, error) {
	return r.queries().ReportBillsPayable(ctx, arg)
}

func (r *Repository) ReportCashSessions(ctx context.Context, arg db.ReportCashSessionsParams) ([]db.ReportCashSessionsRow, error) {
	return r.queries().ReportCashSessions(ctx, arg)
}

func (r *Repository) ReportDreRevenue(ctx context.Context, arg db.ReportDreRevenueParams) (db.ReportDreRevenueRow, error) {
	return r.queries().ReportDreRevenue(ctx, arg)
}

func (r *Repository) ReportDreExpenses(ctx context.Context, arg db.ReportDreExpensesParams) ([]db.ReportDreExpensesRow, error) {
	return r.queries().ReportDreExpenses(ctx, arg)
}
