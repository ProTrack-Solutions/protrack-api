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

func (r *Repository) CreateSubscription(ctx context.Context, arg db.CreateSubscriptionParams) error {
	return r.queries().CreateSubscription(ctx, arg)
}

func (r *Repository) GetSubscriptionById(ctx context.Context, id pgtype.UUID) (db.Subscription, error) {
	return r.queries().GetSubscriptionById(ctx, id)
}

func (r *Repository) UpdateSubscriptionPlan(ctx context.Context, arg db.UpdateSubscriptionPlanParams) error {
	return r.queries().UpdateSubscriptionPlan(ctx, arg)
}

func (r *Repository) UpdateSubscriptionMethod(ctx context.Context, arg db.UpdateSubscriptionMethodParams) error {
	return r.queries().UpdateSubscriptionMethod(ctx, arg)
}

func (r *Repository) UpdateSubscriptionStatus(ctx context.Context, arg db.UpdateSubscriptionStatusParams) error {
	return r.queries().UpdateSubscriptionStatus(ctx, arg)
}

func (r *Repository) CancelSubscription(ctx context.Context, id pgtype.UUID) error {
	return r.queries().CancelSubscription(ctx, id)
}

func (r *Repository) GetSubscriptionByExternalSubscriptionId(ctx context.Context, externalSubscriptionId pgtype.Text) (db.Subscription, error) {
	return r.queries().GetSubscriptionByExternalSubscriptionId(ctx, externalSubscriptionId)
}

func (r *Repository) GetSubscriptionByCompanyID(ctx context.Context, companyID pgtype.UUID) (db.Subscription, error) {
	return r.queries().GetSubscriptionByCompanyID(ctx, companyID)
}

func (r *Repository) ListSubscriptionsDueOn(ctx context.Context, arg db.ListSubscriptionsDueOnParams) ([]db.Subscription, error) {
	return r.queries().ListSubscriptionsDueOn(ctx, arg)
}
