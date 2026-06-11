package service

import (
	"context"
	"time"

	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/analytics/domain"
	productService "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/products/service"
	productCategoriesService "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/products_categories/service"
	saleItemsService "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/sale_items/service"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Service struct {
	productService           *productService.Service
	saleItemsService         *saleItemsService.Service
	productCategoriesService *productCategoriesService.Service
}

func NewService(productService *productService.Service, saleItemsService *saleItemsService.Service) *Service {
	return &Service{
		productService:   productService,
		saleItemsService: saleItemsService,
	}
}

func (s *Service) ProfitMarginProducts(ctx context.Context, companyId uuid.UUID, startAt, startAt_2 time.Time) ([]domain.ProfitMarginProductsResponse, error) {
	log.Info().Time("startAt", startAt).Msg("startAt")
	log.Info().Time("startAt_2", startAt_2).Msg("startAt_2")
	products, err := s.productService.ListProductsByDate(ctx, companyId, startAt, startAt_2)
	if err != nil {
		return []domain.ProfitMarginProductsResponse{}, err
	}

	log.Info().Interface("product", products).Msg("product")

	productItems, err := s.saleItemsService.ListItemsByCompany(ctx, companyId)
	if err != nil {
		return []domain.ProfitMarginProductsResponse{}, err
	}

	var response []domain.ProfitMarginProductsResponse

	for _, product := range products {
		var totalCost float64
		var totalNetSales float64
		var salePrice float64
		found := false

		for _, item := range productItems {
			if item.ProductID == product.ID {
				totalNetSales += (float64(item.UnitPrice) - float64(item.Discount))
				totalCost += product.CostPrice * float64(item.Quantity)
				salePrice = item.UnitPrice
				found = true
			}
		}

		if found && totalNetSales > 0 {
			margin := ((totalNetSales - float64(totalCost)) / totalNetSales) * 100
			response = append(response, domain.ProfitMarginProductsResponse{
				Name:      product.Name,
				CostPrice: product.CostPrice,
				SalePrice: salePrice,
				Profit:    margin,
			})
		}
	}

	return response, nil
}

func (s *Service) ProfitMarginCategoryId(ctx context.Context, companyId uuid.UUID, startAt, startAt_2 time.Time) ([]domain.ProfitMarginCategoryResponse, error) {
	categoroies, err := s.productCategoriesService.ListProductCategoryByCompanyId(ctx, companyId)
	if err != nil {
		return []domain.ProfitMarginCategoryResponse{}, err
	}

	var response []domain.ProfitMarginCategoryResponse

	for _, category := range categoroies {
		var totalCost float64
		var totalSale float64
		products, err := s.productService.ListProductBuCategoryIdAndDate(ctx, category.ID, startAt, startAt_2)
		if err != nil {
			return []domain.ProfitMarginCategoryResponse{}, err
		}

		productItems, err := s.saleItemsService.ListItemsByCompany(ctx, companyId)
		if err != nil {
			return []domain.ProfitMarginCategoryResponse{}, err
		}

		for _, product := range products {
			var soldQuantity int

			soldQuantity += int(product.Quantity)

			for _, item := range productItems {
				if item.ProductID == product.ID {
					soldQuantity += int(item.Quantity)
					totalSale += item.UnitPrice * float64(item.Quantity)
				}
			}

			totalCost += product.CostPrice * float64(soldQuantity)

		}

		profit := ((totalSale - totalCost) / totalSale) * 100

		response = append(response, domain.ProfitMarginCategoryResponse{
			Name:      category.Name,
			TotalCost: totalCost,
			TotalSale: totalSale,
			Profit:    profit,
		})
	}

	return response, nil
}

func (s *Service) CalculateMarginDistribution(ctx context.Context, companyId uuid.UUID, startAt, startAt_2 time.Time) ([]domain.CalculateMarginDistributionResponse, error) {
	products, err := s.productService.ListProductsByDate(ctx, companyId, startAt, startAt_2)
	if err != nil {
		return []domain.CalculateMarginDistributionResponse{}, err
	}

	productItems, err := s.saleItemsService.ListItemsByCompany(ctx, companyId)
	if err != nil {
		return []domain.CalculateMarginDistributionResponse{}, err
	}

	var response []domain.CalculateMarginDistributionResponse

	for _, product := range products {
		var soldQuantity int
		var totalInvestment float64

		soldQuantity += int(product.Quantity)

		for _, item := range productItems {
			if product.ID == item.ProductID {
				soldQuantity += int(item.Quantity)
			}
		}

		totalInvestment = product.CostPrice + float64(soldQuantity)

		response = append(response, domain.CalculateMarginDistributionResponse{
			ProductName:     product.Name,
			Quantity:        int64(soldQuantity),
			CostPrice:       product.CostPrice,
			TotalInvestment: totalInvestment,
		})
	}

	return response, nil
}
