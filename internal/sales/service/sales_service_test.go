package service_test

import (
	"context"
	"errors"
	"testing"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	salesDomain "github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers e Fixtures
// ---------------------------------------------------------------------------

var errDatabase = errors.New("database error")

// newSvc cria um Service com o mock de repositório injetado.
func newSvc(t *testing.T, repo *mocks.MockRepositoryInterface) *service.Service {
	t.Helper()
	return service.NewServiceWithRepo(repo)
}

// buildDbSaleRow constrói um db.GetSaleByIdRow completo para uso nos testes.
func buildDbSaleRow(id, companyID, customerID, createdBy uuid.UUID) db.GetSaleByIdRow {
	return db.GetSaleByIdRow{
		ID:             pgconv.ParseUUIDToPgType(id),
		CompanyID:      pgconv.ParseUUIDToPgType(companyID),
		CustomerID:     pgconv.ParseUUIDToPgType(customerID),
		TotalAmount:    pgconv.Float64ToPgNumeric(500.00),
		Subtotal:       pgconv.Float64ToPgNumeric(500.00),
		DiscountAmount: pgconv.Float64ToPgNumeric(0),
		PaymentMethod:  "paid",
		Status:         "paid",
		CreatedBy:      pgconv.ParseUUIDToPgType(createdBy),
		UpdatedBy:      pgconv.ParseUUIDToPgType(uuid.Nil),
		DeletedBy:      pgconv.ParseUUIDToPgType(uuid.Nil),
		CreatedAt:      pgconv.TimeToPgTimestamptz(pgconv.PgTimestamptzToTime(pgtype.Timestamptz{Valid: false})),
		CustomerName:   "João Teste",
	}
}

// buildDbListSalesRow constrói um db.ListSalesRow para uso nos testes.
func buildDbListSalesRow(id uuid.UUID) db.ListSalesRow {
	return db.ListSalesRow{
		ID:          pgconv.ParseUUIDToPgType(id),
		TotalAmount: pgconv.Float64ToPgNumeric(1000.00),
		Status:      "paid",
	}
}

// ---------------------------------------------------------------------------
// DeleteSale
// ---------------------------------------------------------------------------

func TestDeleteSale_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	deletedBy := uuid.New()

	repo.EXPECT().
		DeleteSales(gomock.Any(), db.DeleteSaleParams{
			ID:        pgconv.ParseUUIDToPgType(id),
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
			DeletedBy: pgconv.ParseUUIDToPgType(deletedBy),
		}).
		Return(nil)

	err := svc.DeleteSale(context.Background(), id, domainDeleteSaleRequest(deletedBy, companyID))
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestDeleteSale_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		DeleteSales(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.DeleteSale(context.Background(), uuid.New(), domainDeleteSaleRequest(uuid.New(), uuid.New()))

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetSaleById
// ---------------------------------------------------------------------------

func TestGetSaleById_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	customerID := uuid.New()

	expected := buildDbSaleRow(id, companyID, customerID, uuid.New())

	repo.EXPECT().
		GetSaleById(gomock.Any(), db.GetSaleByIdParams{
			ID:        pgconv.ParseUUIDToPgType(id),
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
		}).
		Return(expected, nil)

	resp, err := svc.GetSaleById(context.Background(), domainGetSaleByIdRequest(id, companyID))
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
	if resp.CompanyID != companyID {
		t.Errorf("CompanyID incorreto")
	}
	if resp.CustomerName != "João Teste" {
		t.Errorf("CustomerName incorreto: '%s'", resp.CustomerName)
	}
}

func TestGetSaleById_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetSaleById(gomock.Any(), gomock.Any()).
		Return(db.GetSaleByIdRow{}, errDatabase)

	_, err := svc.GetSaleById(context.Background(), domainGetSaleByIdRequest(uuid.New(), uuid.New()))

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListSales
// ---------------------------------------------------------------------------

func TestListSales_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	dbSales := []db.ListSalesRow{
		buildDbListSalesRow(uuid.New()),
		buildDbListSalesRow(uuid.New()),
		buildDbListSalesRow(uuid.New()),
	}

	repo.EXPECT().
		ListSales(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(dbSales, nil)

	resp, err := svc.ListSales(context.Background(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 3 {
		t.Errorf("esperava 3 vendas, obteve %d", len(resp))
	}
}

func TestListSales_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListSales(gomock.Any(), gomock.Any()).
		Return([]db.ListSalesRow{}, nil)

	resp, err := svc.ListSales(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("esperava lista vazia, obteve %d", len(resp))
	}
}

func TestListSales_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListSales(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListSales(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// CountSales
// ---------------------------------------------------------------------------

func TestCountSales_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		CountSales(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(int64(10), nil)

	count, err := svc.CountSales(context.Background(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 10 {
		t.Errorf("esperava 10, obteve %d", count)
	}
}

func TestCountSales_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		CountSales(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	_, err := svc.CountSales(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetSalesPerformanceSummary
// ---------------------------------------------------------------------------

func TestGetSalesPerformanceSummary_PositiveGrowth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		GetSalesPerformanceSummary(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(db.GetSalesPerformanceSummaryRow{
			CurrentMonthCount: 120,
			LastMonthCount:    100,
		}, nil)

	// percentage = ((120 - 100) / 100) * 100 = 20%
	percentage, err := svc.GetSalesPerformanceSummary(context.Background(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 20.0 {
		t.Errorf("esperava 20%%, obteve %f%%", percentage)
	}
}

func TestGetSalesPerformanceSummary_NegativeGrowth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetSalesPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetSalesPerformanceSummaryRow{
			CurrentMonthCount: 60,
			LastMonthCount:    100,
		}, nil)

	// percentage = ((60 - 100) / 100) * 100 = -40%
	percentage, err := svc.GetSalesPerformanceSummary(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != -40.0 {
		t.Errorf("esperava -40%%, obteve %f%%", percentage)
	}
}

func TestGetSalesPerformanceSummary_NoLastMonth_HasCurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetSalesPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetSalesPerformanceSummaryRow{
			CurrentMonthCount: 30,
			LastMonthCount:    0,
		}, nil)

	// lastMonthCount == 0 e currentMonthCount > 0 → 100%
	percentage, err := svc.GetSalesPerformanceSummary(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 100.0 {
		t.Errorf("esperava 100%%, obteve %f%%", percentage)
	}
}

func TestGetSalesPerformanceSummary_BothZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetSalesPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetSalesPerformanceSummaryRow{
			CurrentMonthCount: 0,
			LastMonthCount:    0,
		}, nil)

	percentage, err := svc.GetSalesPerformanceSummary(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 0.0 {
		t.Errorf("esperava 0%%, obteve %f%%", percentage)
	}
}

func TestGetSalesPerformanceSummary_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetSalesPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetSalesPerformanceSummaryRow{}, errDatabase)

	_, err := svc.GetSalesPerformanceSummary(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetTotalAmountSummary
// ---------------------------------------------------------------------------

func TestGetTotalAmountSummary_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		GetTotalAmountSummary(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(db.GetTotalAmountSummaryRow{
			CurrentMonthSt: 10000.00,
			LastMonthSt:    8000.00,
		}, nil)

	resp, err := svc.GetTotalAmountSummary(context.Background(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.CurrentMonthSt != 10000.00 {
		t.Errorf("CurrentMonthSt incorreto: %f", resp.CurrentMonthSt)
	}
	if resp.LastMonthSt != 8000.00 {
		t.Errorf("LastMonthSt incorreto: %f", resp.LastMonthSt)
	}
}

func TestGetTotalAmountSummary_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetTotalAmountSummary(gomock.Any(), gomock.Any()).
		Return(db.GetTotalAmountSummaryRow{}, errDatabase)

	_, err := svc.GetTotalAmountSummary(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetTotalAmountIsPending / GetTotalAmountIsOverdue
// ---------------------------------------------------------------------------

func TestGetTotalAmountIsPending_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		GetTotalAmountByStatus(gomock.Any(), gomock.Any()).
		Return(3500.00, nil)

	total, err := svc.GetTotalAmountIsPending(context.Background(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if total != 3500.00 {
		t.Errorf("esperava 3500.00, obteve %f", total)
	}
}

func TestGetTotalAmountIsPending_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetTotalAmountByStatus(gomock.Any(), gomock.Any()).
		Return(0.0, errDatabase)

	_, err := svc.GetTotalAmountIsPending(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestGetTotalAmountIsOverdue_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		GetTotalAmountByStatus(gomock.Any(), gomock.Any()).
		Return(1200.00, nil)

	total, err := svc.GetTotalAmountIsOverdue(context.Background(), domainGetTotalAmountByStatusRequest(companyID))
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if total != 1200.00 {
		t.Errorf("esperava 1200.00, obteve %f", total)
	}
}

// ---------------------------------------------------------------------------
// ContSalesPendingAndOverdue
// ---------------------------------------------------------------------------

func TestContSalesPendingAndOverdue_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		ContSalesPendingAndOverdue(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(int64(5), nil)

	count, err := svc.ContSalesPendingAndOverdue(context.Background(), companyID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 5 {
		t.Errorf("esperava 5, obteve %d", count)
	}
}

func TestContSalesPendingAndOverdue_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ContSalesPendingAndOverdue(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	_, err := svc.ContSalesPendingAndOverdue(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListSalesWithDetailsPaginate
// ---------------------------------------------------------------------------

func TestListSalesWithDetailsPaginate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	pagination := salesDomain.PaginationParams{PaginationParams: globalDomain.PaginationParams{Page: 1, PerPage: 10}}

	repo.EXPECT().
		CountSalesByCompany(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(int64(2), nil)

	repo.EXPECT().
		CountSalesDeletedByCompany(gomock.Any(), gomock.Any()).
		Return(int64(0), nil)

	repo.EXPECT().
		GetTotalAmountPending(gomock.Any(), gomock.Any()).
		Return(500.00, nil)

	repo.EXPECT().
		GetTotalAmountPaid(gomock.Any(), gomock.Any()).
		Return(2000.00, nil)

	repo.EXPECT().
		ListSalesWithDetailsPaginate(gomock.Any(), db.ListSalesWithDetailsPaginateParams{
			CompanyID:        pgconv.ParseUUIDToPgType(companyID),
			Limit:            10,
			Offset:           0,
			Search:           "",
			SaleStatus:       nil,
			PaymentMethod:    nil,
			PaymentStartDate: pgtype.Date{},
			PaymentEndDate:   pgtype.Date{},
			SaleStartDate:    pgtype.Date{},
			SaleEndDate:      pgtype.Date{},
			SortBy:           "",
			OrderBy:          "",
		}).
		Return([]db.ListSalesWithDetailsPaginateRow{}, nil)

	resp, err := svc.ListSalesWithDetailsPaginate(context.Background(), companyID, pagination)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.SalesCount != 2 {
		t.Errorf("esperava SalesCount=2, obteve %d", resp.SalesCount)
	}
	if resp.TotalInvoiced != 2000.00 {
		t.Errorf("esperava TotalInvoiced=2000.00, obteve %f", resp.TotalInvoiced)
	}
	if resp.TotalPending != 500.00 {
		t.Errorf("esperava TotalPending=500.00, obteve %f", resp.TotalPending)
	}
}

func TestListSalesWithDetailsPaginate_CountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		CountSalesByCompany(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	_, err := svc.ListSalesWithDetailsPaginate(context.Background(), uuid.New(), salesDomain.PaginationParams{PaginationParams: globalDomain.PaginationParams{Page: 1, PerPage: 10}})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestListSalesWithDetailsPaginate_SecondPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	pagination := salesDomain.PaginationParams{PaginationParams: globalDomain.PaginationParams{Page: 3, PerPage: 5}}

	repo.EXPECT().
		CountSalesByCompany(gomock.Any(), gomock.Any()).
		Return(int64(20), nil)

	repo.EXPECT().
		CountSalesDeletedByCompany(gomock.Any(), gomock.Any()).
		Return(int64(2), nil)

	repo.EXPECT().
		GetTotalAmountPending(gomock.Any(), gomock.Any()).
		Return(100.00, nil)

	repo.EXPECT().
		GetTotalAmountPaid(gomock.Any(), gomock.Any()).
		Return(900.00, nil)

	repo.EXPECT().
		ListSalesWithDetailsPaginate(gomock.Any(), db.ListSalesWithDetailsPaginateParams{
			CompanyID:        pgconv.ParseUUIDToPgType(companyID),
			Limit:            5,
			Offset:           10, // (3-1) * 5 = 10
			Search:           "",
			SaleStatus:       nil,
			PaymentMethod:    nil,
			PaymentStartDate: pgtype.Date{},
			PaymentEndDate:   pgtype.Date{},
			SaleStartDate:    pgtype.Date{},
			SaleEndDate:      pgtype.Date{},
			SortBy:           "",
			OrderBy:          "",
		}).
		Return([]db.ListSalesWithDetailsPaginateRow{}, nil)

	resp, err := svc.ListSalesWithDetailsPaginate(context.Background(), companyID, pagination)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Page != 3 {
		t.Errorf("esperava Page=3, obteve %d", resp.Page)
	}
	if resp.TotalPages != 4 { // ceil(20/5) = 4
		t.Errorf("esperava TotalPages=4, obteve %d", resp.TotalPages)
	}
}

// ---------------------------------------------------------------------------
// UpdateSaleStatus
// ---------------------------------------------------------------------------

func TestUpdateSaleStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	customerID := uuid.New()

	currentSale := buildDbSaleRow(id, companyID, customerID, uuid.New())

	repo.EXPECT().
		GetSaleById(gomock.Any(), gomock.Any()).
		Return(currentSale, nil)

	repo.EXPECT().
		UpdateSaleStatus(gomock.Any(), gomock.Any()).
		Return(nil)

	err := svc.UpdateSaleStatus(context.Background(), id, domainUpdateSaleStatusRequest(id, companyID, "paid"))
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestUpdateSaleStatus_GetSaleError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetSaleById(gomock.Any(), gomock.Any()).
		Return(db.GetSaleByIdRow{}, errDatabase)

	err := svc.UpdateSaleStatus(context.Background(), uuid.New(), domainUpdateSaleStatusRequest(uuid.New(), uuid.New(), "paid"))

	if err == nil {
		t.Fatal("esperava erro ao buscar venda")
	}
}

func TestUpdateSaleStatus_UpdateRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	customerID := uuid.New()

	repo.EXPECT().
		GetSaleById(gomock.Any(), gomock.Any()).
		Return(buildDbSaleRow(id, companyID, customerID, uuid.New()), nil)

	repo.EXPECT().
		UpdateSaleStatus(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.UpdateSaleStatus(context.Background(), id, domainUpdateSaleStatusRequest(id, companyID, "paid"))

	if err == nil {
		t.Fatal("esperava erro do repositório ao atualizar status")
	}
}

// ---------------------------------------------------------------------------
// ListSalesByCustomerAndStatus
// ---------------------------------------------------------------------------

func TestListSalesByCustomerAndStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		ListSalesByCompanyAndStatus(gomock.Any(), gomock.Any()).
		Return([]db.ListSalesByCompanyAndStatusRow{
			{CustomerName: "Ana", Status: "pending"},
			{CustomerName: "Bruno", Status: "pending"},
		}, nil)

	resp, err := svc.ListSalesByCustomerAndStatus(context.Background(), domainListSalesByCompanyAndStatusRequest(companyID, "pending"))
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("esperava 2 vendas, obteve %d", len(resp))
	}
}

func TestListSalesByCustomerAndStatus_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListSalesByCompanyAndStatus(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListSalesByCustomerAndStatus(context.Background(), domainListSalesByCompanyAndStatusRequest(uuid.New(), "pending"))

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// Helpers de domínio para construção de requests nos testes
// ---------------------------------------------------------------------------

func domainDeleteSaleRequest(deletedBy, companyID uuid.UUID) salesDomain.DeleteSaleRequest {
	return salesDomain.DeleteSaleRequest{
		DeletedBy: deletedBy,
		CompanyID: companyID,
	}
}

func domainGetSaleByIdRequest(id, companyID uuid.UUID) salesDomain.GetSaleByIdRequest {
	return salesDomain.GetSaleByIdRequest{
		ID:        id,
		CompanyID: companyID,
	}
}

func domainUpdateSaleStatusRequest(id, companyID uuid.UUID, status string) salesDomain.UpdateSaleStatusRequest {
	return salesDomain.UpdateSaleStatusRequest{
		ID:        id,
		CompanyID: companyID,
		Status:    status,
	}
}

func domainGetTotalAmountByStatusRequest(companyID uuid.UUID) salesDomain.GetTotalAmountByStatusRequest {
	return salesDomain.GetTotalAmountByStatusRequest{
		CompanyID: companyID,
	}
}

func domainListSalesByCompanyAndStatusRequest(companyID uuid.UUID, status string) salesDomain.ListSalesByCompanyAndStatusRequest {
	return salesDomain.ListSalesByCompanyAndStatusRequest{
		CompanyID: companyID,
		Status:    status,
	}
}
