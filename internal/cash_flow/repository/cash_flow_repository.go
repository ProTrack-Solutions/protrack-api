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

func (r *Repository) GetTotalInflowByPeriod(ctx context.Context, arg db.GetTotalInflowByPeriodParams) (float64, error) {
	return r.queries().GetTotalInflowByPeriod(ctx, arg)
}

func (r *Repository) GetTotalOutflowByPeriod(ctx context.Context, arg db.GetTotalOutflowByPeriodParams) (float64, error) {
	return r.queries().GetTotalOutflowByPeriod(ctx, arg)
}

func (r *Repository) GetCashInFlowByCategory(ctx context.Context, companyId pgtype.UUID) ([]db.GetCashInFlowByCategoryRow, error) {
	return r.queries().GetCashInFlowByCategory(ctx, companyId)
}

func (r *Repository) GetCashOutFlowByCategory(ctx context.Context, companyId pgtype.UUID) ([]db.GetCashOutFlowByCategoryRow, error) {
	return r.queries().GetCashOutFlowByCategory(ctx, companyId)
}
