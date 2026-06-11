package repository

import (
	"context"

	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	db db.DBTX
}

func NewRepository(tx db.DBTX) *Repository {
	return &Repository{
		db: tx,
	}
}

func (r *Repository) queries() *db.Queries {
	return db.New(r.db)
}

func (r *Repository) CreatePaymentMethod(ctx context.Context, arg db.CreatePaymentMethodParams) error {
	return r.queries().CreatePaymentMethod(ctx, arg)
}

func (r *Repository) GetPaymentMethodById(ctx context.Context, id pgtype.UUID) (db.PaymentMethod, error) {
	return r.queries().GetPaymentMethodByID(ctx, id)
}

func (r *Repository) ListPaymentMethod(ctx context.Context, companyId pgtype.UUID) ([]db.PaymentMethod, error) {
	return r.queries().ListPaymentMethod(ctx, companyId)
}

func (r *Repository) ListPaymentMethodIsActive(ctx context.Context, companyId pgtype.UUID) ([]db.PaymentMethod, error) {
	return r.queries().ListPaymentMethodIsActive(ctx, companyId)
}

func (r *Repository) TogglePaymentMethodActive(ctx context.Context, arg db.TogglePaymentMethodActiveParams) error {
	return r.queries().TogglePaymentMethodActive(ctx, arg)
}

func (r *Repository) GetPaymentMethodsStats(ctx context.Context, companyId pgtype.UUID) ([]db.GetPaymentMethodsStatsRow, error) {
	return r.queries().GetPaymentMethodsStats(ctx, companyId)
}
