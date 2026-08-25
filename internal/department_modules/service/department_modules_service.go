package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/google/uuid"
)

type Service struct {
	repo domain.RepositoryInterface
}

func NewService(repo domain.RepositoryInterface) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) DepartmentHasModule(ctx context.Context, departmentId uuid.UUID, moduleCode string) (bool, error) {
	hasModule, err := s.repo.DepartmentHasModule(ctx, db.DepartmentHasModuleParams{
		DepartmentID: pgconv.OptionalUUIDToPgType(departmentId),
		ModuleCode:   moduleCode,
	})
	if err != nil {
		return hasModule, err
	}

	return hasModule, nil
}

func (s *Service) ListModulesByDepartment(ctx context.Context, departmentId uuid.UUID) ([]domain.ModuleResponse, error) {
	modules, err := s.repo.ListModulesByDepartment(ctx, pgconv.OptionalUUIDToPgType(departmentId))
	if err != nil {
		return []domain.ModuleResponse{}, err
	}

	var response []domain.ModuleResponse

	for _, module := range modules {
		response = append(response, domain.ModuleResponse(module))
	}

	return response, nil
}

func (s *Service) AddModuleToDepartment(ctx context.Context, req domain.AddModuleToDepartmentRequest) error {
	return s.repo.AddModuleToDepartment(ctx, db.AddModuleToDepartmentParams{
		DepartmentID: pgconv.OptionalUUIDToPgType(req.DepartmentID),
		ModuleCode:   req.ModuleCode,
	})
}

func (s *Service) AddModuleToDepartmentTx(ctx context.Context, tx db.DBTX, req domain.AddModuleToDepartmentRequest) error {
	repoTx := db.New(tx)

	return repoTx.AddModuleToDepartment(ctx, db.AddModuleToDepartmentParams{
		DepartmentID: pgconv.OptionalUUIDToPgType(req.DepartmentID),
		ModuleCode:   req.ModuleCode,
	})
}

func (s *Service) RemoveModuleFromDepartment(ctx context.Context, departmentId uuid.UUID, req domain.RemoveModuleFromDepartmentRequest) error {
	return s.repo.RemoveModuleFromDepartment(ctx, db.RemoveModuleFromDepartmentParams{
		DepartmentID: pgconv.OptionalUUIDToPgType(departmentId),
		ModuleCode:   req.ModuleCode,
	})
}

func (s *Service) ReplaceDepartmentModules(ctx context.Context, departmentId uuid.UUID) error {
	return s.repo.ReplaceDepartmentModules(ctx, pgconv.OptionalUUIDToPgType(departmentId))
}
