package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	accountsReceivableDomain "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/domain"
	accountsReceivableService "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/service"
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	companiesService "github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	customerDomain "github.com/ProTrack-Solutions/protrack-api/internal/customers/domain"
	customerService "github.com/ProTrack-Solutions/protrack-api/internal/customers/service"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	productService "github.com/ProTrack-Solutions/protrack-api/internal/products/service"
	productCategoriesService "github.com/ProTrack-Solutions/protrack-api/internal/products_categories/service"
	saleItemDomain "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/domain"
	saleItemsService "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/repository"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/events"
	"github.com/ProTrack-Solutions/protrack-api/internal/whatsapp"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type RepositoryInterface interface {
	CreateSales(ctx context.Context, arg db.CreateSaleParams) (pgtype.UUID, error)
	DeleteSales(ctx context.Context, arg db.DeleteSaleParams) error
	GetSaleById(ctx context.Context, arg db.GetSaleByIdParams) (db.GetSaleByIdRow, error)
	ListSales(ctx context.Context, companyId pgtype.UUID) ([]db.ListSalesRow, error)
	UpdateSaleStatus(ctx context.Context, arg db.UpdateSaleStatusParams) error
	ListSalesByCompanyAndStatus(ctx context.Context, arg db.ListSalesByCompanyAndStatusParams) ([]db.ListSalesByCompanyAndStatusRow, error)
	CountSales(ctx context.Context, companyId pgtype.UUID) (int64, error)
	GetSalesPerformanceSummary(ctx context.Context, companyId pgtype.UUID) (db.GetSalesPerformanceSummaryRow, error)
	GetTotalAmountSummary(ctx context.Context, companyId pgtype.UUID) (db.GetTotalAmountSummaryRow, error)
	GetTotalAmountByStatus(ctx context.Context, arg db.GetTotalAmountByStatusParams) (float64, error)
	GetSaleByIdWhatsapp(ctx context.Context, id pgtype.UUID) (db.GetSaleByIdWhatsappRow, error)
	UpdateOverdueSalesAndAccounts(ctx context.Context) ([]db.UpdateOverdueSalesAndAccountsGlobalRow, error)
	GetSaleByIdJust(ctx context.Context, saleId pgtype.UUID) (db.GetSaleByIdJustRow, error)
	ContSalesPendingAndOverdue(ctx context.Context, companyId pgtype.UUID) (int64, error)
	ListSalesWithDetails(ctx context.Context, companyID pgtype.UUID) ([]db.ListSalesWithDetailsRow, error)
	ListSalesWithDetailsPendingOverdue(ctx context.Context, companyID pgtype.UUID) ([]db.ListSalesWithDetailsPendingOverdueRow, error)
	GetPendingSalesDetailedReport(ctx context.Context, arg db.GetPendingSalesDetailedReportParams) ([]db.GetPendingSalesDetailedReportRow, error)
	ListSalesWithDetailsPaginate(ctx context.Context, arg db.ListSalesWithDetailsPaginateParams) ([]db.ListSalesWithDetailsPaginateRow, error)
	CountSalesByCompany(ctx context.Context, companyId pgtype.UUID) (int64, error)
	UpdateSale(ctx context.Context, arg db.UpdateSaleParams) error
	WithTx(tx db.DBTX) *repository.Repository
}

type Service struct {
	repo                      RepositoryInterface
	pool                      *pgxpool.Pool
	saleItemsService          *saleItemsService.Service
	customerService           *customerService.Service
	accountsReceivableService *accountsReceivableService.Service
	productService            *productService.Service
	productCategoriesService  *productCategoriesService.Service
	companiesService          *companiesService.Service
	whatsApp                  *whatsapp.Whatsapp
}

func NewService(
	repo *repository.Repository,
	pool *pgxpool.Pool,
	saleItemsService *saleItemsService.Service,
	customerService *customerService.Service,
	accountsReceivableService *accountsReceivableService.Service,
	productService *productService.Service,
	productCategoriesService *productCategoriesService.Service,
	companiesService *companiesService.Service,
	whatsApp *whatsapp.Whatsapp,
) *Service {
	return &Service{
		repo:                      repo,
		pool:                      pool,
		saleItemsService:          saleItemsService,
		customerService:           customerService,
		accountsReceivableService: accountsReceivableService,
		productService:            productService,
		productCategoriesService:  productCategoriesService,
		companiesService:          companiesService,
		whatsApp:                  whatsApp,
	}
}

func (s *Service) CreateSale(ctx context.Context, userId, companyId uuid.UUID, req domain.CreateSaleRequest) (uuid.UUID, error) {
	if err := domain.ValidateCreateSaleRequest(req); err != nil {
		return uuid.Nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	log.Info().Interface("request", req).Msg("Teste request")

	var status string
	var subTotal float64
	var totalAmount float64
	var discount float64

	for _, item := range req.Items {
		product, err := s.productService.GetProductByIdTx(ctx, tx, item.ProductID)
		if err != nil {
			return uuid.Nil, err
		}

		subTotal += float64(item.Quantity) * product.SalePrice
	}

	totalAmount = subTotal * (1 - (req.DiscountAmount / 100))

	// 1. Definição do Status e Atualização de Saldo Devedor
	if req.PaymentMethod == "installments" {
		if err := s.customerService.UpdateCustomerBalanceAddTx(ctx, tx, req.CustomerID, customerDomain.UpdateBalanceDueCustomerRequest{
			BalanceDue: subTotal,
			Prohibited: req.Prohibited,
			UpdatedBy:  userId,
		}); err != nil {
			return uuid.Nil, err
		}
		status = "pending"
	} else {
		status = "paid"
	}

	dueDaysVal := 0
	if req.PaymentMethod == "installments" && req.DueDays > 0 {
		dueDaysVal = int(req.DueDays)
	}

	var installments int32
	if req.PaymentMethod == "installments" {
		installments = req.InstallmentsCount
	}

	id, err := txRepo.CreateSales(ctx, db.CreateSaleParams{
		CustomerID:        pgconv.ParseUUIDToPgType(req.CustomerID),
		CompanyID:         pgconv.ParseUUIDToPgType(companyId),
		DiscountAmount:    pgconv.Float64ToPgNumeric(req.DiscountAmount),
		Subtotal:          pgconv.Float64ToPgNumeric(subTotal),
		TotalAmount:       pgconv.Float64ToPgNumeric(totalAmount),
		DueDays:           pgconv.OptionalIntToPgInt4(dueDaysVal),
		PaymentMethod:     req.PaymentMethod,
		Status:            status,
		CreatedBy:         pgconv.ParseUUIDToPgType(userId),
		InstallmentsCount: installments,
		DownPayment:       pgconv.Float64ToPgNumeric(req.Prohibited),
	})
	if err != nil {
		return uuid.Nil, err
	}

	if req.PaymentMethod == "installments" {

		amountToParcel := totalAmount - req.Prohibited
		installmentValue := amountToParcel / float64(req.InstallmentsCount)
		dataBase := time.Now()

		for i := 0; i < int(req.InstallmentsCount); i++ {
			var maturity time.Time

			if dataBase.Day() >= int(req.DueDays) {
				maturity = time.Date(
					dataBase.Year(),
					dataBase.Month()+time.Month(i+1),
					int(req.DueDays),
					0, 0, 0, 0,
					dataBase.Location(),
				)
			} else {
				maturity = time.Date(
					dataBase.Year(),
					dataBase.Month()+time.Month(i),
					int(req.DueDays),
					0, 0, 0, 0,
					dataBase.Location(),
				)
			}

			var reqAR accountsReceivableDomain.CreateAccountReceivableRequest
			reqAR.CustomerID = req.CustomerID
			reqAR.SaleID = pgconv.PgUUIDToUUID(id)

			reqAR.Balance = installmentValue
			reqAR.TotalAmount = installmentValue

			reqAR.InstallmentNumber = int64(i + 1)
			reqAR.TotalInstallments = int64(req.InstallmentsCount)
			reqAR.DueDate = maturity.Format("2006-01-02")

			if err := s.accountsReceivableService.CreateAccountReceivableInTx(ctx, tx, userId, companyId, reqAR); err != nil {
				return uuid.Nil, err
			}
		}
	}

	for _, itemReq := range req.Items {

		product, err := s.productService.GetProductByIdTx(ctx, tx, itemReq.ProductID)
		if err != nil {
			return uuid.Nil, err
		}

		saleID := pgconv.PgUUIDToUUID(id)

		percentageItem := (product.SalePrice / totalAmount) * 100

		discount = req.DiscountAmount * (percentageItem / 100)

		if err := s.saleItemsService.CreateSaleItemInTx(ctx, tx, saleItemDomain.CreateSaleItemRequest{
			SaleID:    saleID,
			ProductID: itemReq.ProductID,
			Quantity:  itemReq.Quantity,
			UnitPrice: product.SalePrice,
			Discount:  discount,
		}, companyId); err != nil {
			return uuid.Nil, err
		}
	}

	return pgconv.PgUUIDToUUID(id), tx.Commit(ctx)
}

func (s *Service) DeleteSale(ctx context.Context, id uuid.UUID, req domain.DeleteSaleRequest) error {
	if err := s.repo.DeleteSales(ctx, db.DeleteSaleParams{
		DeletedBy: pgconv.ParseUUIDToPgType(req.DeletedBy),
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetSaleById(ctx context.Context, req domain.GetSaleByIdRequest) (domain.GetSaleByIdRow, error) {
	sale, err := s.repo.GetSaleById(ctx, db.GetSaleByIdParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
	})
	if err != nil {
		return domain.GetSaleByIdRow{}, err
	}

	return domain.GetSaleByIdRow{
		ID:             pgconv.PgUUIDToUUID(sale.ID),
		CustomerID:     pgconv.PgUUIDToUUID(sale.CustomerID),
		CompanyID:      pgconv.PgUUIDToUUID(sale.CompanyID),
		SaleAt:         pgconv.PgTimestamptzToTime(sale.SaleAt),
		DiscountAmount: pgconv.PgNumericToFloat64(sale.DiscountAmount),
		Subtotal:       pgconv.PgNumericToFloat64(sale.Subtotal),
		TotalAmount:    pgconv.PgNumericToFloat64(sale.TotalAmount),
		DueDays:        int32(pgconv.PgInt4ToInt(sale.DueDays)),
		PaymentMethod:  sale.PaymentMethod,
		Status:         sale.Status,
		CreatedAt:      pgconv.PgTimestamptzToTime(sale.CreatedAt),
		CreatedBy:      pgconv.PgUUIDToUUID(sale.CreatedBy),
		UpdatedAt:      pgconv.PgTimestamptzToTime(sale.UpdatedAt),
		UpdatedBy:      pgconv.PgUUIDToUUID(sale.UpdatedBy),
		DeletedAt:      pgconv.PgTimestamptzToTime(sale.DeletedAt),
		DeletedBy:      pgconv.PgUUIDToUUID(sale.DeletedBy),
		CustomerName:   sale.CustomerName,
	}, nil
}

func (s *Service) GetSaleByIdTx(ctx context.Context, tx db.DBTX, id, companyId uuid.UUID) (domain.GetSaleByIdRow, error) {
	repoTx := db.New(tx)

	sale, err := repoTx.GetSaleById(ctx, db.GetSaleByIdParams{
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if err != nil {
		return domain.GetSaleByIdRow{}, err
	}

	return domain.GetSaleByIdRow{
		ID:             pgconv.PgUUIDToUUID(sale.ID),
		CustomerID:     pgconv.PgUUIDToUUID(sale.CustomerID),
		CompanyID:      pgconv.PgUUIDToUUID(sale.CompanyID),
		SaleAt:         pgconv.PgTimestamptzToTime(sale.SaleAt),
		DiscountAmount: pgconv.PgNumericToFloat64(sale.DiscountAmount),
		Subtotal:       pgconv.PgNumericToFloat64(sale.Subtotal),
		TotalAmount:    pgconv.PgNumericToFloat64(sale.TotalAmount),
		DueDays:        int32(pgconv.PgInt4ToInt(sale.DueDays)),
		PaymentMethod:  sale.PaymentMethod,
		Status:         sale.Status,
		CreatedAt:      pgconv.PgTimestamptzToTime(sale.CreatedAt),
		CreatedBy:      pgconv.PgUUIDToUUID(sale.CreatedBy),
		UpdatedAt:      pgconv.PgTimestamptzToTime(sale.UpdatedAt),
		UpdatedBy:      pgconv.PgUUIDToUUID(sale.UpdatedBy),
		DeletedAt:      pgconv.PgTimestamptzToTime(sale.DeletedAt),
		DeletedBy:      pgconv.PgUUIDToUUID(sale.DeletedBy),
		CustomerName:   sale.CustomerName,
	}, nil
}

func (s *Service) ListSales(ctx context.Context, companyId uuid.UUID) ([]domain.ListSalesRow, error) {
	sales, err := s.repo.ListSales(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListSalesRow{}, err
	}

	var response []domain.ListSalesRow

	for _, sale := range sales {
		response = append(response, domain.ListSalesRow{
			ID:          pgconv.PgUUIDToUUID(sale.ID),
			SaleAt:      pgconv.PgTimestamptzToTime(sale.SaleAt),
			TotalAmount: pgconv.PgNumericToFloat64(sale.TotalAmount),
			Status:      sale.Status,
			CreatedAt:   pgconv.PgTimestamptzToTime(sale.CreatedAt),
		})
	}

	return response, nil
}

func (s *Service) UpdateSaleStatus(ctx context.Context, id uuid.UUID, req domain.UpdateSaleStatusRequest) error {
	sale, err := s.repo.GetSaleById(ctx, db.GetSaleByIdParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
	})
	if err != nil {
		return err
	}

	arg := db.UpdateSaleStatusParams{
		Status:    sale.Status,
		UpdatedBy: sale.UpdatedBy,
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: sale.CompanyID,
	}

	if req.Status != "" {
		arg.Status = req.Status
	}

	if err := s.repo.UpdateSaleStatus(ctx, arg); err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateSaleStatusTx(ctx context.Context, tx db.DBTX, id, company_id, userId uuid.UUID, status string) error {
	repoTx := db.New(tx)

	sale, err := repoTx.GetSaleById(ctx, db.GetSaleByIdParams{
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: pgconv.ParseUUIDToPgType(company_id),
	})
	if err != nil {
		return err
	}

	arg := db.UpdateSaleStatusParams{
		Status:    sale.Status,
		UpdatedBy: sale.UpdatedBy,
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: sale.CompanyID,
	}

	if status != "" {
		arg.Status = status
	}

	if err := repoTx.UpdateSaleStatus(ctx, arg); err != nil {
		return err
	}

	return nil
}

func (s *Service) ListSalesByCustomerAndStatus(ctx context.Context, req domain.ListSalesByCompanyAndStatusRequest) ([]domain.ListSalesByCompanyAndStatusRow, error) {
	sales, err := s.repo.ListSalesByCompanyAndStatus(ctx, db.ListSalesByCompanyAndStatusParams{
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		Column2:   req.Status,
	})
	if err != nil {
		return []domain.ListSalesByCompanyAndStatusRow{}, err
	}

	var response []domain.ListSalesByCompanyAndStatusRow

	for _, sale := range sales {
		response = append(response, domain.ListSalesByCompanyAndStatusRow{
			SaleID:         pgconv.PgUUIDToUUID(sale.SaleID),
			TotalAmount:    pgconv.PgNumericToFloat64(sale.TotalAmount),
			DiscountAmount: pgconv.PgNumericToFloat64(sale.DiscountAmount),
			Status:         sale.Status,
			SaleDate:       pgconv.PgTimestamptzToTime(sale.SaleDate),
			ItemID:         pgconv.PgUUIDToUUID(sale.ItemID),
			ProductID:      pgconv.PgUUIDToUUID(sale.ProductID),
			Quantity:       sale.Quantity,
			UnitPrice:      pgconv.PgNumericToFloat64(sale.UnitPrice),
			Discount:       pgconv.PgNumericToFloat64(sale.Discount),
			ProductName:    sale.ProductName,
			CustomerName:   sale.CustomerName,
		})
	}

	return response, nil
}

func (s *Service) CountSales(ctx context.Context, companyId uuid.UUID) (int64, error) {
	count, err := s.repo.CountSales(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Service) GetSalesPerformanceSummary(ctx context.Context, companyId uuid.UUID) (float64, error) {
	res, err := s.repo.GetSalesPerformanceSummary(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		log.Err(err).Msg("Debug para error")
		return 0, err
	}

	var percentage float64

	if res.LastMonthCount > 0 {
		percentage = (float64(res.CurrentMonthCount) - float64(res.LastMonthCount)) / float64(res.LastMonthCount) * 100
	} else {
		if res.CurrentMonthCount > 0 {
			percentage = 100.0
		} else {
			percentage = 0.0
		}
	}

	return percentage, nil
}

func (s *Service) GetTotalAmountSummary(ctx context.Context, companyId uuid.UUID) (domain.GetTotalAmountSummaryRow, error) {
	res, err := s.repo.GetTotalAmountSummary(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.GetTotalAmountSummaryRow{}, err
	}

	growthPercentage := (res.LastMonthSt / res.CurrentMonthSt) * 100

	return domain.GetTotalAmountSummaryRow{
		CurrentMonthSt:   res.CurrentMonthSt,
		LastMonthSt:      res.LastMonthSt,
		GrowthPercentage: math.Round(growthPercentage),
	}, nil
}

func (s *Service) GetTotalAmountIsPending(ctx context.Context, companyId uuid.UUID) (float64, error) {
	status := "pending"

	total, err := s.repo.GetTotalAmountByStatus(ctx, db.GetTotalAmountByStatusParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Status:    status,
	})
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (s *Service) GetTotalAmountIsOverdue(ctx context.Context, req domain.GetTotalAmountByStatusRequest) (float64, error) {
	req.Status = "overdue"

	total, err := s.repo.GetTotalAmountByStatus(ctx, db.GetTotalAmountByStatusParams{
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		Status:    req.Status,
	})
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (s *Service) UpdateOverdueSales(ctx context.Context) (domain.OverdueSalesResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.OverdueSalesResult{}, err
	}

	defer tx.Rollback(ctx)

	repoTx := s.repo.WithTx(tx)

	response, err := repoTx.UpdateOverdueSalesAndAccounts(ctx)
	if err != nil {
		return domain.OverdueSalesResult{}, err
	}

	var whatsAppEvents []events.WhatsApp
	// agrupa contagem de vendas vencidas por empresa
	companyCounts := make(map[uuid.UUID]int)

	for _, data := range response {
		customer, err := s.customerService.GetCustomerByIdTx(ctx, tx, pgconv.PgUUIDToUUID(data.CustomerID))
		if err != nil {
			return domain.OverdueSalesResult{}, err
		}

		sale, err := repoTx.GetSaleByIdJust(ctx, data.SaleID)
		if err != nil {
			log.Error().Err(err).Str("sale_id", data.SaleID.String()).Msg("Erro ao buscar venda para WhatsApp")
			continue
		}

		company, err := s.companiesService.GetCompanyByIDTx(ctx, tx, pgconv.PgUUIDToUUID(data.CompanyID))
		if err != nil {
			return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve company: %w", err)
		}

		companyID := pgconv.PgUUIDToUUID(data.CompanyID)
		instanceName := fmt.Sprintf("%s-%s", company.Name, companyID.String())

		msg := fmt.Sprintf("⚠️ *Aviso de Vencimento*\n\n"+
			"Informamos que a sua parcela com vencimento no dia %d venceu hoje.\n"+
			"Pedimos que entre em contato para realizar a regularização.", sale.DueDays.Int32)

		whatsAppEvents = append(whatsAppEvents, events.WhatsApp{
			IDSale:       pgconv.PgUUIDToUUID(data.SaleID),
			CompanyID:    companyID,
			CustomerName: sale.CustomerName,
			PhoneNumber:  customer.Whatsapp,
			Value:        pgconv.PgNumericToFloat64(sale.TotalAmount),
			DueDate:      pgconv.PgTimestamptzToTime(sale.CreatedAt),
			InstanceName: instanceName,
			Message:      msg,
		})

		companyCounts[companyID]++

		log.Info().Msgf("CompanyID %s", data.CompanyID)

	}

	var announcementEvents []events.Announcement
	now := time.Now()
	for companyID, total := range companyCounts {
		announcementEvents = append(announcementEvents, events.Announcement{
			CompanyID:     companyID,
			Title:         "Vencimento de vendas",
			Message:       fmt.Sprintf("%d venda(s) da sua empresa venceram hoje.", total),
			Type:          "info",
			TotalVencidas: total,
			StartsAt:      now,
			ExpiresAt:     now.Add(24 * time.Hour),
		})
	}

	return domain.OverdueSalesResult{
		WhatsAppEvents:     whatsAppEvents,
		AnnouncementEvents: announcementEvents,
	}, tx.Commit(ctx)
}

func (s *Service) ContSalesPendingAndOverdue(ctx context.Context, companyId uuid.UUID) (int64, error) {
	return s.repo.ContSalesPendingAndOverdue(ctx, pgconv.ParseUUIDToPgType(companyId))
}

func (s *Service) ListSalesWithDetails(ctx context.Context, companyId uuid.UUID) ([]domain.ListSalesWithInstallmentsResponse, error) {
	rows, err := s.repo.ListSalesWithDetails(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListSalesWithInstallmentsResponse{}, err
	}

	var response []domain.ListSalesWithInstallmentsResponse

	salesMap := make(map[uuid.UUID]*domain.ListSalesWithInstallmentsResponse)
	var orderedIds []uuid.UUID

	for _, row := range rows {
		saleId := pgconv.PgUUIDToUUID(row.SaleID)

		if _, exists := salesMap[saleId]; !exists {

			salesMap[saleId] = &domain.ListSalesWithInstallmentsResponse{
				Sale: domain.ListSalesResponse{
					SaleID:                 saleId,
					SaleAt:                 pgconv.PgTimestamptzToTime(row.SaleAt),
					Subtotal:               pgconv.PgNumericToFloat64(row.Subtotal),
					DiscountAmount:         pgconv.PgNumericToFloat64(row.DiscountAmount),
					TotalAmount:            pgconv.PgNumericToFloat64(row.TotalAmount),
					InstallmentsCount:      row.InstallmentsCount,
					PaymentMethod:          row.PaymentMethod,
					SaleStatus:             row.SaleStatus,
					CustomerID:             pgconv.PgUUIDToUUID(row.CustomerID),
					CustomerName:           row.CustomerName,
					InstallmentTotalAmount: pgconv.PgNumericToFloat64(row.InstallmentBalance),
					DownPayment:            pgconv.PgNumericToFloat64(row.DownPayment),
				},
				Products:      []domain.ListProductResponse{},
				AccReceivable: []domain.ListAccReceivableResponse{},
			}
			orderedIds = append(orderedIds, saleId)
		}
		itemId := pgconv.PgUUIDToUUID(row.SaleItemID)
		isProductNew := true

		for _, p := range salesMap[saleId].Products {
			if p.SaleItemID == itemId {
				isProductNew = false
				break
			}
		}
		if isProductNew && row.ProductID.Valid {
			salesMap[saleId].Products = append(salesMap[saleId].Products, domain.ListProductResponse{
				SaleItemID:   pgconv.PgUUIDToUUID(row.SaleItemID),
				ProductID:    pgconv.PgUUIDToUUID(row.ProductID),
				Quantity:     row.Quantity,
				UnitPrice:    pgconv.PgNumericToFloat64(row.UnitPrice),
				ItemDiscount: pgconv.PgNumericToFloat64(row.ItemDiscount),
				ProductName:  row.ProductName,
			})
		}

		instId := pgconv.PgUUIDToUUID(row.InstallmentID)
		isInstNew := true

		for _, i := range salesMap[saleId].AccReceivable {
			if i.InstallmentID == instId {
				isInstNew = false
				break
			}
		}
		if isInstNew && row.InstallmentID.Valid {
			salesMap[saleId].AccReceivable = append(salesMap[saleId].AccReceivable, domain.ListAccReceivableResponse{
				InstallmentID:      pgconv.PgUUIDToUUID(row.InstallmentID),
				InstallmentBalance: pgconv.PgNumericToFloat64(row.InstallmentBalance),
				DueDate:            pgconv.PgDateToString(row.DueDate),
				InstallmentNumber:  pgconv.PgInt4ToInt(row.InstallmentNumber),
				InstallmentStatus:  pgconv.ParsePgTextToString(row.InstallmentStatus),
			})
		}

	}

	for _, id := range orderedIds {
		response = append(response, *salesMap[id])
	}

	return response, nil
}

func (s *Service) ListSalesWithDetailsPendingOverdue(ctx context.Context, companyId uuid.UUID) ([]domain.ListSalesWithInstallmentsResponse, error) {
	rows, err := s.repo.ListSalesWithDetailsPendingOverdue(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListSalesWithInstallmentsResponse{}, err
	}

	var response []domain.ListSalesWithInstallmentsResponse

	salesMap := make(map[uuid.UUID]*domain.ListSalesWithInstallmentsResponse)
	var orderedIds []uuid.UUID

	for _, row := range rows {
		saleId := pgconv.PgUUIDToUUID(row.SaleID)

		if _, exists := salesMap[saleId]; !exists {

			salesMap[saleId] = &domain.ListSalesWithInstallmentsResponse{
				Sale: domain.ListSalesResponse{
					SaleID:                 saleId,
					SaleAt:                 pgconv.PgTimestamptzToTime(row.SaleAt),
					Subtotal:               pgconv.PgNumericToFloat64(row.Subtotal),
					DiscountAmount:         pgconv.PgNumericToFloat64(row.DiscountAmount),
					TotalAmount:            pgconv.PgNumericToFloat64(row.TotalAmount),
					InstallmentsCount:      row.InstallmentsCount,
					PaymentMethod:          row.PaymentMethod,
					SaleStatus:             row.SaleStatus,
					CustomerID:             pgconv.PgUUIDToUUID(row.CustomerID),
					CustomerName:           row.CustomerName,
					InstallmentTotalAmount: pgconv.PgNumericToFloat64(row.InstallmentBalance),
					DownPayment:            pgconv.PgNumericToFloat64(row.DownPayment),
				},
				Products:      []domain.ListProductResponse{},
				AccReceivable: []domain.ListAccReceivableResponse{},
			}
			orderedIds = append(orderedIds, saleId)
		}
		itemId := pgconv.PgUUIDToUUID(row.SaleItemID)
		isProductNew := true

		for _, p := range salesMap[saleId].Products {
			if p.SaleItemID == itemId {
				isProductNew = false
				break
			}
		}
		if isProductNew && row.ProductID.Valid {
			salesMap[saleId].Products = append(salesMap[saleId].Products, domain.ListProductResponse{
				SaleItemID:   pgconv.PgUUIDToUUID(row.SaleItemID),
				ProductID:    pgconv.PgUUIDToUUID(row.ProductID),
				Quantity:     row.Quantity,
				UnitPrice:    pgconv.PgNumericToFloat64(row.UnitPrice),
				ItemDiscount: pgconv.PgNumericToFloat64(row.ItemDiscount),
				ProductName:  row.ProductName,
			})
		}

		instId := pgconv.PgUUIDToUUID(row.InstallmentID)
		isInstNew := true

		for _, i := range salesMap[saleId].AccReceivable {
			if i.InstallmentID == instId {
				isInstNew = false
				break
			}
		}
		if isInstNew && row.InstallmentID.Valid {
			salesMap[saleId].AccReceivable = append(salesMap[saleId].AccReceivable, domain.ListAccReceivableResponse{
				InstallmentID:      pgconv.PgUUIDToUUID(row.InstallmentID),
				InstallmentBalance: pgconv.PgNumericToFloat64(row.InstallmentBalance),
				DueDate:            pgconv.PgDateToString(row.DueDate),
				InstallmentNumber:  pgconv.PgInt4ToInt(row.InstallmentNumber),
				InstallmentStatus:  pgconv.ParsePgTextToString(row.InstallmentStatus),
			})
		}

	}

	for _, id := range orderedIds {
		response = append(response, *salesMap[id])
	}

	return response, nil
}

func (s *Service) GetRealProfitItem(ctx context.Context, companyId uuid.UUID) (float64, error) {
	productItems, err := s.saleItemsService.ListItemsByCompany(ctx, companyId)
	if err != nil {
		return 0, err
	}

	var totalCost float64
	var totalNetSales float64

	for _, pi := range productItems {
		product, err := s.productService.GetProductById(ctx, pi.ProductID)
		if err != nil {
			return 0, err
		}

		totalNetSales += (pi.UnitPrice - pi.Discount) * float64(pi.Quantity)

		totalCost += product.CostPrice * float64(pi.Quantity)
	}

	if totalNetSales <= 0 {
		return 0, nil
	}

	profitMargin := ((totalNetSales - totalCost) / totalNetSales) * 100

	return profitMargin, nil
}

func (s *Service) GetTop5RealProfitItem(ctx context.Context, companyId uuid.UUID) ([]domain.GetTop5RealProfitItemResponse, error) {
	products, err := s.productService.ListProductsByCompany(ctx, companyId)
	if err != nil {
		return []domain.GetTop5RealProfitItemResponse{}, err
	}

	productItems, err := s.saleItemsService.ListItemsByCompany(ctx, companyId)
	if err != nil {
		return []domain.GetTop5RealProfitItemResponse{}, err
	}

	var response []domain.GetTop5RealProfitItemResponse

	for _, product := range products {
		var totalCost float64
		var totalNetSales float32
		found := false

		for _, item := range productItems {
			if item.ProductID == product.ID {
				totalNetSales += (float32(item.UnitPrice) - float32(item.Discount))
				totalCost += product.CostPrice * float64(item.Quantity)
				found = true
			}
		}

		if found && totalNetSales > 0 {
			margin := ((totalNetSales - float32(totalCost)) / totalNetSales) * 100
			response = append(response, domain.GetTop5RealProfitItemResponse{
				ProductsName:      product.Name,
				ProductRealProfit: float64(margin),
				TotalSale:         float64(totalNetSales),
			})
		}
	}

	sort.Slice(response, func(i, j int) bool {
		return response[i].ProductRealProfit > response[j].ProductRealProfit
	})

	limit := 5

	if len(response) < 5 {
		limit = len(response)
	}

	return response[:limit], nil
}

func (s *Service) GetPerformanceMonth(ctx context.Context, companyId uuid.UUID) ([]domain.GetPerformanceMonthResponse, error) {
	dataBase := time.Now()

	var response []domain.GetPerformanceMonthResponse

	for i := 0; i <= 6; i++ {
		startMount := time.Date(dataBase.Year(), dataBase.Month()-time.Month(i), 1, 0, 0, 0, 0, dataBase.Location())

		productItems, err := s.saleItemsService.ListItemsByDate(ctx, companyId, startMount)
		if err != nil {
			return []domain.GetPerformanceMonthResponse{}, err
		}

		var totalCost float64
		var totalNetSales float64

		for _, pi := range productItems {
			product, err := s.productService.GetProductById(ctx, pi.ProductID)
			if err != nil {
				return []domain.GetPerformanceMonthResponse{}, err
			}

			totalNetSales += (pi.UnitPrice - pi.Discount) * float64(pi.Quantity)

			totalCost += product.CostPrice * float64(pi.Quantity)
		}

		var profitMargin float64
		if totalNetSales > 0 {
			profitMargin = ((totalNetSales - totalCost) / totalNetSales) * 100
		}

		response = append(response, domain.GetPerformanceMonthResponse{
			Mount:      startMount.Format("01/2006"),
			RealProfit: math.Round(profitMargin*100) / 100,
			TotalSale:  totalNetSales,
		})

	}

	return response, nil
}

func (s *Service) GetTotalInvestmentCategory(ctx context.Context, companyId uuid.UUID) ([]domain.GetTotalInvestmentCategoryResponse, error) {
	categories, err := s.productCategoriesService.ListProductCategoryByCompanyId(ctx, companyId)
	if err != nil {
		return []domain.GetTotalInvestmentCategoryResponse{}, err
	}

	productItems, err := s.saleItemsService.ListItemsByCompany(ctx, companyId)
	if err != nil {
		return []domain.GetTotalInvestmentCategoryResponse{}, err
	}

	var response []domain.GetTotalInvestmentCategoryResponse

	for _, category := range categories {
		var totalInvestment float64
		products, err := s.productService.ListProductsByCategoryId(ctx, category.ID, companyId)
		if err != nil {
			return []domain.GetTotalInvestmentCategoryResponse{}, err
		}
		var catQuantity int
		var finalStock int

		for _, product := range products {

			var soldQuantity int

			soldQuantity += int(product.Quantity)
			finalStock += int(product.Quantity)

			for _, item := range productItems {
				if item.ProductID == product.ID {
					soldQuantity += int(item.Quantity)
				}
			}

			totalInvestment += product.CostPrice * float64(soldQuantity)

			catQuantity += soldQuantity
		}

		mediaStock := (catQuantity + finalStock) / 2

		var stockTurnover float64

		if catQuantity > 0 {
			stockTurnover = float64(catQuantity) / float64(mediaStock)
		}

		response = append(response, domain.GetTotalInvestmentCategoryResponse{
			CategoryName:    category.Name,
			TotalInvestment: totalInvestment,
			Amount:          int(catQuantity),
			StockTurnover:   stockTurnover,
		})

	}

	return response, nil
}

func (s *Service) MarginDistribution(ctx context.Context, companyId uuid.UUID) ([]domain.MarginDistributionResponse, error) {
	products, err := s.productService.ListProductsByCompany(ctx, companyId)
	if err != nil {
		return []domain.MarginDistributionResponse{}, err
	}

	productItems, err := s.saleItemsService.ListItemsByCompany(ctx, companyId)
	if err != nil {
		return []domain.MarginDistributionResponse{}, err
	}

	var countBaixa, countMedia10_20, countMedia20_30, countMedia30_40, countAlta int

	for _, product := range products {
		var totalCost float64
		var totalNetSales float32
		found := false

		for _, item := range productItems {
			if item.ProductID == product.ID {
				totalNetSales += (float32(item.UnitPrice) - float32(item.Discount))
				totalCost += product.CostPrice * float64(item.Quantity)
				found = true
			}
		}

		if found && totalNetSales > 0 {
			margin := ((totalNetSales - float32(totalCost)) / totalNetSales) * 100
			if margin < 10 {
				countBaixa++
			} else if margin >= 10 && margin <= 20 {
				countMedia10_20++
			} else if margin >= 20 && margin <= 30 {
				countMedia20_30++
			} else if margin >= 30 && margin <= 40 {
				countMedia30_40++
			} else {
				countAlta++
			}
		}
	}

	response := []domain.MarginDistributionResponse{
		{Label: "0% - 10%", Count: countBaixa},
		{Label: "10% - 20%", Count: countMedia10_20},
		{Label: "20% - 30%", Count: countMedia20_30},
		{Label: "30% - 40%", Count: countMedia30_40},
		{Label: "40%+", Count: countAlta},
	}

	return response, nil
}

func (s *Service) GetPendingSalesDetailedReport(ctx context.Context, companyId uuid.UUID, saleAt time.Time, saleAt2 time.Time) ([]domain.GetPendingSalesDetailedReportResponse, error) {
	report, err := s.repo.GetPendingSalesDetailedReport(ctx, db.GetPendingSalesDetailedReportParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		SaleAt:    pgconv.TimeToPgTimestamptz(saleAt),
		SaleAt_2:  pgconv.TimeToPgTimestamptz(saleAt2),
	})
	if err != nil {
		return []domain.GetPendingSalesDetailedReportResponse{}, err
	}

	var response []domain.GetPendingSalesDetailedReportResponse

	for _, row := range report {
		response = append(response, domain.GetPendingSalesDetailedReportResponse{
			SaleID:                 pgconv.PgUUIDToUUID(row.SaleID),
			SaleAt:                 pgconv.PgTimestamptzToTime(row.SaleAt),
			Subtotal:               pgconv.PgNumericToFloat64(row.Subtotal),
			DiscountAmount:         pgconv.PgNumericToFloat64(row.DiscountAmount),
			TotalAmount:            pgconv.PgNumericToFloat64(row.TotalAmount),
			InstallmentsCount:      row.InstallmentsCount,
			PaymentMethod:          row.PaymentMethod,
			SaleStatus:             row.SaleStatus,
			CustomerID:             pgconv.PgUUIDToUUID(row.CustomerID),
			CustomerName:           row.CustomerName,
			SaleItemID:             pgconv.PgUUIDToUUID(row.SaleItemID),
			ProductID:              pgconv.PgUUIDToUUID(row.ProductID),
			Quantity:               row.Quantity,
			UnitPrice:              pgconv.PgNumericToFloat64(row.UnitPrice),
			ItemDiscount:           pgconv.PgNumericToFloat64(row.ItemDiscount),
			ProductName:            row.ProductName,
			InstallmentID:          pgconv.PgUUIDToUUID(row.InstallmentID),
			InstallmentTotalAmount: pgconv.PgNumericToFloat64(row.InstallmentTotalAmount),
			InstallmentBalance:     pgconv.PgNumericToFloat64(row.InstallmentBalance),
			DueDate:                pgconv.PgDateToString(row.DueDate),
			InstallmentNumber:      pgconv.PgInt4ToInt(row.InstallmentNumber),
			InstallmentStatus:      pgconv.ParsePgTextToString(row.InstallmentStatus),
		})
	}
	return response, nil
}

func (s *Service) ListSalesWithDetailsPaginate(ctx context.Context, companyId uuid.UUID, pagination globalDomain.PaginationParams) (domain.SaleResponsePaginate, error) {
	total, err := s.repo.CountSalesByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.SaleResponsePaginate{}, err
	}

	rows, err := s.repo.ListSalesWithDetailsPaginate(ctx, db.ListSalesWithDetailsPaginateParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Limit:     pagination.PerPage,
		Offset:    (pagination.Page - 1) * pagination.PerPage,
	})

	var response []domain.ListSalesWithInstallmentsResponse
	var totalInvoiced float64
	var totalPending float64
	var salesCanceled int64

	salesMap := make(map[uuid.UUID]*domain.ListSalesWithInstallmentsResponse)
	var orderedIds []uuid.UUID

	for _, row := range rows {
		saleId := pgconv.PgUUIDToUUID(row.SaleID)

		if _, exists := salesMap[saleId]; !exists {

			if row.SaleStatus == "paid" {
				totalInvoiced += pgconv.PgNumericToFloat64(row.TotalAmount)
			}

			if pgconv.ParsePgTextToString(row.InstallmentStatus) == "pending" {
				totalPending += pgconv.PgNumericToFloat64(row.InstallmentBalance)
			}

			if row.SaleStatus == "canceled" {
				salesCanceled += 1
			}

			salesMap[saleId] = &domain.ListSalesWithInstallmentsResponse{
				Sale: domain.ListSalesResponse{
					SaleID:                 saleId,
					SaleAt:                 pgconv.PgTimestamptzToTime(row.SaleAt),
					Subtotal:               pgconv.PgNumericToFloat64(row.Subtotal),
					DiscountAmount:         pgconv.PgNumericToFloat64(row.DiscountAmount),
					TotalAmount:            pgconv.PgNumericToFloat64(row.TotalAmount),
					InstallmentsCount:      row.InstallmentsCount,
					PaymentMethod:          row.PaymentMethod,
					SaleStatus:             row.SaleStatus,
					CustomerID:             pgconv.PgUUIDToUUID(row.CustomerID),
					CustomerName:           row.CustomerName,
					InstallmentTotalAmount: pgconv.PgNumericToFloat64(row.InstallmentBalance),
					DownPayment:            pgconv.PgNumericToFloat64(row.DownPayment),
				},
				Products:      []domain.ListProductResponse{},
				AccReceivable: []domain.ListAccReceivableResponse{},
			}
			orderedIds = append(orderedIds, saleId)
		}
		itemId := pgconv.PgUUIDToUUID(row.SaleItemID)
		isProductNew := true

		for _, p := range salesMap[saleId].Products {
			if p.SaleItemID == itemId {
				isProductNew = false
				break
			}
		}
		if isProductNew && row.ProductID.Valid {
			salesMap[saleId].Products = append(salesMap[saleId].Products, domain.ListProductResponse{
				SaleItemID:   pgconv.PgUUIDToUUID(row.SaleItemID),
				ProductID:    pgconv.PgUUIDToUUID(row.ProductID),
				Quantity:     row.Quantity,
				UnitPrice:    pgconv.PgNumericToFloat64(row.UnitPrice),
				ItemDiscount: pgconv.PgNumericToFloat64(row.ItemDiscount),
				ProductName:  row.ProductName,
			})
		}

		instId := pgconv.PgUUIDToUUID(row.InstallmentID)
		isInstNew := true

		for _, i := range salesMap[saleId].AccReceivable {
			if i.InstallmentID == instId {
				isInstNew = false
				break
			}
		}
		if isInstNew && row.InstallmentID.Valid {
			salesMap[saleId].AccReceivable = append(salesMap[saleId].AccReceivable, domain.ListAccReceivableResponse{
				InstallmentID:      pgconv.PgUUIDToUUID(row.InstallmentID),
				InstallmentBalance: pgconv.PgNumericToFloat64(row.InstallmentBalance),
				DueDate:            pgconv.PgDateToString(row.DueDate),
				InstallmentNumber:  pgconv.PgInt4ToInt(row.InstallmentNumber),
				InstallmentStatus:  pgconv.ParsePgTextToString(row.InstallmentStatus),
			})
		}

	}

	for _, id := range orderedIds {
		response = append(response, *salesMap[id])
	}

	paginationResponse := globalDomain.NewPaginatedResponse(response, total, pagination)

	return domain.SaleResponsePaginate{
		PaginatedResponse: paginationResponse,
		SalesCount:        total,
		TotalInvoiced:     totalInvoiced,
		TotalPending:      totalPending,
		SalesCanceled:     salesCanceled,
	}, nil
}

func (s *Service) UpdateSale(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, saleId uuid.UUID, req domain.UpdateSaleParams) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	if req.PaymentMethod != enums.PaymentMethodInstallments {
		return errors.New("Não é possivel atualiar esse venda")
	}

	currentSale, err := s.repo.GetSaleById(ctx, db.GetSaleByIdParams{
		ID:        pgconv.ParseUUIDToPgType(saleId),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if err != nil {
		return err
	}

	targetTime := time.Now().Add(2 * time.Hour)

	if currentSale.CreatedAt.Valid && !currentSale.CreatedAt.Time.Before(targetTime) {
		return errors.New("A venda so pode ser atualizada até 2 horas depois de ser realizada")
	}

	arg := db.UpdateSaleParams{
		DiscountAmount:    currentSale.DiscountAmount,
		InstallmentsCount: currentSale.InstallmentsCount,
		DownPayment:       currentSale.DownPayment,
		DueDays:           currentSale.DueDays,
		PaymentMethod:     currentSale.PaymentMethod,
	}

	domain.ApplyUpdateSaleParams(req, &arg)

	totalAmount := pgconv.PgNumericToFloat64(currentSale.Subtotal) * (1 - (pgconv.PgNumericToFloat64(arg.DiscountAmount) / 100))
	var status string

	if arg.PaymentMethod == enums.PaymentMethodInstallments {

		if currentSale.InstallmentsCount < arg.InstallmentsCount || currentSale.InstallmentsCount > arg.InstallmentsCount {
			err = s.accountsReceivableService.DeleteAccountReceivableBySaleIDTx(ctx, tx, saleId, companyId)
			if err != nil {
				return err
			}
		}
		installmentValue := (totalAmount - pgconv.PgNumericToFloat64(arg.DownPayment)) / float64(arg.InstallmentsCount)
		status = "pending"

		dataBase := time.Now()
		for i := 1; i < int(arg.InstallmentsCount); i++ {
			var maturity time.Time

			if dataBase.Day() >= int(req.DueDays) {
				maturity = time.Date(
					dataBase.Year(),
					dataBase.Month()+time.Month(i+1),
					int(req.DueDays),
					0, 0, 0, 0,
					dataBase.Location(),
				)
			} else {
				maturity = time.Date(
					dataBase.Year(),
					dataBase.Month()+time.Month(i),
					int(req.DueDays),
					0, 0, 0, 0,
					dataBase.Location(),
				)
			}

			err = s.accountsReceivableService.CreateAccountReceivable(ctx, tx, userId, companyId, accountsReceivableDomain.CreateAccountReceivableRequest{
				CustomerID:        pgconv.PgUUIDToUUID(currentSale.CustomerID),
				SaleID:            saleId,
				TotalAmount:       installmentValue,
				Balance:           installmentValue,
				DueDate:           maturity.Format("2006-01-02"),
				InstallmentNumber: int64(i),
				TotalInstallments: int64(arg.InstallmentsCount),
			})
			if err != nil {
				return err
			}

		}

	}

	err = s.customerService.UpdateCustomerBalanceSubTx(ctx, tx, pgconv.PgUUIDToUUID(currentSale.CustomerID), customerDomain.UpdateBalanceDueCustomerRequest{
		BalanceDue: totalAmount,
		Prohibited: pgconv.PgNumericToFloat64(arg.DownPayment),
		UpdatedBy:  userId,
	})
	if err != nil {
		return err
	}

	err = txRepo.UpdateSale(ctx, db.UpdateSaleParams{
		DiscountAmount:    arg.DiscountAmount,
		Subtotal:          currentSale.Subtotal,
		TotalAmount:       pgconv.Float64ToPgNumeric(totalAmount),
		InstallmentsCount: arg.InstallmentsCount,
		DownPayment:       arg.DownPayment,
		DueDays:           arg.DueDays,
		PaymentMethod:     arg.PaymentMethod,
		UpdatedBy:         pgconv.ParseUUIDToPgType(userId),
		Status:            status,
		ID:                pgconv.ParseUUIDToPgType(saleId),
		CompanyID:         pgconv.ParseUUIDToPgType(companyId),
	})
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetInventoryTurnover(ctx context.Context, companyID uuid.UUID) (domain.GetInventoryTurnoverResponse, error) {
	products, err := s.productService.ListProductsByCompany(ctx, companyID)
	if err != nil {
		return domain.GetInventoryTurnoverResponse{}, err
	}

	productsSales, err := s.saleItemsService.ListItemsByCompany(ctx, companyID)
	if err != nil {
		return domain.GetInventoryTurnoverResponse{}, err
	}

	var totalStockProducts float64
	var totalStockProductsSale float64

	productMap := make(map[uuid.UUID]float64)
	for _, product := range products {
		totalStockProducts += product.CostPrice * float64(product.Quantity)
		productMap[product.ID] = product.CostPrice
	}

	for _, saleItem := range productsSales {
		product, exists := productMap[saleItem.ProductID]
		if !exists {
			continue
		}

		totalStockProductsSale += product * float64(saleItem.Quantity)
	}

	var stockTurnover float64
	if totalStockProducts > 0 {
		stockTurnover = (totalStockProductsSale / totalStockProducts) * 100
	}

	return domain.GetInventoryTurnoverResponse{InventoryTurnover: math.Round(stockTurnover*100) / 100}, nil
}
