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

func (r *Repository) ListCashRegisters(ctx context.Context, companyId pgtype.UUID) ([]db.ListCashRegistersRow, error) {
	return r.queries().ListCashRegisters(ctx, companyId)
}

func (r *Repository) GetCashRegisterById(ctx context.Context, arg db.GetCashRegisterByIdParams) (db.CashRegister, error) {
	return r.queries().GetCashRegisterById(ctx, arg)
}

func (r *Repository) ListActiveCashRegisterIds(ctx context.Context, companyId pgtype.UUID) ([]pgtype.UUID, error) {
	return r.queries().ListActiveCashRegisterIds(ctx, companyId)
}

func (r *Repository) CreateCashRegister(ctx context.Context, arg db.CreateCashRegisterParams) (db.CashRegister, error) {
	return r.queries().CreateCashRegister(ctx, arg)
}

func (r *Repository) UpdateCashRegister(ctx context.Context, arg db.UpdateCashRegisterParams) (db.CashRegister, error) {
	return r.queries().UpdateCashRegister(ctx, arg)
}

func (r *Repository) CreateCashSession(ctx context.Context, arg db.CreateCashSessionParams) (db.CashSession, error) {
	return r.queries().CreateCashSession(ctx, arg)
}

func (r *Repository) GetOpenCashSessionByRegister(ctx context.Context, registerId pgtype.UUID) (db.CashSession, error) {
	return r.queries().GetOpenCashSessionByRegister(ctx, registerId)
}

func (r *Repository) GetOpenCashSessionByRegisterForUpdate(ctx context.Context, registerId pgtype.UUID) (db.CashSession, error) {
	return r.queries().GetOpenCashSessionByRegisterForUpdate(ctx, registerId)
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

func (r *Repository) UserTookPartInCashSession(ctx context.Context, arg db.UserTookPartInCashSessionParams) (bool, error) {
	return r.queries().UserTookPartInCashSession(ctx, arg)
}

func (r *Repository) CreateCashSessionMovement(ctx context.Context, arg db.CreateCashSessionMovementParams) (db.CashSessionMovement, error) {
	return r.queries().CreateCashSessionMovement(ctx, arg)
}

func (r *Repository) ListCashSessionMovements(ctx context.Context, sessionId pgtype.UUID) ([]db.ListCashSessionMovementsRow, error) {
	return r.queries().ListCashSessionMovements(ctx, sessionId)
}

func (r *Repository) GetCashSessionSalesByMethod(ctx context.Context, sessionId pgtype.UUID) ([]db.GetCashSessionSalesByMethodRow, error) {
	return r.queries().GetCashSessionSalesByMethod(ctx, sessionId)
}

func (r *Repository) GetCashSessionReceiptsByMethod(ctx context.Context, sessionId pgtype.UUID) ([]db.GetCashSessionReceiptsByMethodRow, error) {
	return r.queries().GetCashSessionReceiptsByMethod(ctx, sessionId)
}

func (r *Repository) GetCashSessionRefundsByMethod(ctx context.Context, sessionId pgtype.UUID) ([]db.GetCashSessionRefundsByMethodRow, error) {
	return r.queries().GetCashSessionRefundsByMethod(ctx, sessionId)
}

func (r *Repository) GetCashSessionByOperator(ctx context.Context, sessionId pgtype.UUID) ([]db.GetCashSessionByOperatorRow, error) {
	return r.queries().GetCashSessionByOperator(ctx, sessionId)
}

func (r *Repository) EnsureDefaultCashRegister(ctx context.Context, companyId pgtype.UUID) error {
	return r.queries().EnsureDefaultCashRegister(ctx, companyId)
}

func (r *Repository) LockCashSessions(ctx context.Context, ids []pgtype.UUID) ([]db.CashSession, error) {
	return r.queries().LockCashSessions(ctx, ids)
}

func (r *Repository) CreateCashTransfer(ctx context.Context, arg db.CreateCashTransferParams) (db.CashTransfer, error) {
	return r.queries().CreateCashTransfer(ctx, arg)
}

func (r *Repository) GetCashTransferForUpdate(ctx context.Context, arg db.GetCashTransferForUpdateParams) (db.CashTransfer, error) {
	return r.queries().GetCashTransferForUpdate(ctx, arg)
}

func (r *Repository) ResolveCashTransfer(ctx context.Context, arg db.ResolveCashTransferParams) error {
	return r.queries().ResolveCashTransfer(ctx, arg)
}

func (r *Repository) ListPendingCashTransfers(ctx context.Context, sessionId pgtype.UUID) ([]db.ListPendingCashTransfersRow, error) {
	return r.queries().ListPendingCashTransfers(ctx, sessionId)
}

func (r *Repository) CountPendingCashTransfers(ctx context.Context, sessionId pgtype.UUID) (int32, error) {
	return r.queries().CountPendingCashTransfers(ctx, sessionId)
}
