package service

import (
	"context"
	"errors"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/domain/enums"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/products_categories/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/products_categories/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RepositoryInterface interface {
	CreateProductsCategories(ctx context.Context, arg db.CreateProductCategoryParams) (db.ProductCategory, error)
	DeleteProductCategory(ctx context.Context, arg db.DeleteProductCategoryParams) error
	GetProductCategoryById(ctx context.Context, id pgtype.UUID) (db.ProductCategory, error)
	ListProductCategoryByCompanyId(ctx context.Context, companyId pgtype.UUID) ([]db.ProductCategory, error)
	SetProductCategoryStatus(ctx context.Context, arg db.SetProductCategoryStatusParams) (int64, error)
	UpdateProductCategory(ctx context.Context, arg db.UpdateProductCategoryParams) (db.ProductCategory, error)
}

type Service struct {
	repo RepositoryInterface
}

func NewService(repo *repository.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateProductCategory(ctx context.Context, userId, companyId uuid.UUID, req domain.CreateProductCategoryRequest) error {
	_, err := s.repo.CreateProductsCategories(ctx, db.CreateProductCategoryParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Name:      req.Name,
		Color:     pgconv.ParseStringToPgType(req.Color),
		CreatedBy: pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) DeleteProductCategory(ctx context.Context, req domain.DeleteProductCategoryRequest) error {
	return s.repo.DeleteProductCategory(ctx, db.DeleteProductCategoryParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		DeletedBy: pgconv.ParseUUIDToPgType(req.DeletedBy),
	})
}

func (s *Service) GetProductCategoryById(ctx context.Context, id uuid.UUID) (domain.ProductCategoryResponse, error) {
	category, err := s.repo.GetProductCategoryById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.ProductCategoryResponse{}, err
	}

	statusStr, ok := category.Status.(string)
	if !ok {
		return domain.ProductCategoryResponse{}, errors.New("status inválido")
	}

	return domain.ProductCategoryResponse{
		ID:        pgconv.PgUUIDToUUID(category.ID),
		CompanyID: pgconv.PgUUIDToUUID(category.CompanyID),
		Name:      category.Name,
		Color:     pgconv.ParsePgTextToString(category.Color),
		Status:    enums.Status(statusStr),
		CreatedBy: pgconv.PgUUIDToUUID(category.CreatedBy),
		UpdatedBy: pgconv.PgUUIDToUUID(category.UpdatedBy),
		DeletedBy: pgconv.PgUUIDToUUID(category.DeletedBy),
		CreatedAt: pgconv.PgTimestamptzToTime(category.CreatedAt),
		UpdatedAt: pgconv.PgTimestamptzToTime(category.UpdatedAt),
		DeletedAt: pgconv.PgTimestamptzToTime(category.DeletedAt),
	}, nil
}

func (s *Service) ListProductCategoryByCompanyId(ctx context.Context, companyId uuid.UUID) ([]domain.ProductCategoryResponse, error) {
	categories, err := s.repo.ListProductCategoryByCompanyId(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ProductCategoryResponse{}, err
	}

	var response []domain.ProductCategoryResponse

	for _, category := range categories {

		statusStr, ok := category.Status.(string)
		if !ok {
			return []domain.ProductCategoryResponse{}, errors.New("status inválido")
		}

		response = append(response, domain.ProductCategoryResponse{
			ID:        pgconv.PgUUIDToUUID(category.ID),
			CompanyID: pgconv.PgUUIDToUUID(category.CompanyID),
			Name:      category.Name,
			Color:     pgconv.ParsePgTextToString(category.Color),
			Status:    enums.Status(statusStr),
			CreatedBy: pgconv.PgUUIDToUUID(category.CreatedBy),
			UpdatedBy: pgconv.PgUUIDToUUID(category.UpdatedBy),
			DeletedBy: pgconv.PgUUIDToUUID(category.DeletedBy),
			CreatedAt: pgconv.PgTimestamptzToTime(category.CreatedAt),
			UpdatedAt: pgconv.PgTimestamptzToTime(category.UpdatedAt),
			DeletedAt: pgconv.PgTimestamptzToTime(category.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) SetProductCategoryStatus(ctx context.Context, req domain.SetProductCategoryStatusRequest) (int64, error) {
	count, err := s.repo.SetProductCategoryStatus(ctx, db.SetProductCategoryStatusParams{
		ID:      pgconv.ParseUUIDToPgType(req.ID),
		Column2: req.Status,
	})
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) UpdateProductCategory(ctx context.Context, id uuid.UUID, req domain.UpdateProductCategoryRequest) (domain.ProductCategoryResponse, error) {
	currentCategory, err := s.repo.GetProductCategoryById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.ProductCategoryResponse{}, err
	}

	arg := db.UpdateProductCategoryParams{
		ID:        currentCategory.ID,
		Name:      currentCategory.Name,
		Color:     currentCategory.Color,
		UpdatedBy: currentCategory.UpdatedBy,
	}

	domain.ApplyUpdateProductCategoryParams(req, &arg)

	category, err := s.repo.UpdateProductCategory(ctx, arg)
	if err != nil {
		return domain.ProductCategoryResponse{}, err
	}

	statusStr, ok := category.Status.(string)
	if !ok {
		return domain.ProductCategoryResponse{}, errors.New("status inválido")
	}

	return domain.ProductCategoryResponse{
		ID:        pgconv.PgUUIDToUUID(category.ID),
		CompanyID: pgconv.PgUUIDToUUID(category.CompanyID),
		Name:      category.Name,
		Color:     pgconv.ParsePgTextToString(category.Color),
		Status:    enums.Status(statusStr),
		CreatedBy: pgconv.PgUUIDToUUID(category.CreatedBy),
		UpdatedBy: pgconv.PgUUIDToUUID(category.UpdatedBy),
		CreatedAt: pgconv.PgTimestamptzToTime(category.CreatedAt),
		UpdatedAt: pgconv.PgTimestamptzToTime(category.UpdatedAt),
	}, nil
}
