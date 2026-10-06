package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	accountsReceivableDomain "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/domain"
	accountsReceivableService "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/service"
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	companiesService "github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	companySettingsDomain "github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	customerDomain "github.com/ProTrack-Solutions/protrack-api/internal/customers/domain"
	customerService "github.com/ProTrack-Solutions/protrack-api/internal/customers/service"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/demo"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	metaWhatsAppService "github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/service"
	plansService "github.com/ProTrack-Solutions/protrack-api/internal/plans/service"
	productService "github.com/ProTrack-Solutions/protrack-api/internal/products/service"
	productCategoriesService "github.com/ProTrack-Solutions/protrack-api/internal/products_categories/service"
	saleItemDomain "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/domain"
	saleItemsService "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/repository"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/events"
	subscriptionService "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	CountSalesDeletedByCompany(ctx context.Context, companyId pgtype.UUID) (int64, error)
	GetTotalAmountPending(ctx context.Context, companyId pgtype.UUID) (float64, error)
	GetTotalAmountPaid(ctx context.Context, companyId pgtype.UUID) (float64, error)
	GetTotalAmountIsPending(ctx context.Context, companyID pgtype.UUID) (pgtype.Numeric, error)
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
	subscriptionService       *subscriptionService.Service
	plansService              *plansService.Service
	metaWhatsAppService       *metaWhatsAppService.Service
	companySettings           companySettingsDomain.ServiceInterface
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
	subscriptionService *subscriptionService.Service,
	plansService *plansService.Service,
	metaWhatsAppService *metaWhatsAppService.Service,
	companySettings companySettingsDomain.ServiceInterface,
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
		subscriptionService:       subscriptionService,
		plansService:              plansService,
		metaWhatsAppService:       metaWhatsAppService,
		companySettings:           companySettings,
	}
}

func calculatePeriodStart(currentPeriodEnd time.Time, billingCycle string) time.Time {
	switch strings.ToLower(billingCycle) {
	case "yearly":
		return currentPeriodEnd.AddDate(-1, 0, 0)
	default: // "monthly" e qualquer valor não reconhecido caem aqui
		return currentPeriodEnd.AddDate(0, -1, 0)
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
	q := db.New(tx)

	var status string
	var subTotal float64
	var totalAmount float64
	var discount float64

	// Cliente informado precisa existir e ser da mesma empresa
	if req.CustomerID != uuid.Nil {
		customer, err := q.GetCustomerById(ctx, pgconv.ParseUUIDToPgType(req.CustomerID))
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && pgconv.PgUUIDToUUID(customer.CompanyID) != companyId) {
			return uuid.Nil, domain.ErrSaleCustomerNotFound
		}
		if err != nil {
			return uuid.Nil, err
		}
	}

	// Confere produtos e estoque antes de gravar qualquer coisa, para devolver um erro claro
	for i, item := range req.Items {
		product, err := q.GetProductById(ctx, pgconv.ParseUUIDToPgType(item.ProductID))
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && pgconv.PgUUIDToUUID(product.CompanyID) != companyId) {
			return uuid.Nil, fmt.Errorf("%w (item %d)", domain.ErrSaleProductNotFound, i+1)
		}
		if err != nil {
			return uuid.Nil, err
		}

		if !product.SellInBulk {
			available := int32(pgconv.PgInt4ToInt(product.Quantity))
			if available < item.Quantity {
				return uuid.Nil, fmt.Errorf("%w para o produto %s: disponível %d, solicitado %d",
					domain.ErrInsufficientStock, product.Name, available, item.Quantity)
			}
		}

		subTotal += float64(item.Quantity) * pgconv.PgNumericToFloat64(product.SalePrice)
	}

	if subTotal <= 0 {
		return uuid.Nil, fmt.Errorf("%w: o valor total dos produtos deve ser maior que zero", domain.ErrInvalidSale)
	}

	// O desconto chega em R$ e é gravado na venda e rateado entre os itens
	discountValue := roundMoney(req.DiscountAmount)
	if discountValue > subTotal {
		return uuid.Nil, fmt.Errorf("%w: o desconto não pode ser maior que o valor dos produtos (R$ %.2f)", domain.ErrInvalidSale, subTotal)
	}
	totalAmount = subTotal - discountValue

	if req.PaymentMethod == enums.PaymentMethodInstallments && req.Prohibited >= totalAmount {
		return uuid.Nil, fmt.Errorf("%w: a entrada deve ser menor que o total da venda (R$ %.2f)", domain.ErrInvalidSale, totalAmount)
	}

	// Entrada, parcelas e vencimento só fazem sentido na venda a prazo
	if req.PaymentMethod != enums.PaymentMethodInstallments {
		req.Prohibited = 0
	}

	// 1. Definição do Status e Atualização de Saldo Devedor
	if req.PaymentMethod == "installments" {
		if err := s.customerService.UpdateCustomerBalanceAddTx(ctx, tx, req.CustomerID, customerDomain.UpdateBalanceDueCustomerRequest{
			BalanceDue: totalAmount,
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
		CustomerID:        pgconv.OptionalUUIDToPgType(req.CustomerID),
		CompanyID:         pgconv.ParseUUIDToPgType(companyId),
		DiscountAmount:    pgconv.Float64ToPgNumeric(discountValue),
		Subtotal:          pgconv.Float64ToPgNumeric(subTotal),
		TotalAmount:       pgconv.Float64ToPgNumeric(totalAmount),
		DueDays:           pgconv.OptionalIntToPgInt4(dueDaysVal),
		PaymentMethod:     req.PaymentMethod,
		Status:            status,
		CreatedBy:         pgconv.ParseUUIDToPgType(userId),
		InstallmentsCount: installments,
		DownPayment:       pgconv.Float64ToPgNumeric(req.Prohibited),
		BuyerDocument:     pgconv.ParseStringToPgText(req.BuyerDocument),
	})
	if err != nil {
		return uuid.Nil, err
	}

	if req.PaymentMethod == "installments" {
		if err := s.createInstallmentsTx(ctx, tx, userId, companyId, req.CustomerID, pgconv.PgUUIDToUUID(id),
			totalAmount-req.Prohibited, req.InstallmentsCount, req.DueDays); err != nil {
			return uuid.Nil, err
		}
	}

	for _, itemReq := range req.Items {

		product, err := s.productService.GetProductByIdTx(ctx, tx, itemReq.ProductID)
		if err != nil {
			return uuid.Nil, err
		}

		saleID := pgconv.PgUUIDToUUID(id)



		itemTotal := float64(itemReq.Quantity) * product.SalePrice
		proportion := itemTotal / subTotal
		discount = discountValue * proportion

		if err := s.saleItemsService.CreateSaleItemInTx(ctx, tx, saleItemDomain.CreateSaleItemRequest{
			SaleID:    saleID,
			ProductID: itemReq.ProductID,
			Quantity:  itemReq.Quantity,
			UnitPrice: product.SalePrice,
			Discount:  discount,
		}, companyId); err != nil {
			// Estoque pode ter mudado entre a conferência e a baixa (venda concorrente)
			switch {
			case errors.Is(err, saleItemDomain.ErrInsufficientStock):
				detail := strings.TrimPrefix(err.Error(), saleItemDomain.ErrInsufficientStock.Error())
				return uuid.Nil, fmt.Errorf("%w%s", domain.ErrInsufficientStock, detail)
			case errors.Is(err, saleItemDomain.ErrProductNotFound):
				return uuid.Nil, domain.ErrSaleProductNotFound
			}
			return uuid.Nil, err
		}
	}

	return pgconv.PgUUIDToUUID(id), tx.Commit(ctx)
}

// roundMoney arredonda um valor em R$ para centavos.
func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

// createInstallmentsTx gera as parcelas (contas a receber) de uma venda a prazo.
// Cada parcela vence no dia dueDays; a primeira cai no mês atual se esse dia ainda não chegou.
func (s *Service) createInstallmentsTx(ctx context.Context, tx db.DBTX, userId, companyId, customerId, saleId uuid.UUID, amount float64, installmentsCount, dueDays int32) error {
	installmentValue := amount / float64(installmentsCount)
	dataBase := time.Now()

	for i := 0; i < int(installmentsCount); i++ {
		monthOffset := i
		if dataBase.Day() >= int(dueDays) {
			monthOffset = i + 1
		}

		maturity := time.Date(
			dataBase.Year(),
			dataBase.Month()+time.Month(monthOffset),
			int(dueDays),
			0, 0, 0, 0,
			dataBase.Location(),
		)

		if err := s.accountsReceivableService.CreateAccountReceivableInTx(ctx, tx, userId, companyId, accountsReceivableDomain.CreateAccountReceivableRequest{
			CustomerID:        customerId,
			SaleID:            saleId,
			Balance:           installmentValue,
			TotalAmount:       installmentValue,
			InstallmentNumber: int64(i + 1),
			TotalInstallments: int64(installmentsCount),
			DueDate:           maturity.Format("2006-01-02"),
		}); err != nil {
			return err
		}
	}

	return nil
}

// DeleteSale cancela a venda desfazendo, na mesma transação, tudo o que o CreateSale fez:
// devolve o estoque, cancela as parcelas em aberto e abate do saldo devedor do cliente
// o valor que ainda não foi pago. Parcelas já pagas e o histórico de pagamentos são mantidos.
func (s *Service) DeleteSale(ctx context.Context, id uuid.UUID, req domain.DeleteSaleRequest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)
	saleID := pgconv.ParseUUIDToPgType(id)
	companyID := pgconv.ParseUUIDToPgType(req.CompanyID)
	userID := pgconv.ParseUUIDToPgType(req.DeletedBy)

	// FOR UPDATE evita que dois cancelamentos simultâneos devolvam o estoque duas vezes
	sale, err := q.GetSaleByIdForUpdate(ctx, db.GetSaleByIdForUpdateParams{
		ID:        saleID,
		CompanyID: companyID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSaleNotFound
	}
	if err != nil {
		return err
	}

	if sale.DeletedAt.Valid {
		return domain.ErrSaleAlreadyCanceled
	}

	// 1. Devolve o estoque (inverso do DecrementStock feito no CreateSaleItemInTx)
	items, err := q.ListItemsBySaleForRestock(ctx, saleID)
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.SellInBulk {
			continue
		}
		if err := q.IncrementStock(ctx, db.IncrementStockParams{
			Quantity: pgconv.IntToPgInt4(int(item.Quantity)),
			ID:       item.ProductID,
		}); err != nil {
			return err
		}
	}

	// 2. Cancela as parcelas em aberto e abate do saldo do cliente o que ainda era devido
	// (vendas à vista não têm parcelas, então o saldo em aberto é zero e nada é alterado)
	if sale.CustomerID.Valid {
		openBalance, err := q.GetOpenBalanceBySale(ctx, db.GetOpenBalanceBySaleParams{
			SaleID:    saleID,
			CompanyID: companyID,
		})
		if err != nil {
			return err
		}

		if err := q.CancelAccountsReceivableBySaleId(ctx, db.CancelAccountsReceivableBySaleIdParams{
			SaleID:    saleID,
			CompanyID: companyID,
			UpdatedBy: userID,
		}); err != nil {
			return err
		}

		if remaining := pgconv.PgNumericToFloat64(openBalance); remaining > 0 {
			if err := s.customerService.UpdateCustomerBalanceSubTx(ctx, tx, pgconv.PgUUIDToUUID(sale.CustomerID), customerDomain.UpdateBalanceDueCustomerRequest{
				BalanceDue: remaining,
				UpdatedBy:  req.DeletedBy,
			}); err != nil {
				return err
			}
		}
	}

	// 3. Marca a venda como cancelada
	if err := q.DeleteSale(ctx, db.DeleteSaleParams{
		DeletedBy: userID,
		ID:        saleID,
		CompanyID: companyID,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
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

	var growthPercentage float64
	if res.LastMonthSt > 0 {
		growthPercentage = ((res.CurrentMonthSt - res.LastMonthSt) / res.LastMonthSt) * 100
	} else if res.CurrentMonthSt > 0 {
		growthPercentage = 100
	}

	return domain.GetTotalAmountSummaryRow{
		CurrentMonthSt:   res.CurrentMonthSt,
		LastMonthSt:      res.LastMonthSt,
		GrowthPercentage: math.Round(growthPercentage),
	}, nil
}

func (s *Service) GetTotalAmountIsPending(ctx context.Context, companyId uuid.UUID) (float64, error) {
	total, err := s.repo.GetTotalAmountIsPending(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return 0, err
	}

	return pgconv.PgNumericToFloat64(total), nil
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
		return domain.OverdueSalesResult{}, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	repoTx := s.repo.WithTx(tx)

	response, err := repoTx.UpdateOverdueSalesAndAccounts(ctx)
	if err != nil {
		return domain.OverdueSalesResult{}, fmt.Errorf("failed to update overdue sales and accounts: %w", err)
	}

	var whatsAppEvents []events.WhatsApp
	// agrupa contagem de vendas vencidas por empresa
	companyCounts := make(map[uuid.UUID]int)

	whatsAppEligibility := make(map[uuid.UUID]bool)

	for _, data := range response {

		customer, err := s.customerService.GetCustomerByIdTx(ctx, tx, pgconv.PgUUIDToUUID(data.CustomerID))
		if err != nil {
			return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve customer %s: %w", pgconv.PgUUIDToUUID(data.CustomerID), err)
		}

		sale, err := repoTx.GetSaleByIdJust(ctx, data.SaleID)
		if err != nil {
			log.Error().Err(err).Str("sale_id", data.SaleID.String()).Msg("Erro ao buscar venda para WhatsApp")
			continue
		}

		company, err := s.companiesService.GetCompanyByIDTx(ctx, tx, pgconv.PgUUIDToUUID(data.CompanyID))
		if err != nil {
			return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve company %s: %w", pgconv.PgUUIDToUUID(data.CompanyID), err)
		}

		companyID := pgconv.PgUUIDToUUID(data.CompanyID)

		settingTemplate, err := s.companySettings.GetCompanySetting(ctx, companyID, enums.SaleOverdueTemplate)
		if err != nil {
			return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve setting %s for company %s: %w", enums.SaleOverdueTemplate, companyID, err)
		}

		settingLanguage, err := s.companySettings.GetCompanySetting(ctx, companyID, enums.LanguageSaleOverdueTemplate)
		if err != nil {
			return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve setting %s for company %s: %w", enums.LanguageSaleOverdueTemplate, companyID, err)
		}

		isWhatsappPlan, cached := whatsAppEligibility[companyID]

		if !cached {
			sub, err := s.subscriptionService.GetSubscriptionByCompanyID(ctx, companyID)
			if err != nil {
				return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve subscription for company %s: %w", companyID, err)
			}

			plan, err := s.plansService.GetPlanByID(ctx, sub.PlanID)
			if err != nil {
				return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve plan %s for company %s: %w", sub.PlanID, companyID, err)
			}

			periodStart := calculatePeriodStart(sub.CurrentPeriodEnd, plan.BillingCycle)

			countMessages, err := s.metaWhatsAppService.CountMessagesInPeriod(ctx, companyID, periodStart, sub.CurrentPeriodEnd)
			if err != nil {
				return domain.OverdueSalesResult{}, fmt.Errorf("failed to count whatsapp messages for company %s: %w", companyID, err)
			}
			if countMessages == nil {
				return domain.OverdueSalesResult{}, fmt.Errorf("contagem de mensagens indisponível para empresa %s", companyID)
			}

			isExcessUsage, err := s.companySettings.GetCompanySetting(ctx, companyID, enums.IsExcessUsage)
			if err != nil {
				return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve setting %s for company %s: %w", enums.IsExcessUsage, companyID, err)
			}

			isWhatsAppActive, err := s.companySettings.GetCompanySetting(ctx, companyID, enums.IsWhatsappActive)
			if err != nil {
				return domain.OverdueSalesResult{}, fmt.Errorf("failed to retrieve setting %s for company %s: %w", enums.IsWhatsappActive, companyID, err)
			}

			for _, ft := range plan.Features {
				if ft.FeatureKey == "max_whatsapp_integration" && isWhatsAppActive.Value.(bool) {
					switch {
					case ft.LimitValue > *countMessages:
						isWhatsappPlan = true
					case ft.LimitValue <= *countMessages && isExcessUsage.Value.(bool):
						isWhatsappPlan = true
					default:
						isWhatsappPlan = false
					}
				}
			}

			whatsAppEligibility[companyID] = isWhatsappPlan
		}

		// Os clientes da empresa demo são fictícios: nunca envia WhatsApp.
		if isWhatsappPlan && !demo.IsDemoCompany(companyID) {
			whatsAppEvents = append(whatsAppEvents, events.WhatsApp{
				IDSale:       pgconv.PgUUIDToUUID(data.SaleID),
				CompanyID:    companyID,
				CustomerName: sale.CustomerName,
				PhoneNumber:  customer.Whatsapp,
				Value:        pgconv.PgNumericToFloat64(data.Balance),
				DueDate:      data.DueDate.Time,
				CompanyName:  company.Name,
				PurchaseDate: sale.SaleAt.Time,
				ContactInfo:  company.Phone,
				TemplateName: settingTemplate.Value.(string),
				LanguageCode: settingLanguage.Value.(string),
			})
		}

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

func (s *Service) ListSalesWithDetailsPaginate(ctx context.Context, companyId uuid.UUID, pagination domain.PaginationParams) (domain.SaleResponsePaginate, error) {
	total, err := s.repo.CountSalesByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.SaleResponsePaginate{}, err
	}

	contSaleCancel, err := s.repo.CountSalesDeletedByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.SaleResponsePaginate{}, err
	}

	totalPending, err := s.repo.GetTotalAmountPending(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.SaleResponsePaginate{}, err
	}

	totalPaid, err := s.repo.GetTotalAmountPaid(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.SaleResponsePaginate{}, err
	}

	var saleStatus interface{}
	if pagination.SaleStatus != "" {
		saleStatus = pagination.SaleStatus
	}

	var paymentMethod interface{}
	if pagination.PaymentMethod != "" {
		paymentMethod = pagination.PaymentMethod
	}

	rows, err := s.repo.ListSalesWithDetailsPaginate(ctx, db.ListSalesWithDetailsPaginateParams{
		CompanyID:        pgconv.ParseUUIDToPgType(companyId),
		Limit:            pagination.PerPage,
		Offset:           (pagination.Page - 1) * pagination.PerPage,
		Search:           pagination.Search,
		SaleStatus:       saleStatus,
		PaymentMethod:    paymentMethod,
		PaymentStartDate: pgconv.StringToPgDate(pagination.PaymentStartDate),
		PaymentEndDate:   pgconv.StringToPgDate(pagination.PaymentEndDate),
		SaleStartDate:    pgconv.StringToPgDate(pagination.SaleStartDate),
		SaleEndDate:      pgconv.StringToPgDate(pagination.SaleEndDate),
		SortBy:           pagination.SortBy,
		OrderBy:          pagination.OrderBy,
	})
	if err != nil {
		return domain.SaleResponsePaginate{}, err
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

	paginationResponse := globalDomain.NewPaginatedResponse(response, total, globalDomain.PaginationParams{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Search:    pagination.Search,
		OrderBy:   pagination.OrderBy,
		StartDate: pagination.StartDate,
		EndDate:   pagination.EndDate,
	})

	return domain.SaleResponsePaginate{
		PaginatedResponse: paginationResponse,
		SalesCount:        total,
		TotalInvoiced:     totalPaid,
		TotalPending:      totalPending,
		SalesCanceled:     contSaleCancel,
	}, nil
}

// UpdateSale altera as condições de pagamento (desconto, entrada, parcelas e vencimento)
// de uma venda a prazo. Só é permitido nas primeiras 2 horas e enquanto nenhuma parcela
// tiver sido paga. A dívida antiga do cliente e as parcelas são desfeitas e geradas de novo.
func (s *Service) UpdateSale(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, saleId uuid.UUID, req domain.UpdateSaleParams) error {
	if req.PaymentMethod != enums.PaymentMethodInstallments {
		return fmt.Errorf("%w: apenas vendas a prazo podem ser alteradas", domain.ErrInvalidSaleUpdate)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)
	saleID := pgconv.ParseUUIDToPgType(saleId)
	companyID := pgconv.ParseUUIDToPgType(companyId)

	sale, err := q.GetSaleByIdForUpdate(ctx, db.GetSaleByIdForUpdateParams{
		ID:        saleID,
		CompanyID: companyID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSaleNotFound
	}
	if err != nil {
		return err
	}

	if sale.DeletedAt.Valid {
		return domain.ErrSaleAlreadyCanceled
	}

	if sale.CreatedAt.Valid && time.Since(sale.CreatedAt.Time) > domain.SaleUpdateWindow {
		return domain.ErrSaleUpdateExpired
	}

	if !sale.CustomerID.Valid {
		return fmt.Errorf("%w: venda a prazo exige um cliente", domain.ErrInvalidSaleUpdate)
	}

	receivables, err := s.accountsReceivableService.GetReceivablesBySaleTx(ctx, tx, saleId)
	if err != nil {
		return err
	}
	for _, receivable := range receivables {
		if receivable.Balance < receivable.TotalAmount {
			return domain.ErrSaleHasPayments
		}
	}

	// Campos não informados mantêm o valor atual da venda
	subtotal := pgconv.PgNumericToFloat64(sale.Subtotal)

	discountValue := pgconv.PgNumericToFloat64(sale.DiscountAmount)
	if req.DiscountAmount != nil {
		discountValue = roundMoney(*req.DiscountAmount)
		if discountValue < 0 {
			return fmt.Errorf("%w: o desconto não pode ser negativo", domain.ErrInvalidSaleUpdate)
		}
		if discountValue > subtotal {
			return fmt.Errorf("%w: o desconto não pode ser maior que o valor dos produtos (R$ %.2f)", domain.ErrInvalidSaleUpdate, subtotal)
		}
	}
	totalAmount := subtotal - discountValue

	downPayment := pgconv.PgNumericToFloat64(sale.DownPayment)
	if req.Prohibited != nil {
		downPayment = *req.Prohibited
	}
	if downPayment < 0 || downPayment >= totalAmount {
		return fmt.Errorf("%w: a entrada deve ser menor que o total da venda", domain.ErrInvalidSaleUpdate)
	}

	installmentsCount := sale.InstallmentsCount
	if req.InstallmentsCount > 0 {
		installmentsCount = req.InstallmentsCount
	}
	if installmentsCount <= 0 {
		return fmt.Errorf("%w: informe a quantidade de parcelas", domain.ErrInvalidSaleUpdate)
	}

	dueDays := int32(pgconv.PgInt4ToInt(sale.DueDays))
	if req.DueDays > 0 {
		dueDays = req.DueDays
	}
	if dueDays <= 0 || dueDays > 31 {
		return fmt.Errorf("%w: informe um dia de vencimento entre 1 e 31", domain.ErrInvalidSaleUpdate)
	}

	// 1. Tira do saldo do cliente o que a venda ainda devia (zero se ela era à vista)
	openBalance, err := q.GetOpenBalanceBySale(ctx, db.GetOpenBalanceBySaleParams{
		SaleID:    saleID,
		CompanyID: companyID,
	})
	if err != nil {
		return err
	}

	customerId := pgconv.PgUUIDToUUID(sale.CustomerID)

	if previousDebt := pgconv.PgNumericToFloat64(openBalance); previousDebt > 0 {
		if err := s.customerService.UpdateCustomerBalanceSubTx(ctx, tx, customerId, customerDomain.UpdateBalanceDueCustomerRequest{
			BalanceDue: previousDebt,
			UpdatedBy:  userId,
		}); err != nil {
			return err
		}
	}

	// 2. Lança a nova dívida e gera as parcelas de novo
	newDebt := totalAmount - downPayment

	if err := s.customerService.UpdateCustomerBalanceAddTx(ctx, tx, customerId, customerDomain.UpdateBalanceDueCustomerRequest{
		BalanceDue: newDebt,
		UpdatedBy:  userId,
	}); err != nil {
		return err
	}

	if err := s.accountsReceivableService.DeleteAccountReceivableBySaleIDTx(ctx, tx, saleId, companyId); err != nil {
		return err
	}

	if err := s.createInstallmentsTx(ctx, tx, userId, companyId, customerId, saleId, newDebt, installmentsCount, dueDays); err != nil {
		return err
	}

	// 3. Rateia o novo desconto entre os itens e atualiza a venda
	if err := q.UpdateSaleItemsDiscount(ctx, db.UpdateSaleItemsDiscountParams{
		DiscountAmount: pgconv.Float64ToPgNumeric(discountValue),
		Subtotal:       sale.Subtotal,
		SaleID:         saleID,
	}); err != nil {
		return err
	}

	if err := q.UpdateSale(ctx, db.UpdateSaleParams{
		DiscountAmount:    pgconv.Float64ToPgNumeric(discountValue),
		Subtotal:          sale.Subtotal,
		TotalAmount:       pgconv.Float64ToPgNumeric(totalAmount),
		InstallmentsCount: installmentsCount,
		DownPayment:       pgconv.Float64ToPgNumeric(downPayment),
		DueDays:           pgconv.IntToPgInt4(int(dueDays)),
		PaymentMethod:     string(enums.PaymentMethodInstallments),
		UpdatedBy:         pgconv.ParseUUIDToPgType(userId),
		Status:            "pending",
		ID:                saleID,
		CompanyID:         companyID,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
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
