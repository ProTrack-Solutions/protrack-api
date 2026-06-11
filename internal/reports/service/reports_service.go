package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	analyticsService "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/analytics/service"
	paymentHistoryService "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/payment_history/service"
	productService "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/products/service"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/reports"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/reports/domain"
	saleService "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/sales/service"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Service struct {
	saleService           *saleService.Service
	analyticsService      *analyticsService.Service
	paymentHistoryService *paymentHistoryService.Service
	productService        *productService.Service
}

func NewService(saleService *saleService.Service, analyticsService *analyticsService.Service, paymentHistoryService *paymentHistoryService.Service, productService *productService.Service) *Service {
	return &Service{
		saleService:           saleService,
		analyticsService:      analyticsService,
		paymentHistoryService: paymentHistoryService,
		productService:        productService,
	}
}

func (s *Service) GenerateReports(ctx context.Context, reportType string, companyId uuid.UUID, At time.Time, At2 time.Time) (domain.ReportResponse, error) {
	var response domain.ReportResponse
	var headers []string
	var rows [][]any
	var fileName string
	switch reportType {
	case string(domain.ReportSales):
		report, err := s.saleService.GetPendingSalesDetailedReport(ctx, companyId, At, At2)
		if err != nil {
			return domain.ReportResponse{}, errors.New("erro ao gerar relatório de vendas pendentes")
		}

		headers = []string{
			"Data da Venda",
			"Subtotal",
			"Desconto",
			"Total",
			"Parcelas",
			"Método de Pagamento",
			"Status da Venda",
			"Nome do Cliente",
			"Quantidade",
			"Preço Unitário",
			"Desconto do Item",
			"Nome do Produto",
			"Total da Parcela",
			"Saldo da Parcela",
			"Data de Vencimento",
			"Número da Parcela",
			"Status da Parcela",
		}
		rows = [][]any{}
		for _, row := range report {
			rows = append(rows, []any{
				row.SaleAt.Format("02/01/2006"),
				row.Subtotal,
				row.DiscountAmount,
				row.TotalAmount,
				row.InstallmentsCount,
				row.PaymentMethod,
				row.SaleStatus,
				row.CustomerName,
				row.Quantity,
				row.UnitPrice,
				row.ItemDiscount,
				row.ProductName,
				row.InstallmentTotalAmount,
				row.InstallmentBalance,
				row.DueDate,
				row.InstallmentNumber,
				row.InstallmentStatus,
			})
		}

		rows = reports.AddRow(rows, "total sale:", 1)
		fileName = fmt.Sprintf("sales_%s.xlsx", At.Format("02-01-2006"))
		response = domain.ReportResponse{
			Headers:  headers,
			Rows:     rows,
			FileName: fileName,
		}
		return response, nil
	case string(domain.ReportPayments):
		history, err := s.paymentHistoryService.GetPaymentsHistoryReport(ctx, companyId, At, At2)
		if err != nil {
			return domain.ReportResponse{}, err
		}

		headers, rows = domain.MapStructToReport(history)

		rows = reports.AddRow(rows, "TOTAL PAYMENTS:", 0)

		fileName = fmt.Sprintf("paymentHistory_%s.xlsx", At.Format("02-01-2006"))

		return domain.ReportResponse{
			Headers:  headers,
			Rows:     rows,
			FileName: fileName,
		}, nil

	case string(domain.ReportInventory):
		inventory, err := s.productService.GetInventoryReport(ctx, companyId, At, At2)
		if err != nil {
			return domain.ReportResponse{}, err
		}

		log.Info().Int("quantidade_registros", len(inventory)).Msg("Dados do banco")

		headers, rows = domain.MapStructToReport(inventory)

		rows = reports.AddRow(rows, "TOTAL STOCK:", 4)

		fileName = fmt.Sprintf("inventory_%s.xlsx", time.Now().Format("02-01-2006"))

		return domain.ReportResponse{
			Headers:  headers,
			Rows:     rows,
			FileName: fileName,
		}, nil
	case string(domain.ReportProfitProduct):
		products, err := s.analyticsService.ProfitMarginProducts(ctx, companyId, At, At2)
		if err != nil {
			return domain.ReportResponse{}, err
		}

		log.Info().Interface("product", products).Msg("product")
		headers, rows = domain.MapStructToReport(products)

		fileName = fmt.Sprintf("profit_products_%s.xlsx", time.Now().Format("02-01-2006"))

		return domain.ReportResponse{
			Headers:  headers,
			Rows:     rows,
			FileName: fileName,
		}, err
	case string(domain.ReportProfitCategory):
		categories, err := s.analyticsService.ProfitMarginCategoryId(ctx, companyId, At, At2)
		if err != nil {
			return domain.ReportResponse{}, err
		}

		headers, rows := domain.MapStructToReport(categories)

		fileName = fmt.Sprintf("profit_category_%s.xlsx", time.Now().Format("02-01-2006"))

		return domain.ReportResponse{
			Headers:  headers,
			Rows:     rows,
			FileName: fileName,
		}, nil
	}

	return domain.ReportResponse{}, errors.New("relatório não encontrado")
}
