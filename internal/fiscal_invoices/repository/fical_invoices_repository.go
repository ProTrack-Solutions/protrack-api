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

func (r *Repository) CreateFiscalInvoice(ctx context.Context, arg db.CreateFiscalInvoiceParams) (db.FiscalInvoice, error) {
	return r.queries().CreateFiscalInvoice(ctx, arg)
}

func (r *Repository) GetFiscalInvoiceByID(ctx context.Context, arg db.GetFiscalInvoiceByIDParams) (db.FiscalInvoice, error) {
	return r.queries().GetFiscalInvoiceByID(ctx, arg)
}

func (r *Repository) GetFiscalInvoiceBySaleAndType(ctx context.Context, arg db.GetFiscalInvoiceBySaleAndTypeParams) (db.FiscalInvoice, error) {
	return r.queries().GetFiscalInvoiceBySaleAndType(ctx, arg)
}

func (r *Repository) GetFiscalInvoiceByProviderInvoiceID(ctx context.Context, providerInvoiceID pgtype.Text) (db.FiscalInvoice, error) {
	return r.queries().GetFiscalInvoiceByProviderInvoiceID(ctx, providerInvoiceID)
}

func (r *Repository) ListFiscalInvoicesBySale(ctx context.Context, arg db.ListFiscalInvoicesBySaleParams) ([]db.FiscalInvoice, error) {
	return r.queries().ListFiscalInvoicesBySale(ctx, arg)
}

func (r *Repository) ListStaleProcessingFiscalInvoices(ctx context.Context, updatedBefore pgtype.Timestamptz) ([]db.FiscalInvoice, error) {
	return r.queries().ListStaleProcessingFiscalInvoices(ctx, updatedBefore)
}

func (r *Repository) ResetFiscalInvoiceForRetry(ctx context.Context, id pgtype.UUID) (db.FiscalInvoice, error) {
	return r.queries().ResetFiscalInvoiceForRetry(ctx, id)
}

func (r *Repository) UpdateFiscalInvoiceProcessing(ctx context.Context, arg db.UpdateFiscalInvoiceProcessingParams) (db.FiscalInvoice, error) {
	return r.queries().UpdateFiscalInvoiceProcessing(ctx, arg)
}

func (r *Repository) UpdateFiscalInvoiceAuthorized(ctx context.Context, arg db.UpdateFiscalInvoiceAuthorizedParams) (db.FiscalInvoice, error) {
	return r.queries().UpdateFiscalInvoiceAuthorized(ctx, arg)
}

func (r *Repository) UpdateFiscalInvoiceRejected(ctx context.Context, arg db.UpdateFiscalInvoiceRejectedParams) (db.FiscalInvoice, error) {
	return r.queries().UpdateFiscalInvoiceRejected(ctx, arg)
}

func (r *Repository) UpdateFiscalInvoiceCancelled(ctx context.Context, arg db.UpdateFiscalInvoiceCancelledParams) (db.FiscalInvoice, error) {
	return r.queries().UpdateFiscalInvoiceCancelled(ctx, arg)
}

func (r *Repository) UpdateFiscalInvoiceCancelProcessing(ctx context.Context, id pgtype.UUID) (db.FiscalInvoice, error) {
	return r.queries().UpdateFiscalInvoiceCancelProcessing(ctx, id)
}

func (r *Repository) CreateFiscalInvoiceEvent(ctx context.Context, arg db.CreateFiscalInvoiceEventParams) (db.FiscalInvoiceEvent, error) {
	return r.queries().CreateFiscalInvoiceEvent(ctx, arg)
}

func (r *Repository) ListFiscalInvoiceEventsByInvoiceID(ctx context.Context, fiscalInvoiceID pgtype.UUID) ([]db.FiscalInvoiceEvent, error) {
	return r.queries().ListFiscalInvoiceEventsByInvoiceID(ctx, fiscalInvoiceID)
}
