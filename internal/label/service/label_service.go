package service

import (
	"bytes"
	"context"

	"github.com/ProTrack-Solutions/protrack-api/internal/label/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/pkg/generator"
	productService "github.com/ProTrack-Solutions/protrack-api/internal/products/service"
	"github.com/google/uuid"
)

type Service struct {
	productService *productService.Service
}

func NewService(productService *productService.Service) *Service {
	return &Service{
		productService: productService,
	}
}

// GenerateProductsLabelPDF só imprime produtos da empresa do usuário
func (s *Service) GenerateProductsLabelPDF(ctx context.Context, companyId uuid.UUID, productIDs []domain.GenetareTagProductRequest) (*bytes.Buffer, error) {
	var products []domain.GenetareTagProduct

	for _, productId := range productIDs {

		product, err := s.productService.GetCompanyProductById(ctx, companyId, productId.ProductId)
		if err != nil {
			return &bytes.Buffer{}, err
		}

		copies := int(product.Quantity)
		if productId.Quantity > 0 {
			copies = int(productId.Quantity)
		}
		if len(products)+copies > domain.MaxLabelsPerSheetRequest {
			return &bytes.Buffer{}, domain.ErrTooManyLabels
		}

		label := domain.GenetareTagProduct{
			Name:    product.Name,
			Amount:  product.SalePrice,
			Barcode: product.Barcode,
		}
		if product.ModelName != "" {
			label.Name = product.ModelName
			label.Color = product.Color
			label.Size = product.Size
		}

		for i := 0; i < copies; i++ {
			products = append(products, label)
		}

	}

	layout := domain.Layout5x13

	// Chama a função que retorna o buffer em memória
	return generator.GenerateLabelSheetPDF(products, layout)
}
