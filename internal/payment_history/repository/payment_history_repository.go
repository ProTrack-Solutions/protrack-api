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

func (r *Repository) CreatePaymentHistory(ctx context.Context, arg db.CreatePaymentHistoryParams) error {
	return r.queries().CreatePaymentHistory(ctx, arg)
}

func (r *Repository) GetPaymentsByCustomer(ctx context.Context, arg db.GetPaymentsByCustomerParams) ([]db.PaymentHistory, error) {
	return r.queries().GetPaymentsByCustomer(ctx, arg)
}

func (r *Repository) GetPaymentsBySale(ctx context.Context, arg db.GetPaymentsBySaleParams) ([]db.PaymentHistory, error) {
	return r.queries().GetPaymentsBySale(ctx, arg)
}

func (r *Repository) GetTotalReceivedByPeriod(ctx context.Context, arg db.GetTotalReceivedByPeriodParams) (pgtype.Numeric, error) {
	return r.queries().GetTotalReceivedByPeriod(ctx, arg)
}

func (r *Repository) ListPaymentHistory(ctx context.Context, companyId pgtype.UUID) ([]db.ListPaymentHistoryRow, error) {
	return r.queries().ListPaymentHistory(ctx, companyId)
}

func (r *Repository) GetPaymentsHistoryReport(ctx context.Context, arg db.GetPaymentsHistoryReportParams) ([]db.GetPaymentsHistoryReportRow, error) {
	return r.queries().GetPaymentsHistoryReport(ctx, arg)
}
