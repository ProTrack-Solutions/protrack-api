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

func (r *Repository) CreateInvoiceHistory(ctx context.Context, arg db.CreateInvoiceHistoryParams) error {
	return r.queries().CreateInvoiceHistory(ctx, arg)
}

func (r *Repository) UpdateInvoiceStatus(ctx context.Context, arg db.UpdateInvoiceStatusParams) error {
	return r.queries().UpdateInvoiceStatus(ctx, arg)
}

func (r *Repository) GetInvoiceById(ctx context.Context, id pgtype.UUID) (db.InvoiceHistory, error) {
	return r.queries().GetInvoiceById(ctx, id)
}

func (r *Repository) GetInvoiceByMpPaymentId(ctx context.Context, mpPaymentId string) (db.InvoiceHistory, error) {
	return r.queries().GetInvoiceByMpPaymentId(ctx, mpPaymentId)
}

func (r *Repository) ListInvoiceByCompany(ctx context.Context, arg db.ListInvoicesByCompanyParams) ([]db.InvoiceHistory, error) {
	return r.queries().ListInvoicesByCompany(ctx, arg)
}

func (r *Repository) CountInvouces(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	return r.queries().CountInvoices(ctx, companyId)
}
