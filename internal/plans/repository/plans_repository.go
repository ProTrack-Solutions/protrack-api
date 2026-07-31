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

func (r *Repository) queries() *db.Queries {
	return db.New(r.db)
}

func (r *Repository) CreatePlans(ctx context.Context, arg db.CreatePlanParams) (pgtype.UUID, error) {
	return r.queries().CreatePlan(ctx, arg)
}

func (r *Repository) GetPlanByID(ctx context.Context, planId pgtype.UUID) (db.Plan, error) {
	return r.queries().GetPlanByID(ctx, planId)
}

func (r *Repository) ListPlans(ctx context.Context) ([]db.Plan, error) {
	return r.queries().ListPlans(ctx)
}

func (r *Repository) ListPlansByActiveStatus(ctx context.Context, active pgtype.Bool) ([]db.Plan, error) {
	return r.queries().ListPlansByActiveStatus(ctx, active)
}

func (r *Repository) UpdatePlan(ctx context.Context, arg db.UpdatePlanParams) error {
	return r.queries().UpdatePlan(ctx, arg)
}

func (r *Repository) TogglePlanActiveStatus(ctx context.Context, arg db.TogglePlanActiveStatusParams) error {
	return r.queries().TogglePlanActiveStatus(ctx, arg)
}
