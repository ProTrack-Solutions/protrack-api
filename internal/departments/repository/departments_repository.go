package repository

import (
	"context"

	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (r *Repository) CreateDepartment(ctx context.Context, arg db.CreateDepartmentParams) (db.Department, error) {
	return r.q.CreateDepartment(ctx, arg)
}

func (r *Repository) DeleteDepartment(ctx context.Context, arg db.DeleteDepartmentParams) error {
	return r.q.DeleteDepartment(ctx, arg)
}

func (r *Repository) GetDepartmentById(ctx context.Context, id pgtype.UUID) (db.Department, error) {
	return r.q.GetDepartmentById(ctx, id)
}

func (r *Repository) ListDepartmentsByCompanyId(ctx context.Context, departmentId pgtype.UUID) ([]db.Department, error) {
	return r.q.ListDepartmentsByCompanyId(ctx, departmentId)
}

func (r *Repository) SetStatusDepartment(ctx context.Context, arg db.SetStatusDepartmentParams) (int64, error) {
	return r.q.SetStatusDepartment(ctx, arg)
}

func (r *Repository) UpdateDepartment(ctx context.Context, arg db.UpdateDepartmentParams) (db.Department, error) {
	return r.q.UpdateDepartment(ctx, arg)
}
