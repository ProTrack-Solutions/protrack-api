package domain

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RepositoryInterface interface {
	DepartmentHasModule(ctx context.Context, arg db.DepartmentHasModuleParams) (bool, error)
	ListModulesByDepartment(ctx context.Context, departmentId pgtype.UUID) ([]db.Module, error)
	AddModuleToDepartment(ctx context.Context, arg db.AddModuleToDepartmentParams) error
	RemoveModuleFromDepartment(ctx context.Context, arg db.RemoveModuleFromDepartmentParams) error
	ReplaceDepartmentModules(ctx context.Context, departmentId pgtype.UUID) error
}

type ServiceInterface interface {
	DepartmentHasModule(ctx context.Context, departmentId uuid.UUID, moduleCode string) (bool, error)
	ListModulesByDepartment(ctx context.Context, departmentId uuid.UUID) ([]ModuleResponse, error)
	AddModuleToDepartment(ctx context.Context, req AddModuleToDepartmentRequest) error
	RemoveModuleFromDepartment(ctx context.Context, departmentId uuid.UUID, req RemoveModuleFromDepartmentRequest) error
	ReplaceDepartmentModules(ctx context.Context, departmentId uuid.UUID) error
	AddModuleToDepartmentTx(ctx context.Context, tx db.DBTX, req AddModuleToDepartmentRequest) error
}

type ModuleResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type AddModuleToDepartmentRequest struct {
	DepartmentID uuid.UUID `json:"department_id"`
	ModuleCode   string    `json:"module_code"`
}

type RemoveModuleFromDepartmentRequest struct {
	ModuleCode string `json:"module_code"`
}
