package domain

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type RepositoryInterface interface {
	DepartmentHasModule(ctx context.Context, arg db.DepartmentHasModuleParams) (bool, error)
	ListModulesByDepartment(ctx context.Context, departmentId pgtype.UUID) ([]db.Module, error)
	AddModuleToDepartment(ctx context.Context, arg db.AddModuleToDepartmentParams) error
	RemoveModuleFromDepartment(ctx context.Context, arg db.RemoveModuleFromDepartmentParams) error
	ReplaceDepartmentModules(ctx context.Context, departmentId pgtype.UUID) error
}
