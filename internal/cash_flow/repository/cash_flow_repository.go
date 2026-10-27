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

func (r *Repository) GetCashInFlowCategoryByPeriod(ctx context.Context, arg db.GetCashInFlowCategoryByPeriodParams) ([]db.GetCashInFlowCategoryByPeriodRow, error) {
	return r.queries().GetCashInFlowCategoryByPeriod(ctx, arg)
}

func (r *Repository) GetCashOutFlowCategoryByPeriod(ctx context.Context, arg db.GetCashOutFlowCategoryByPeriodParams) ([]db.GetCashOutFlowCategoryByPeriodRow, error) {
	return r.queries().GetCashOutFlowCategoryByPeriod(ctx, arg)
}

func (r *Repository) GetCompanyTimezone(ctx context.Context, companyId pgtype.UUID) (string, error) {
	return r.queries().GetCompanyTimezone(ctx, companyId)
}

func (r *Repository) GetCashFlowSeries(ctx context.Context, arg db.GetCashFlowSeriesParams) ([]db.GetCashFlowSeriesRow, error) {
	return r.queries().GetCashFlowSeries(ctx, arg)
}

func (r *Repository) GetCashFlowTotalsByRange(ctx context.Context, arg db.GetCashFlowTotalsByRangeParams) (db.GetCashFlowTotalsByRangeRow, error) {
	return r.queries().GetCashFlowTotalsByRange(ctx, arg)
}

func (r *Repository) GetCashInFlowCategoryByRange(ctx context.Context, arg db.GetCashInFlowCategoryByRangeParams) ([]db.GetCashInFlowCategoryByRangeRow, error) {
	return r.queries().GetCashInFlowCategoryByRange(ctx, arg)
}

func (r *Repository) GetCashOutFlowCategoryByRange(ctx context.Context, arg db.GetCashOutFlowCategoryByRangeParams) ([]db.GetCashOutFlowCategoryByRangeRow, error) {
	return r.queries().GetCashOutFlowCategoryByRange(ctx, arg)
}

func (r *Repository) GetCashInFlowPaymentMethodByRange(ctx context.Context, arg db.GetCashInFlowPaymentMethodByRangeParams) ([]db.GetCashInFlowPaymentMethodByRangeRow, error) {
	return r.queries().GetCashInFlowPaymentMethodByRange(ctx, arg)
}

func (r *Repository) GetCashFlowForecast(ctx context.Context, arg db.GetCashFlowForecastParams) ([]db.GetCashFlowForecastRow, error) {
	return r.queries().GetCashFlowForecast(ctx, arg)
}

func (r *Repository) GetCashFlowOverdue(ctx context.Context, arg db.GetCashFlowOverdueParams) (db.GetCashFlowOverdueRow, error) {
	return r.queries().GetCashFlowOverdue(ctx, arg)
}

func (r *Repository) ListRecentCashMovements(ctx context.Context, arg db.ListRecentCashMovementsParams) ([]db.ListRecentCashMovementsRow, error) {
	return r.queries().ListRecentCashMovements(ctx, arg)
}
