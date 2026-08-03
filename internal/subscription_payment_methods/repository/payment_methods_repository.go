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

func (r *Repository) CreateSubscriptionPaymentMethod(ctx context.Context, arg db.CreateSubscriptionPaymentMethodParams) (pgtype.UUID, error) {
	return r.queries().CreateSubscriptionPaymentMethod(ctx, arg)
}

func (r *Repository) GetSubscriptionPaymentMethodByCompanyId(ctx context.Context, companyId pgtype.UUID) ([]db.SubscriptionPaymentMethod, error) {
	return r.queries().ListSubscriptionPaymentMethodsByCompanyId(ctx, companyId)
}

func (r *Repository) GetSubscriptionPaymentMethodById(ctx context.Context, paymentMethodId pgtype.UUID) (db.SubscriptionPaymentMethod, error) {
	return r.queries().GetSubscriptionPaymentMethodById(ctx, paymentMethodId)
}

func (r *Repository) UpdateSubscriptionPaymentMethod(ctx context.Context, arg db.UpdateSubscriptionPaymentMethodParams) error {
	return r.queries().UpdateSubscriptionPaymentMethod(ctx, arg)
}

func (r *Repository) SetDefaultSubscriptionPaymentMethod(ctx context.Context, arg db.SetDefaultSubscriptionPaymentMethodParams) error {
	return r.queries().SetDefaultSubscriptionPaymentMethod(ctx, arg)
}

func (r *Repository) DeleteSubscriptionPaymentMethod(ctx context.Context, paymentMethodId pgtype.UUID) error {
	return r.queries().DeleteSubscriptionPaymentMethod(ctx, paymentMethodId)
}
