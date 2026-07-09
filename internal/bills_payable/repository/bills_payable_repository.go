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

func (r *Repository) CreateBillsPayable(ctx context.Context, arg db.CreateBillPayableParams) error {
	return r.queries().CreateBillPayable(ctx, arg)
}

func (r *Repository) GetBillsByStatus(ctx context.Context, arg db.GetBillsByStatusParams) ([]db.BillsPayable, error) {
	return r.queries().GetBillsByStatus(ctx, arg)
}

func (r *Repository) GetOverdueBills(ctx context.Context, companyId pgtype.UUID) ([]db.BillsPayable, error) {
	return r.queries().GetOverdueBills(ctx, companyId)
}

func (r *Repository) ListBillsPayable(ctx context.Context, companyId pgtype.UUID) ([]db.ListBillsPayableRow, error) {
	return r.queries().ListBillsPayable(ctx, companyId)
}

func (r *Repository) PayBill(ctx context.Context, arg db.PayBillParams) error {
	return r.queries().PayBill(ctx, arg)
}

func (r *Repository) UpdateBillPayable(ctx context.Context, arg db.UpdateBillPayableParams) error {
	return r.queries().UpdateBillPayable(ctx, arg)
}

func (r *Repository) GetBillsById(ctx context.Context, arg db.GetBillsByIdParams) (db.BillsPayable, error) {
	return r.queries().GetBillsById(ctx, arg)
}

func (r *Repository) ScheduleBill(ctx context.Context, arg db.ScheduleBillParams) error {
	return r.queries().ScheduleBill(ctx, arg)
}

func (r *Repository) GetBillsPayableSummary(ctx context.Context, companyId pgtype.UUID) (db.GetBillsPayableSummaryRow, error) {
	return r.queries().GetBillsPayableSummary(ctx, companyId)
}

func (r *Repository) UpdateOverdueBillsPayable(ctx context.Context) error {
	return r.queries().UpdateOverdueBillsPayable(ctx)
}

func (r *Repository) CountBillsPayableByCompany(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	return r.queries().CountBillsPayableByCompany(ctx, companyId)
}
