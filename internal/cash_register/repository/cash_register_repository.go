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

func (r *Repository) CreateCashSession(ctx context.Context, arg db.CreateCashSessionParams) (db.CashSession, error) {
	return r.queries().CreateCashSession(ctx, arg)
}

func (r *Repository) GetOpenCashSessionByUser(ctx context.Context, arg db.GetOpenCashSessionByUserParams) (db.CashSession, error) {
	return r.queries().GetOpenCashSessionByUser(ctx, arg)
}

func (r *Repository) GetOpenCashSessionByUserForUpdate(ctx context.Context, arg db.GetOpenCashSessionByUserForUpdateParams) (db.CashSession, error) {
	return r.queries().GetOpenCashSessionByUserForUpdate(ctx, arg)
}

func (r *Repository) GetCashSessionById(ctx context.Context, arg db.GetCashSessionByIdParams) (db.GetCashSessionByIdRow, error) {
	return r.queries().GetCashSessionById(ctx, arg)
}

func (r *Repository) CloseCashSession(ctx context.Context, arg db.CloseCashSessionParams) (db.CashSession, error) {
	return r.queries().CloseCashSession(ctx, arg)
}

func (r *Repository) ListCashSessions(ctx context.Context, arg db.ListCashSessionsParams) ([]db.ListCashSessionsRow, error) {
	return r.queries().ListCashSessions(ctx, arg)
}

func (r *Repository) CreateCashSessionMovement(ctx context.Context, arg db.CreateCashSessionMovementParams) (db.CashSessionMovement, error) {
	return r.queries().CreateCashSessionMovement(ctx, arg)
}

func (r *Repository) ListCashSessionMovements(ctx context.Context, sessionId pgtype.UUID) ([]db.CashSessionMovement, error) {
	return r.queries().ListCashSessionMovements(ctx, sessionId)
}

func (r *Repository) GetCashSessionSalesByMethod(ctx context.Context, arg db.GetCashSessionSalesByMethodParams) ([]db.GetCashSessionSalesByMethodRow, error) {
	return r.queries().GetCashSessionSalesByMethod(ctx, arg)
}

func (r *Repository) GetCashSessionReceiptsByMethod(ctx context.Context, arg db.GetCashSessionReceiptsByMethodParams) ([]db.GetCashSessionReceiptsByMethodRow, error) {
	return r.queries().GetCashSessionReceiptsByMethod(ctx, arg)
}

func (r *Repository) GetCashSessionRefundsByMethod(ctx context.Context, arg db.GetCashSessionRefundsByMethodParams) ([]db.GetCashSessionRefundsByMethodRow, error) {
	return r.queries().GetCashSessionRefundsByMethod(ctx, arg)
}
