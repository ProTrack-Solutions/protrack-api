package service

import (
	"context"
	"errors"
	"time"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	productRepo "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/products/repository"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/sale_items/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/sale_items/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateSaleItem(ctx context.Context, arg db.CreateSaleItemParams) error
	DeleteItemsBySale(ctx context.Context, saleId pgtype.UUID) error
	DeleteSaleItem(ctx context.Context, id pgtype.UUID) error
	ListItemsFromPendingSale(ctx context.Context, saleID pgtype.UUID) ([]db.ListItemsFromPendingSaleRow, error)
	ListItemsByCompany(ctx context.Context, companyId pgtype.UUID) ([]db.ListItemsByCompanyRow, error)
	ListItemsByDate(ctx context.Context, arg db.ListItemsByDateParams) ([]db.ListItemsByDateRow, error)
	WithTx(tx db.DBTX) *repository.Repository
}

type Service struct {
	repo        RepositoryInterface
	pool        *pgxpool.Pool
	productRepo *productRepo.Repository
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool, productRepo *productRepo.Repository) *Service {
	return &Service{
		repo:        repo,
		pool:        pool,
		productRepo: productRepo,
	}
}

// CreateSaleItemInTx cria um item de venda dentro de uma transação existente.
// Usado pelo CreateSale para garantir atomicidade (sale + items + decremento de estoque).
func (s *Service) CreateSaleItemInTx(ctx context.Context, tx db.DBTX, req domain.CreateSaleItemRequest, companyID uuid.UUID) error {
	txRepo := s.repo.WithTx(tx)
	txProductRepo := s.productRepo.WithTx(tx)

	product, err := txProductRepo.GetProductById(ctx, pgconv.ParseUUIDToPgType(req.ProductID))
	if err != nil {
		return err
	}

	// Garante que o produto pertence à company da venda
	if pgconv.PgUUIDToUUID(product.CompanyID) != companyID {
		return errors.New("The product does not belong to the company selling it.")
	}

	if product.Quantity < req.Quantity {
		return errors.New("insufficient quantity")
	}

	if err := txProductRepo.DecrementStock(ctx, db.DecrementStockParams{
		ID:       pgconv.ParseUUIDToPgType(req.ProductID),
		Quantity: req.Quantity,
	}); err != nil {
		return err
	}

	// Verifica se o decremento realmente ocorreu (evita falha silenciosa em condição de corrida)
	productAfter, err := txProductRepo.GetProductById(ctx, pgconv.ParseUUIDToPgType(req.ProductID))
	if err != nil {
		return err
	}
	if productAfter.Quantity != product.Quantity-req.Quantity {
		return errors.New("insufficient quantity")
	}

	return txRepo.CreateSaleItem(ctx, db.CreateSaleItemParams{
		SaleID:    pgconv.ParseUUIDToPgType(req.SaleID),
		ProductID: pgconv.ParseUUIDToPgType(req.ProductID),
		Quantity:  req.Quantity,
		UnitPrice: pgconv.Float64ToPgNumeric(req.UnitPrice),
		Discount:  pgconv.Float64ToPgNumeric(req.Discount),
	})
}

func (s *Service) DeleteItemsBySale(ctx context.Context, saleId uuid.UUID) error {
	if err := s.repo.DeleteItemsBySale(ctx, pgconv.ParseUUIDToPgType(saleId)); err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteSaleItem(ctx context.Context, Id uuid.UUID) error {
	if err := s.repo.DeleteSaleItem(ctx, pgconv.ParseUUIDToPgType(Id)); err != nil {
		return err
	}
	return nil
}

func (s *Service) ListItemsFromPendingSale(ctx context.Context, saleId uuid.UUID) ([]domain.ListItemsFromPendingSaleRow, error) {
	items, err := s.repo.ListItemsFromPendingSale(ctx, pgconv.ParseUUIDToPgType(saleId))
	if err != nil {
		return []domain.ListItemsFromPendingSaleRow{}, err
	}

	var response []domain.ListItemsFromPendingSaleRow

	for _, item := range items {
		response = append(response, domain.ListItemsFromPendingSaleRow{
			ID:          pgconv.PgUUIDToUUID(item.ID),
			SaleID:      pgconv.PgUUIDToUUID(item.SaleID),
			ProductID:   pgconv.PgUUIDToUUID(item.ProductID),
			Quantity:    item.Quantity,
			UnitPrice:   pgconv.PgNumericToFloat64(item.UnitPrice),
			Discount:    pgconv.PgNumericToFloat64(item.Discount),
			ProductName: item.ProductName,
		})
	}

	return response, nil
}

func (s *Service) ListItemsByCompany(ctx context.Context, companyId uuid.UUID) ([]domain.ListItemsByCompanyResponse, error) {
	items, err := s.repo.ListItemsByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListItemsByCompanyResponse{}, err
	}

	var response []domain.ListItemsByCompanyResponse

	for _, item := range items {
		response = append(response, domain.ListItemsByCompanyResponse{
			ID:          pgconv.PgUUIDToUUID(item.ID),
			SaleID:      pgconv.PgUUIDToUUID(item.SaleID),
			ProductID:   pgconv.PgUUIDToUUID(item.ProductID),
			Quantity:    item.Quantity,
			UnitPrice:   pgconv.PgNumericToFloat64(item.UnitPrice),
			Discount:    pgconv.PgNumericToFloat64(item.Discount),
			ProductName: item.ProductName,
		})
	}

	return response, nil
}

func (s *Service) ListItemsByDate(ctx context.Context, companyId uuid.UUID, date time.Time) ([]domain.ListItemsByDateResponse, error) {
	items, err := s.repo.ListItemsByDate(ctx, db.ListItemsByDateParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Column2:   pgconv.TimeToPgTimestamptz(date),
	})
	if err != nil {
		return []domain.ListItemsByDateResponse{}, err
	}

	var response []domain.ListItemsByDateResponse

	for _, item := range items {
		response = append(response, domain.ListItemsByDateResponse{
			ID:          pgconv.PgUUIDToUUID(item.ID),
			SaleID:      pgconv.PgUUIDToUUID(item.SaleID),
			ProductID:   pgconv.PgUUIDToUUID(item.ProductID),
			Quantity:    item.Quantity,
			UnitPrice:   pgconv.PgNumericToFloat64(item.UnitPrice),
			Discount:    pgconv.PgNumericToFloat64(item.Discount),
			ProductName: item.ProductName,
		})
	}

	return response, nil
}
