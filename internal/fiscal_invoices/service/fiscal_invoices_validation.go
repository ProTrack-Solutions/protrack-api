package service

import (
	"context"

	companiesDomain "github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal_invoices/domain"
	saleItemsDomain "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/domain"
	"github.com/google/uuid"
)

// validateFiscalFields confere se a empresa e cada produto vendido têm os
// campos fiscais mínimos preenchidos para emissão. CFOP não entra aqui
// porque tem fallback de mercado (5102/6102) — só NCM, CSOSN (produto) e
// Inscrição Estadual/Regime Tributário (empresa) bloqueiam a emissão.
func (s *Service) validateFiscalFields(
	ctx context.Context,
	company companiesDomain.CompanyResponse,
	items []saleItemsDomain.SaleItemRow,
) domain.MissingFiscalFields {
	var missing domain.MissingFiscalFields
	missing.ProductFields = make(map[uuid.UUID][]string)

	if company.RegimeTributario == nil {
		missing.CompanyFields = append(missing.CompanyFields, "regime_tributario")
	}
	if !company.InscricaoEstadualIsento && company.InscricaoEstadual == "" {
		missing.CompanyFields = append(missing.CompanyFields, "inscricao_estadual")
	}

	for _, item := range items {
		product, err := s.productsService.GetProductById(ctx, item.ProductID)
		if err != nil {
			missing.ProductFields[item.ProductID] = []string{"produto não encontrado"}
			continue
		}

		var productMissing []string
		if product.Ncm == "" {
			productMissing = append(productMissing, "ncm")
		}
		if product.Csosn == "" {
			productMissing = append(productMissing, "csosn")
		}
		if len(productMissing) > 0 {
			missing.ProductFields[item.ProductID] = productMissing
		}
	}

	return missing
}
