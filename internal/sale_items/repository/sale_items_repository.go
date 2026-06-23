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

func (r *Repository) CreateSaleItem(ctx context.Context, arg db.CreateSaleItemParams) error {
	q := db.New(r.db)
	return q.CreateSaleItem(ctx, arg)
}

func (r *Repository) DeleteItemsBySale(ctx context.Context, saleId pgtype.UUID) error {
	q := db.New(r.db)
	return q.DeleteItemsBySale(ctx, saleId)
}

func (r *Repository) DeleteSaleItem(ctx context.Context, id pgtype.UUID) error {
	q := db.New(r.db)
	return q.DeleteSaleItem(ctx, id)
}

func (r *Repository) ListItemsFromPendingSale(ctx context.Context, saleID pgtype.UUID) ([]db.ListItemsFromPendingSaleRow, error) {
	q := db.New(r.db)
	return q.ListItemsFromPendingSale(ctx, saleID)
}

func (r *Repository) ListItemsByCompany(ctx context.Context, companyId pgtype.UUID) ([]db.ListItemsByCompanyRow, error) {
	q := db.New(r.db)
	return q.ListItemsByCompany(ctx, companyId)
}

func (r *Repository) ListItemsByDate(ctx context.Context, arg db.ListItemsByDateParams) ([]db.ListItemsByDateRow, error) {
	q := db.New(r.db)
	return q.ListItemsByDate(ctx, arg)
}
