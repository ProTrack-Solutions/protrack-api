package service

import (
	"bytes"
	"context"

	"github.com/ProTrack-Solutions/protrack-api/internal/label/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/pkg/generator"
	productService "github.com/ProTrack-Solutions/protrack-api/internal/products/service"
)

type Service struct {
	productService *productService.Service
}

func NewService(productService *productService.Service) *Service {
	return &Service{
		productService: productService,
	}
}

func (s *Service) GenerateProductsLabelPDF(ctx context.Context, productIDs []domain.GenetareTagProductRequest) (*bytes.Buffer, error) {
	var products []domain.GenetareTagProduct

	for _, productId := range productIDs {

		product, err := s.productService.GetProductById(ctx, productId.ProductId)
		if err != nil {
			return &bytes.Buffer{}, err
		}

		for i := 0; i < int(product.Quantity); i++ {
			products = append(products, domain.GenetareTagProduct{
				Name:    product.Name,
				Amount:  product.SalePrice,
				Barcode: product.Barcode,
			})
		}

	}

	layout := domain.Layout5x13

	// Chama a função que retorna o buffer em memória
	return generator.GenerateLabelSheetPDF(products, layout)
}
