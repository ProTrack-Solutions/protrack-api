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

func (r *Repository) DepartmentHasModule(ctx context.Context, arg db.DepartmentHasModuleParams) (bool, error) {
	return r.queries().DepartmentHasModule(ctx, arg)
}

func (r *Repository) ListModulesByDepartment(ctx context.Context, departmentId pgtype.UUID) ([]db.Module, error) {
	return r.queries().ListModulesByDepartment(ctx, departmentId)
}

func (r *Repository) AddModuleToDepartment(ctx context.Context, arg db.AddModuleToDepartmentParams) error {
	return r.queries().AddModuleToDepartment(ctx, arg)
}

func (r *Repository) RemoveModuleFromDepartment(ctx context.Context, arg db.RemoveModuleFromDepartmentParams) error {
	return r.queries().RemoveModuleFromDepartment(ctx, arg)
}

func (r *Repository) ReplaceDepartmentModules(ctx context.Context, departmentId pgtype.UUID) error {
	return r.queries().ReplaceDepartmentModules(ctx, departmentId)
}
