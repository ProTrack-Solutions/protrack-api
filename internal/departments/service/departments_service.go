package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/departments/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/departments/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RepositoryInterface interface {
	CreateDepartment(ctx context.Context, arg db.CreateDepartmentParams) (db.Department, error)
	DeleteDepartment(ctx context.Context, arg db.DeleteDepartmentParams) error
	GetDepartmentById(ctx context.Context, id pgtype.UUID) (db.Department, error)
	ListDepartmentsByCompanyId(ctx context.Context, departmentId pgtype.UUID) ([]db.Department, error)
	SetStatusDepartment(ctx context.Context, arg db.SetStatusDepartmentParams) (int64, error)
	UpdateDepartment(ctx context.Context, arg db.UpdateDepartmentParams) (db.Department, error)
}

type Service struct {
	repo RepositoryInterface
}

func NewService(repo *repository.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateDepartment(ctx context.Context, req domain.CreateDepartmentParams, companyId uuid.UUID, userId uuid.UUID) (domain.DepartmentResponse, error) {
	department, err := s.repo.CreateDepartment(ctx, db.CreateDepartmentParams{
		CompanyID:   pgconv.ParseUUIDToPgType(companyId),
		Name:        req.Name,
		Description: pgconv.ParseStringToPgText(req.Description),
		CreatedBy:   pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.DepartmentResponse{}, err
	}

	return domain.DepartmentResponse{
		ID:          pgconv.PgUUIDToUUID(department.ID),
		CompanyID:   pgconv.PgUUIDToUUID(department.CompanyID),
		Name:        department.Name,
		Description: pgconv.ParsePgTextToString(department.Description),
		Status:      department.Status,
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
		Status:      department.Status,
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
		response = append(response, domain.DepartmentResponse{
			ID:          pgconv.PgUUIDToUUID(department.ID),
			CompanyID:   pgconv.PgUUIDToUUID(department.CompanyID),
			Name:        department.Name,
			Description: pgconv.ParsePgTextToString(department.Description),
			Status:      department.Status,
			CreatedBy:   pgconv.PgUUIDToUUID(department.CreatedBy),
			UpdatedBy:   pgconv.PgUUIDToUUID(department.UpdatedBy),
			DeletedBy:   pgconv.PgUUIDToUUID(department.DeletedBy),
			CreatedAt:   pgconv.PgTimestamptzToTime(department.CreatedAt),
			UpdatedAt:   pgconv.PgTimestamptzToTime(department.UpdatedAt),
			DeletedAt:   pgconv.PgTimestamptzToTime(department.DeletedAt),
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
	currentDepartment, err := s.repo.GetDepartmentById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.DepartmentResponse{}, err
	}

	arg := db.UpdateDepartmentParams{
		Name:        currentDepartment.Name,
		Description: currentDepartment.Description,
	}

	domain.ApplyUpdateProductCategoryParams(req, &arg)

	department, err := s.repo.UpdateDepartment(ctx, db.UpdateDepartmentParams{
		ID:          pgconv.ParseUUIDToPgType(id),
		Name:        arg.Name,
		Description: arg.Description,
		UpdatedBy:   pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.DepartmentResponse{}, err
	}

	return domain.DepartmentResponse{
		ID:          pgconv.PgUUIDToUUID(department.ID),
		CompanyID:   pgconv.PgUUIDToUUID(department.CompanyID),
		Name:        department.Name,
		Description: pgconv.ParsePgTextToString(department.Description),
		Status:      department.Status,
		CreatedBy:   pgconv.PgUUIDToUUID(department.CreatedBy),
		UpdatedBy:   pgconv.PgUUIDToUUID(department.UpdatedBy),
		CreatedAt:   pgconv.PgTimestamptzToTime(department.CreatedAt),
		UpdatedAt:   pgconv.PgTimestamptzToTime(department.UpdatedAt),
	}, nil
}
