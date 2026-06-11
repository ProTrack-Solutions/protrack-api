package repository

import (
	"context"

	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
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

func (r *Repository) CreateAccountReceivable(ctx context.Context, arg db.CreateAccountReceivableParams) error {
	return r.queries().CreateAccountReceivable(ctx, arg)
}

func (r *Repository) GetCustomerDebtSummary(ctx context.Context, customerId pgtype.UUID) (db.GetCustomerDebtSummaryRow, error) {
	return r.queries().GetCustomerDebtSummary(ctx, customerId)
}

func (r *Repository) GetPendingReceivablesByCustomer(ctx context.Context, arg db.GetPendingReceivablesByCustomerParams) ([]db.AccountsReceivable, error) {
	return r.queries().GetPendingReceivablesByCustomer(ctx, arg)
}

func (r *Repository) GetReceivablesBySale(ctx context.Context, saleId pgtype.UUID) ([]db.AccountsReceivable, error) {
	return r.queries().GetReceivablesBySale(ctx, saleId)
}

func (r *Repository) ListOverdueReceivables(ctx context.Context, companyId pgtype.UUID) ([]db.ListOverdueReceivablesRow, error) {
	return r.queries().ListOverdueReceivables(ctx, companyId)
}

func (r *Repository) UpdateAccountReceivableBalance(ctx context.Context, arg db.UpdateAccountReceivableBalanceParams) (pgtype.UUID, error) {
	return r.queries().UpdateAccountReceivableBalance(ctx, arg)
}

func (r *Repository) GetTotalOpenAmountByCompany(ctx context.Context, companyId pgtype.UUID) (db.GetTotalOpenAmountByCompanyRow, error) {
	return r.queries().GetTotalOpenAmountByCompany(ctx, companyId)
}

func (r *Repository) GetTotalOverdueAmountByCompany(ctx context.Context, companyId pgtype.UUID) (db.GetTotalOverdueAmountByCompanyRow, error) {
	return r.queries().GetTotalOverdueAmountByCompany(ctx, companyId)
}
