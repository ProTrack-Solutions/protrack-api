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

func (r *Repository) CreateDepartment(ctx context.Context, arg db.CreateDepartmentParams) (db.Department, error) {
	return r.queries().CreateDepartment(ctx, arg)
}

func (r *Repository) DeleteDepartment(ctx context.Context, arg db.DeleteDepartmentParams) error {
	return r.queries().DeleteDepartment(ctx, arg)
}

func (r *Repository) GetDepartmentById(ctx context.Context, id pgtype.UUID) (db.Department, error) {
	return r.queries().GetDepartmentById(ctx, id)
}

func (r *Repository) ListDepartmentsByCompanyId(ctx context.Context, departmentId pgtype.UUID) ([]db.ListDepartmentsByCompanyIdRow, error) {
	return r.queries().ListDepartmentsByCompanyId(ctx, departmentId)
}

func (r *Repository) SetStatusDepartment(ctx context.Context, arg db.SetStatusDepartmentParams) (int64, error) {
	return r.queries().SetStatusDepartment(ctx, arg)
}

func (r *Repository) UpdateDepartment(ctx context.Context, arg db.UpdateDepartmentParams) (db.Department, error) {
	return r.queries().UpdateDepartment(ctx, arg)
}
