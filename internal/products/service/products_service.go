package service

import (
	"context"
	"time"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	globalDomain "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/products/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/products/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateProduct(ctx context.Context, arg db.CreateProductParams) (db.Product, error)
	DeleteProduct(ctx context.Context, arg db.DeleteProductParams) error
	GetProductByBarcode(ctx context.Context, barcode pgtype.Text) (db.Product, error)
	GetProductById(ctx context.Context, id pgtype.UUID) (db.Product, error)
	ListProductsByCategoryId(ctx context.Context, arg db.ListProductsByCategoryIdParams) ([]db.Product, error)
	ListProductsByCompany(ctx context.Context, companyId pgtype.UUID) ([]db.ListProductsByCompanyRow, error)
	UpdateProduct(ctx context.Context, arg db.UpdateProductParams) (db.Product, error)
	DecrementStock(ctx context.Context, arg db.DecrementStockParams) error
	CountProducts(ctx context.Context, companyId pgtype.UUID) (int64, error)
	GetProductsPerformanceSummary(ctx context.Context, companyId pgtype.UUID) (db.GetProductsPerformanceSummaryRow, error)
	GetCostTotalStock(ctx context.Context, companyId pgtype.UUID) (float64, error)
	GetTop5BestSellingProducts(ctx context.Context, companyId pgtype.UUID) ([]db.GetTop5BestSellingProductsRow, error)
	GetInventoryReport(ctx context.Context, arg db.GetInventoryReportParams) ([]db.GetInventoryReportRow, error)
	ListProductsByDate(ctx context.Context, arg db.ListProductsByDateParams) ([]db.ListProductsByDateRow, error)
	ListProductBuCategoryIdAndDate(ctx context.Context, arg db.ListProductsByCategoryAndDateParams) ([]db.ListProductsByCategoryAndDateRow, error)
	CountProductsByCompany(ctx context.Context, companyID pgtype.UUID) (int64, error)
	ListProductsByCompanyPaginated(ctx context.Context, arg db.ListProductsByCompanyPaginatedParams) ([]db.ListProductsByCompanyPaginatedRow, error)
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

func (s *Service) CreateProduct(ctx context.Context, req domain.CreateProductRequest) (domain.ProductResponse, error) {
	product, err := s.repo.CreateProduct(ctx, db.CreateProductParams{
		CompanyID:   pgconv.ParseUUIDToPgType(req.CompanyID),
		Name:        req.Name,
		Description: pgconv.ParseStringToPgText(req.Description),
		CategoryID:  pgconv.ParseUUIDToPgType(req.CategoryID),
		Barcode:     pgconv.ParseStringToPgText(req.Barcode),
		Quantity:    req.Quantity,
		Size:        pgconv.ParseStringToPgText(req.Size),
		CostPrice:   pgconv.Float64ToPgNumeric(req.CostPrice),
		SalePrice:   pgconv.Float64ToPgNumeric(req.SalePrice),
		CreatedBy:   pgconv.ParseUUIDToPgType(req.CreatedBy),
	})
	if err != nil {
		return domain.ProductResponse{}, err
	}

	return domain.ProductResponse{
		ID:          pgconv.PgUUIDToUUID(product.ID),
		CompanyID:   pgconv.PgUUIDToUUID(product.CompanyID),
		CategoryID:  pgconv.PgUUIDToUUID(product.CategoryID),
		Name:        product.Name,
		Description: pgconv.ParsePgTextToString(product.Description),
		Barcode:     pgconv.ParsePgTextToString(product.Barcode),
		Quantity:    product.Quantity,
		Size:        pgconv.ParsePgTextToString(product.Size),
		CostPrice:   pgconv.PgNumericToFloat64(product.CostPrice),
		SalePrice:   pgconv.PgNumericToFloat64(product.SalePrice),
		CreatedBy:   pgconv.PgUUIDToUUID(product.CreatedBy),
		CreatedAt:   pgconv.PgTimestamptzToTime(product.CreatedAt),
	}, nil
}

func (s *Service) DeleteProduct(ctx context.Context, req domain.DeleteProductRequest) error {
	return s.repo.DeleteProduct(ctx, db.DeleteProductParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		DeletedBy: pgconv.ParseUUIDToPgType(req.DeletedBy),
	})
}

func (s *Service) GetProductByBarcode(ctx context.Context, barcode string) (domain.ProductResponse, error) {
	product, err := s.repo.GetProductByBarcode(ctx, pgconv.ParseStringToPgText(barcode))
	if err != nil {
		return domain.ProductResponse{}, err
	}

	return domain.ProductResponse{
		ID:          pgconv.PgUUIDToUUID(product.ID),
		CompanyID:   pgconv.PgUUIDToUUID(product.CompanyID),
		CategoryID:  pgconv.PgUUIDToUUID(product.CategoryID),
		Name:        product.Name,
		Description: pgconv.ParsePgTextToString(product.Description),
		Barcode:     pgconv.ParsePgTextToString(product.Barcode),
		Quantity:    product.Quantity,
		Size:        pgconv.ParsePgTextToString(product.Size),
		CostPrice:   pgconv.PgNumericToFloat64(product.CostPrice),
		SalePrice:   pgconv.PgNumericToFloat64(product.SalePrice),
		CreatedBy:   pgconv.PgUUIDToUUID(product.CreatedBy),
		UpdatedBy:   pgconv.PgUUIDToUUID(product.UpdatedBy),
		DeletedBy:   pgconv.PgUUIDToUUID(product.DeletedBy),
		CreatedAt:   pgconv.PgTimestamptzToTime(product.CreatedAt),
		UpdatedAt:   pgconv.PgTimestamptzToTime(product.UpdatedAt),
		DeletedAt:   pgconv.PgTimestamptzToTime(product.DeletedAt),
	}, nil
}

func (s *Service) GetProductById(ctx context.Context, id uuid.UUID) (domain.ProductResponse, error) {
	product, err := s.repo.GetProductById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.ProductResponse{}, err
	}

	return domain.ProductResponse{
		ID:          pgconv.PgUUIDToUUID(product.ID),
		CompanyID:   pgconv.PgUUIDToUUID(product.CompanyID),
		CategoryID:  pgconv.PgUUIDToUUID(product.CategoryID),
		Name:        product.Name,
		Description: pgconv.ParsePgTextToString(product.Description),
		Barcode:     pgconv.ParsePgTextToString(product.Barcode),
		Quantity:    product.Quantity,
		Size:        pgconv.ParsePgTextToString(product.Size),
		CostPrice:   pgconv.PgNumericToFloat64(product.CostPrice),
		SalePrice:   pgconv.PgNumericToFloat64(product.SalePrice),
		CreatedBy:   pgconv.PgUUIDToUUID(product.CreatedBy),
		UpdatedBy:   pgconv.PgUUIDToUUID(product.UpdatedBy),
		DeletedBy:   pgconv.PgUUIDToUUID(product.DeletedBy),
		CreatedAt:   pgconv.PgTimestamptzToTime(product.CreatedAt),
		UpdatedAt:   pgconv.PgTimestamptzToTime(product.UpdatedAt),
		DeletedAt:   pgconv.PgTimestamptzToTime(product.DeletedAt),
	}, nil
}

func (s *Service) ListProductsByCategoryId(ctx context.Context, categoryId, companyID uuid.UUID) ([]domain.ProductResponse, error) {
	products, err := s.repo.ListProductsByCategoryId(ctx, db.ListProductsByCategoryIdParams{
		CategoryID: pgconv.ParseUUIDToPgType(categoryId),
		CompanyID:  pgconv.ParseUUIDToPgType(companyID),
	})
	if err != nil {
		return []domain.ProductResponse{}, err
	}

	var response []domain.ProductResponse

	for _, product := range products {
		response = append(response, domain.ProductResponse{
			ID:          pgconv.PgUUIDToUUID(product.ID),
			CompanyID:   pgconv.PgUUIDToUUID(product.CompanyID),
			CategoryID:  pgconv.PgUUIDToUUID(product.CategoryID),
			Name:        product.Name,
			Description: pgconv.ParsePgTextToString(product.Description),
			Barcode:     pgconv.ParsePgTextToString(product.Barcode),
			Quantity:    product.Quantity,
			Size:        pgconv.ParsePgTextToString(product.Size),
			CostPrice:   pgconv.PgNumericToFloat64(product.CostPrice),
			SalePrice:   pgconv.PgNumericToFloat64(product.SalePrice),
			CreatedBy:   pgconv.PgUUIDToUUID(product.CreatedBy),
			UpdatedBy:   pgconv.PgUUIDToUUID(product.UpdatedBy),
			DeletedBy:   pgconv.PgUUIDToUUID(product.DeletedBy),
			CreatedAt:   pgconv.PgTimestamptzToTime(product.CreatedAt),
			UpdatedAt:   pgconv.PgTimestamptzToTime(product.UpdatedAt),
			DeletedAt:   pgconv.PgTimestamptzToTime(product.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) ListProductsByCompany(ctx context.Context, companyId uuid.UUID) ([]domain.ListProductsByCompanyRow, error) {
	products, err := s.repo.ListProductsByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListProductsByCompanyRow{}, err
	}

	var response []domain.ListProductsByCompanyRow

	for _, product := range products {
		response = append(response, domain.ListProductsByCompanyRow{
			ID:           pgconv.PgUUIDToUUID(product.ID),
			CompanyID:    pgconv.PgUUIDToUUID(product.CompanyID),
			CategoryID:   pgconv.PgUUIDToUUID(product.CategoryID),
			Name:         product.Name,
			Description:  pgconv.ParsePgTextToString(product.Description),
			Barcode:      pgconv.ParsePgTextToString(product.Barcode),
			Quantity:     product.Quantity,
			Size:         pgconv.ParsePgTextToString(product.Size),
			CostPrice:    pgconv.PgNumericToFloat64(product.CostPrice),
			SalePrice:    pgconv.PgNumericToFloat64(product.SalePrice),
			CreatedBy:    pgconv.PgUUIDToUUID(product.CreatedBy),
			UpdatedBy:    pgconv.PgUUIDToUUID(product.UpdatedBy),
			DeletedBy:    pgconv.PgUUIDToUUID(product.DeletedBy),
			CreatedAt:    pgconv.PgTimestamptzToTime(product.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(product.UpdatedAt),
			DeletedAt:    pgconv.PgTimestamptzToTime(product.DeletedAt),
			CategoryName: product.CategoryName,
		})
	}

	return response, nil
}

func (s *Service) ListProductsByCompanyPaginated(ctx context.Context, companyId uuid.UUID, pagination globalDomain.PaginationParams) (domain.ProductPaginatedResponse, error) {
	total, err := s.repo.CountProductsByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.ProductPaginatedResponse{}, err
	}

	products, err := s.repo.ListProductsByCompanyPaginated(ctx, db.ListProductsByCompanyPaginatedParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Limit:     pagination.PerPage,
		Offset:    (pagination.Page - 1) * pagination.PerPage,
	})
	if err != nil {
		return domain.ProductPaginatedResponse{}, err
	}

	var response []domain.ListProductsByCompanyRow

	var totalValueInStock float64

	for _, product := range products {

		totalValueInStock += pgconv.PgNumericToFloat64(product.CostPrice) * float64(product.Quantity)
		response = append(response, domain.ListProductsByCompanyRow{
			ID:           pgconv.PgUUIDToUUID(product.ID),
			CompanyID:    pgconv.PgUUIDToUUID(product.CompanyID),
			CategoryID:   pgconv.PgUUIDToUUID(product.CategoryID),
			Name:         product.Name,
			Description:  pgconv.ParsePgTextToString(product.Description),
			Barcode:      pgconv.ParsePgTextToString(product.Barcode),
			Quantity:     product.Quantity,
			Size:         pgconv.ParsePgTextToString(product.Size),
			CostPrice:    pgconv.PgNumericToFloat64(product.CostPrice),
			SalePrice:    pgconv.PgNumericToFloat64(product.SalePrice),
			CreatedBy:    pgconv.PgUUIDToUUID(product.CreatedBy),
			UpdatedBy:    pgconv.PgUUIDToUUID(product.UpdatedBy),
			DeletedBy:    pgconv.PgUUIDToUUID(product.DeletedBy),
			CreatedAt:    pgconv.PgTimestamptzToTime(product.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(product.UpdatedAt),
			DeletedAt:    pgconv.PgTimestamptzToTime(product.DeletedAt),
			CategoryName: product.CategoryName,
		})
	}

	paginationResponse := globalDomain.NewPaginatedResponse(response, total, pagination)

	return domain.ProductPaginatedResponse{
		PaginatedResponse: paginationResponse,
		TotalValueInStock: totalValueInStock,
	}, nil
}

func (s *Service) UpdateProduct(ctx context.Context, id uuid.UUID, req domain.UpdateProductRequest) (domain.ProductResponse, error) {
	currentProduct, err := s.repo.GetProductById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.ProductResponse{}, err
	}

	arg := db.UpdateProductParams{
		ID:          currentProduct.ID,
		Name:        currentProduct.Name,
		Description: currentProduct.Description,
		CategoryID:  currentProduct.CategoryID,
		Barcode:     currentProduct.Barcode,
		Quantity:    currentProduct.Quantity,
		Size:        currentProduct.Size,
		CostPrice:   currentProduct.CostPrice,
		SalePrice:   currentProduct.SalePrice,
		UpdatedBy:   currentProduct.UpdatedBy,
	}

	domain.ApplyUpdateProductCategoryParams(req, &arg)

	product, err := s.repo.UpdateProduct(ctx, arg)
	if err != nil {
		return domain.ProductResponse{}, err
	}

	return domain.ProductResponse{
		ID:          pgconv.PgUUIDToUUID(product.ID),
		CompanyID:   pgconv.PgUUIDToUUID(product.CompanyID),
		CategoryID:  pgconv.PgUUIDToUUID(product.CategoryID),
		Name:        product.Name,
		Description: pgconv.ParsePgTextToString(product.Description),
		Barcode:     pgconv.ParsePgTextToString(product.Barcode),
		Quantity:    product.Quantity,
		Size:        pgconv.ParsePgTextToString(product.Size),
		CostPrice:   pgconv.PgNumericToFloat64(product.CostPrice),
		SalePrice:   pgconv.PgNumericToFloat64(product.SalePrice),
		UpdatedBy:   pgconv.PgUUIDToUUID(product.UpdatedBy),
		UpdatedAt:   pgconv.PgTimestamptzToTime(product.UpdatedAt),
	}, nil
}

func (s *Service) DecrementStock(ctx context.Context, req domain.DecrementStockRequest) error {
	if err := s.repo.DecrementStock(ctx, db.DecrementStockParams{
		ID:       pgconv.ParseUUIDToPgType(req.ID),
		Quantity: req.Quantity,
	}); err != nil {
		return err
	}
	return nil
}

func (s *Service) CountProducts(ctx context.Context, companyId uuid.UUID) (int64, error) {
	count, err := s.repo.CountProducts(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) GetProductsPerformanceSummary(ctx context.Context, companyId uuid.UUID) (float64, error) {
	res, err := s.repo.GetProductsPerformanceSummary(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return 0, err
	}

	var percentage float64

	if res.LastMonthQty > 0 {
		percentage = ((res.CurrentMonthQty - res.LastMonthQty) / res.LastMonthQty) * 100
	} else {
		if res.CurrentMonthQty > 0 {
			percentage = 100.0
		} else {
			percentage = 0.0
		}
	}

	return percentage, nil
}

func (s *Service) GetCostTotalStock(ctx context.Context, companyId uuid.UUID) (float64, error) {
	total, err := s.repo.GetCostTotalStock(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (s *Service) GetTop5BestSellingProducts(ctx context.Context, companyId uuid.UUID) ([]domain.GetTop5BestSellingProductsRow, error) {
	products, err := s.repo.GetTop5BestSellingProducts(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.GetTop5BestSellingProductsRow{}, err
	}

	var response []domain.GetTop5BestSellingProductsRow

	for _, product := range products {
		response = append(response, domain.GetTop5BestSellingProductsRow{
			ID:                pgconv.PgUUIDToUUID(product.ID),
			Name:              product.Name,
			TotalQuantitySold: product.TotalQuantitySold,
		})
	}

	return response, nil
}

func (s *Service) GetInventoryReport(ctx context.Context, companyId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.GetInventoryReportResponse, error) {
	inventory, err := s.repo.GetInventoryReport(ctx, db.GetInventoryReportParams{
		CompanyID:   pgconv.ParseUUIDToPgType(companyId),
		CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
		CreatedAt_2: pgconv.TimeToPgTimestamptz(startAt_2),
	})
	if err != nil {
		return []domain.GetInventoryReportResponse{}, err
	}

	var response []domain.GetInventoryReportResponse

	for _, product := range inventory {
		response = append(response, domain.GetInventoryReportResponse{
			Name:         product.Name,
			CategoryName: pgconv.ParsePgTextToString(product.CategoryName),
			Quantity:     product.Quantity,
			SalePrice:    pgconv.PgNumericToFloat64(product.SalePrice),
			TotalValue:   pgconv.PgNumericToFloat64(product.TotalValue),
			CostPrice:    pgconv.PgNumericToFloat64(product.CostPrice),
			Barcode:      pgconv.ParsePgTextToString(product.Barcode),
			CreatedAt:    pgconv.PgTimestamptzToTime(product.CreatedAt),
		})
	}

	return response, nil
}

func (s *Service) ListProductsByDate(ctx context.Context, companyId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.ListProductsByDateResponse, error) {
	products, err := s.repo.ListProductsByDate(ctx, db.ListProductsByDateParams{
		CompanyID:   pgconv.ParseUUIDToPgType(companyId),
		CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
		CreatedAt_2: pgconv.TimeToPgTimestamptz(startAt_2),
	})
	if err != nil {
		return []domain.ListProductsByDateResponse{}, err
	}

	var response []domain.ListProductsByDateResponse

	for _, product := range products {
		response = append(response, domain.ListProductsByDateResponse{
			ID:           pgconv.PgUUIDToUUID(product.ID),
			CategoryID:   pgconv.PgUUIDToUUID(product.CategoryID),
			Name:         product.Name,
			Quantity:     product.Quantity,
			CostPrice:    pgconv.PgNumericToFloat64(product.CostPrice),
			CreatedAt:    pgconv.PgTimestamptzToTime(product.CreatedAt),
			CategoryName: product.CategoryName,
		})
	}

	return response, nil
}

func (s *Service) ListProductBuCategoryIdAndDate(ctx context.Context, categoryId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.ListProductsByCategoryAndDateResponse, error) {
	products, err := s.repo.ListProductBuCategoryIdAndDate(ctx, db.ListProductsByCategoryAndDateParams{
		CategoryID:  pgconv.ParseUUIDToPgType(categoryId),
		CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
		CreatedAt_2: pgconv.TimeToPgTimestamptz(startAt_2),
	})
	if err != nil {
		return []domain.ListProductsByCategoryAndDateResponse{}, err
	}

	var response []domain.ListProductsByCategoryAndDateResponse

	for _, product := range products {
		response = append(response, domain.ListProductsByCategoryAndDateResponse{
			ID:           pgconv.PgUUIDToUUID(product.ID),
			Name:         product.Name,
			CostPrice:    pgconv.PgNumericToFloat64(product.CostPrice),
			Quantity:     product.Quantity,
			CategoryID:   pgconv.PgUUIDToUUID(product.CategoryID),
			CreatedAt:    pgconv.PgTimestamptzToTime(product.CreatedAt),
			CategoryName: product.CategoryName,
		})
	}

	return response, nil
}
