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

func (r *Repository) CreatePlanFeatures(ctx context.Context, arg db.CreatePlanFeatureParams) error {
	return r.queries().CreatePlanFeature(ctx, arg)
}

func (r *Repository) ListFeaturesByPlanID(ctx context.Context, planId pgtype.UUID) ([]db.PlanFeature, error) {
	return r.queries().ListFeaturesByPlanID(ctx, planId)
}

func (r *Repository) DeletePlanFeature(ctx context.Context, id pgtype.UUID) error {
	return r.queries().DeletePlanFeature(ctx, id)
}

func (r *Repository) ListFeaturesActiveByPlanID(ctx context.Context, planId pgtype.UUID)([]db.PlanFeature, error){
	return r.queries().ListFeaturesActiveByPlanID(ctx, planId)
}