package service

import (
	"context"
	"encoding/json"
	"fmt"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/departments/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/departments/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateDepartment(ctx context.Context, arg db.CreateDepartmentParams) (db.Department, error)
	DeleteDepartment(ctx context.Context, arg db.DeleteDepartmentParams) error
	GetDepartmentById(ctx context.Context, id pgtype.UUID) (db.Department, error)
	ListDepartmentsByCompanyId(ctx context.Context, departmentId pgtype.UUID) ([]db.ListDepartmentsByCompanyIdRow, error)
	SetStatusDepartment(ctx context.Context, arg db.SetStatusDepartmentParams) (int64, error)
	UpdateDepartment(ctx context.Context, arg db.UpdateDepartmentParams) (db.Department, error)
	WithTx(tx db.DBTX) *repository.Repository
}

type Service struct {
	repo                     RepositoryInterface
	pool                     *pgxpool.Pool
	departmentModulesService departmentModulesDomain.ServiceInterface
}

func NewService(repo *repository.Repository, departmentModulesService departmentModulesDomain.ServiceInterface, pool *pgxpool.Pool) *Service {
	return &Service{
		repo:                     repo,
		departmentModulesService: departmentModulesService,
		pool:                     pool,
	}
}

func parseModules(raw interface{}) ([]departmentModulesDomain.ModuleResponse, error) {
	if raw == nil {
		return []departmentModulesDomain.ModuleResponse{}, nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modules: %w", err)
	}

	var modules []departmentModulesDomain.ModuleResponse
	if err := json.Unmarshal(b, &modules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal modules: %w", err)
	}

	return modules, nil
}

func (s *Service) CreateDepartment(ctx context.Context, req domain.CreateDepartmentParams, companyId uuid.UUID, userId uuid.UUID) (domain.DepartmentResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.DepartmentResponse{}, err
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	department, err := txRepo.CreateDepartment(ctx, db.CreateDepartmentParams{
		CompanyID:   pgconv.ParseUUIDToPgType(companyId),
		Name:        req.Name,
		Description: pgconv.ParseStringToPgText(req.Description),
		CreatedBy:   pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.DepartmentResponse{}, err
	}

	for _, mc := range req.ModuleCode {
		if err := s.departmentModulesService.AddModuleToDepartmentTx(ctx, tx, departmentModulesDomain.AddModuleToDepartmentRequest{
			DepartmentID: pgconv.PgUUIDToUUID(department.ID),
			ModuleCode:   mc.Code,
		}); err != nil {
			return domain.DepartmentResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.DepartmentResponse{}, err
	}

	return domain.DepartmentResponse{
		ID:          pgconv.PgUUIDToUUID(department.ID),
		CompanyID:   pgconv.PgUUIDToUUID(department.CompanyID),
		Name:        department.Name,
		Description: pgconv.ParsePgTextToString(department.Description),
		Status:      department.Status.(string),
		CreatedBy:   pgconv.PgUUIDToUUID(department.CreatedBy),
		CreatedAt:   pgconv.PgTimestamptzToTime(department.CreatedAt),
	}, nil
}

func (s *Service) DeleteDepartment(ctx context.Context, departmentId, userId uuid.UUID) error {
	return s.repo.DeleteDepartment(ctx, db.DeleteDepartmentParams{
		ID:        pgconv.ParseUUIDToPgType(departmentId),
		DeletedBy: pgconv.ParseUUIDToPgType(userId),
	})
}

func (s *Service) GetDepartmentById(ctx context.Context, id uuid.UUID) (domain.DepartmentResponse, error) {
	department, err := s.repo.GetDepartmentById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.DepartmentResponse{}, err
	}

	return domain.DepartmentResponse{
		ID:          pgconv.PgUUIDToUUID(department.ID),
		CompanyID:   pgconv.PgUUIDToUUID(department.CompanyID),
		Name:        department.Name,
		Description: pgconv.ParsePgTextToString(department.Description),
		Status:      department.Status.(string),
		CreatedBy:   pgconv.PgUUIDToUUID(department.CreatedBy),
		UpdatedBy:   pgconv.PgUUIDToUUID(department.UpdatedBy),
		DeletedBy:   pgconv.PgUUIDToUUID(department.DeletedBy),
		CreatedAt:   pgconv.PgTimestamptzToTime(department.CreatedAt),
		UpdatedAt:   pgconv.PgTimestamptzToTime(department.UpdatedAt),
		DeletedAt:   pgconv.PgTimestamptzToTime(department.DeletedAt),
	}, nil
}

func (s *Service) ListDepartmentsByCompanyId(ctx context.Context, companyId uuid.UUID) ([]domain.DepartmentResponse, error) {
	departments, err := s.repo.ListDepartmentsByCompanyId(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.DepartmentResponse{}, err
	}

	var response []domain.DepartmentResponse

	for _, department := range departments {
		modules, err := parseModules(department.Modules)
		if err != nil {
			return nil, err
		}

		response = append(response, domain.DepartmentResponse{
			ID:          pgconv.PgUUIDToUUID(department.ID),
			CompanyID:   pgconv.PgUUIDToUUID(department.CompanyID),
			Name:        department.Name,
			Description: pgconv.ParsePgTextToString(department.Description),
			Status:      department.Status.(string),
			CreatedBy:   pgconv.PgUUIDToUUID(department.CreatedBy),
			UpdatedBy:   pgconv.PgUUIDToUUID(department.UpdatedBy),
			DeletedBy:   pgconv.PgUUIDToUUID(department.DeletedBy),
			CreatedAt:   pgconv.PgTimestamptzToTime(department.CreatedAt),
			UpdatedAt:   pgconv.PgTimestamptzToTime(department.UpdatedAt),
			DeletedAt:   pgconv.PgTimestamptzToTime(department.DeletedAt),
			Modules:     modules,
		})
	}

	return response, nil
}

func (s *Service) SetStatusDepartment(ctx context.Context, req domain.SetStatusDepartmentParams, userId uuid.UUID, departmentId uuid.UUID) (int64, error) {
	count, err := s.repo.SetStatusDepartment(ctx, db.SetStatusDepartmentParams{
		ID:        pgconv.ParseUUIDToPgType(departmentId),
		Column2:   req.Status,
		UpdatedBy: pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) UpdateDepartment(ctx context.Context, id uuid.UUID, userId uuid.UUID, req domain.UpdateDepartmentParams) (domain.DepartmentResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.DepartmentResponse{}, err
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	currentDepartment, err := s.repo.GetDepartmentById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.DepartmentResponse{}, err
	}

	arg := db.UpdateDepartmentParams{
		Name:        currentDepartment.Name,
		Description: currentDepartment.Description,
	}

	domain.ApplyUpdateProductCategoryParams(req, &arg)

	department, err := txRepo.UpdateDepartment(ctx, db.UpdateDepartmentParams{
		ID:          pgconv.ParseUUIDToPgType(id),
		Name:        arg.Name,
		Description: arg.Description,
		UpdatedBy:   pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.DepartmentResponse{}, err
	}

	for _, mc := range req.ModuleCode {
		if err := s.departmentModulesService.AddModuleToDepartmentTx(ctx, tx, departmentModulesDomain.AddModuleToDepartmentRequest{
			DepartmentID: pgconv.PgUUIDToUUID(department.ID),
			ModuleCode:   mc.Code,
		}); err != nil {
			return domain.DepartmentResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.DepartmentResponse{}, err
	}

	return domain.DepartmentResponse{
		ID:          pgconv.PgUUIDToUUID(department.ID),
		CompanyID:   pgconv.PgUUIDToUUID(department.CompanyID),
		Name:        department.Name,
		Description: pgconv.ParsePgTextToString(department.Description),
		Status:      department.Status.(string),
		CreatedBy:   pgconv.PgUUIDToUUID(department.CreatedBy),
		UpdatedBy:   pgconv.PgUUIDToUUID(department.UpdatedBy),
		CreatedAt:   pgconv.PgTimestamptzToTime(department.CreatedAt),
		UpdatedAt:   pgconv.PgTimestamptzToTime(department.UpdatedAt),
	}, nil
}
