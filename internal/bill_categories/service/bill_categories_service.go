package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/bill_categories/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/bill_categories/repository"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateBillCategories(ctx context.Context, arg db.CreateBillCategoriesParams) error
	DeleteBillCategories(ctx context.Context, id pgtype.UUID) error
	GetBillCategoriesById(ctx context.Context, id pgtype.UUID) (db.BillCategory, error)
	ListBillCategories(ctx context.Context, companyId pgtype.UUID) ([]db.BillCategory, error)
	ListBillCategoriesActive(ctx context.Context, companyId pgtype.UUID) ([]db.BillCategory, error)
	ToggleBillCategoriesActive(ctx context.Context, arg db.ToggleBillCategoriesActiveParams) error
}

type Service struct {
	repo RepositoryInterface
	pool *pgxpool.Pool
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool) *Service {
	return &Service{
		repo: repo,
		pool: pool,
	}
}

func (s *Service) CreateBillCategories(ctx context.Context, req domain.CreateBillCategoriesRequest) error {
	return s.repo.CreateBillCategories(ctx, db.CreateBillCategoriesParams{
		CompanyID:   pgconv.ParseUUIDToPgType(req.CompanyID),
		Name:        req.Name,
		Description: pgconv.ParseStringToPgText(req.Description),
	})
}

func (s *Service) DeleteBillCategories(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteBillCategories(ctx, pgconv.ParseUUIDToPgType(id))
}

func (s *Service) GetBillCategoriesById(ctx context.Context, id uuid.UUID) (domain.BillCategoryResponse, error) {
	billCategory, err := s.repo.GetBillCategoriesById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.BillCategoryResponse{}, err
	}

	return domain.BillCategoryResponse{
		ID:          pgconv.PgUUIDToUUID(billCategory.ID),
		CompanyID:   pgconv.PgUUIDToUUID(billCategory.CompanyID),
		Name:        billCategory.Name,
		Description: pgconv.ParsePgTextToString(billCategory.Description),
		IsActive:    billCategory.IsActive,
		CreatedAt:   pgconv.PgTimestamptzToTime(billCategory.CreatedAt),
		UpdatedAt:   pgconv.PgTimestamptzToTime(billCategory.UpdatedAt),
		DeletedAt:   pgconv.PgTimestamptzToTime(billCategory.DeletedAt),
	}, nil
}

func (s *Service) ListBillCategories(ctx context.Context, companyId uuid.UUID) ([]domain.BillCategoryResponse, error) {
	billCategories, err := s.repo.ListBillCategories(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.BillCategoryResponse{}, err
	}

	var response []domain.BillCategoryResponse

	for _, billCategory := range billCategories {
		response = append(response, domain.BillCategoryResponse{
			ID:          pgconv.PgUUIDToUUID(billCategory.ID),
			CompanyID:   pgconv.PgUUIDToUUID(billCategory.CompanyID),
			Name:        billCategory.Name,
			Description: pgconv.ParsePgTextToString(billCategory.Description),
			IsActive:    billCategory.IsActive,
			CreatedAt:   pgconv.PgTimestamptzToTime(billCategory.CreatedAt),
			UpdatedAt:   pgconv.PgTimestamptzToTime(billCategory.UpdatedAt),
			DeletedAt:   pgconv.PgTimestamptzToTime(billCategory.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) ListBillCategoriesActive(ctx context.Context, companyId uuid.UUID) ([]domain.BillCategoryResponse, error) {
	billCategories, err := s.repo.ListBillCategoriesActive(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.BillCategoryResponse{}, err
	}

	var response []domain.BillCategoryResponse

	for _, billCategory := range billCategories {
		response = append(response, domain.BillCategoryResponse{
			ID:          pgconv.PgUUIDToUUID(billCategory.ID),
			CompanyID:   pgconv.PgUUIDToUUID(billCategory.CompanyID),
			Name:        billCategory.Name,
			Description: pgconv.ParsePgTextToString(billCategory.Description),
			IsActive:    billCategory.IsActive,
			CreatedAt:   pgconv.PgTimestamptzToTime(billCategory.CreatedAt),
			UpdatedAt:   pgconv.PgTimestamptzToTime(billCategory.UpdatedAt),
			DeletedAt:   pgconv.PgTimestamptzToTime(billCategory.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) ToggleBillCategoriesActive(ctx context.Context, req domain.ToggleBillCategoriesActiveRequest) error {
	return s.repo.ToggleBillCategoriesActive(ctx, db.ToggleBillCategoriesActiveParams{
		ID:       pgconv.ParseUUIDToPgType(req.ID),
		IsActive: req.IsActive,
	})
}
