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

func (r *Repository) CreateCustomer(ctx context.Context, arg db.CreateCustomersParams) (pgtype.UUID, error) {
	q := db.New(r.db)
	return q.CreateCustomers(ctx, arg)
}

func (r *Repository) DeleteCustomer(ctx context.Context, arg db.DeleteCustomerParams) error {
	q := db.New(r.db)
	return q.DeleteCustomer(ctx, arg)
}

func (r *Repository) GetCustomerByCPF(ctx context.Context, cpf string) (db.Customer, error) {
	q := db.New(r.db)
	return q.GetCustomerByCPF(ctx, cpf)
}

func (r *Repository) GetCustomerById(ctx context.Context, id pgtype.UUID) (db.Customer, error) {
	q := db.New(r.db)
	return q.GetCustomerById(ctx, id)
}

func (r *Repository) ListCustomers(ctx context.Context, companyID pgtype.UUID) ([]db.Customer, error) {
	q := db.New(r.db)
	return q.ListCustomers(ctx, companyID)
}

func (r *Repository) UpdateBalanceDueCustomer(ctx context.Context, arg db.UpdateBalanceDueCustomerParams) error {
	q := db.New(r.db)
	return q.UpdateBalanceDueCustomer(ctx, arg)
}

func (r *Repository) UpdateCustomer(ctx context.Context, arg db.UpdateCustomerParams) error {
	q := db.New(r.db)
	return q.UpdateCustomer(ctx, arg)
}

func (r *Repository) CountCustomers(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	q := db.New(r.db)
	return q.CountCustomers(ctx, companyId)
}

func (r *Repository) GetCustomersPerformanceSummary(ctx context.Context, companyId pgtype.UUID) (db.GetCustomersPerformanceSummaryRow, error) {
	q := db.New(r.db)
	return q.GetCustomersPerformanceSummary(ctx, companyId)
}

func (r *Repository) UpdateCustomerBalance(ctx context.Context, arg db.UpdateCustomerBalanceParams) error {
	q := db.New(r.db)
	return q.UpdateCustomerBalance(ctx, arg)
}

func (r *Repository) ListCustomersPaginate(ctx context.Context, arg db.ListCustomersPaginateParams) ([]db.Customer, error) {
	q := db.New(r.db)
	return q.ListCustomersPaginate(ctx, arg)
}

func (r *Repository) CountCustomersByCompany(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	q := db.New(r.db)
	return q.CountCustomersByCompany(ctx, companyId)
}
