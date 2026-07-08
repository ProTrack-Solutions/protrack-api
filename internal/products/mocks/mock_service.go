// Package mocks fornece interfaces de mock para uso nos testes de serviços de products.
// Este arquivo define o MockServiceInterface utilizado pelos testes de handler.
package mocks

import (
	"context"
	"reflect"
	"time"

	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/google/uuid"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"go.uber.org/mock/gomock"
)

// ServiceInterface define o contrato de serviço utilizado pelo handler.
// Deve ser mantido em sincronia com os métodos expostos por service.Service.
type ServiceInterface interface {
	CreateProduct(ctx context.Context, req domain.CreateProductRequest) (domain.ProductResponse, error)
	DeleteProduct(ctx context.Context, req domain.DeleteProductRequest) error
	GetProductByBarcode(ctx context.Context, barcode string) (domain.ProductResponse, error)
	GetProductById(ctx context.Context, id uuid.UUID) (domain.ProductResponse, error)
	GetProductByIdTx(ctx context.Context, tx db.DBTX, id uuid.UUID) (domain.ProductResponse, error)
	ListProductsByCategoryId(ctx context.Context, categoryId, companyID uuid.UUID) ([]domain.ProductResponse, error)
	ListProductsByCompany(ctx context.Context, companyId uuid.UUID) ([]domain.ListProductsByCompanyRow, error)
	ListProductsByCompanyPaginated(ctx context.Context, companyId uuid.UUID, pagination globalDomain.PaginationParams) (domain.ProductPaginatedResponse, error)
	UpdateProduct(ctx context.Context, id uuid.UUID, req domain.UpdateProductRequest) (domain.ProductResponse, error)
	DecrementStock(ctx context.Context, req domain.DecrementStockRequest) error
	CountProducts(ctx context.Context, companyId uuid.UUID) (int64, error)
	GetProductsPerformanceSummary(ctx context.Context, companyId uuid.UUID) (float64, error)
	GetCostTotalStock(ctx context.Context, companyId uuid.UUID) (float64, error)
	GetTop5BestSellingProducts(ctx context.Context, companyId uuid.UUID) ([]domain.GetTop5BestSellingProductsRow, error)
	GetInventoryReport(ctx context.Context, companyId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.GetInventoryReportResponse, error)
	ListProductsByDate(ctx context.Context, companyId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.ListProductsByDateResponse, error)
	ListProductBuCategoryIdAndDate(ctx context.Context, categoryId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.ListProductsByCategoryAndDateResponse, error)
}

// MockServiceInterface é o mock da interface de serviço utilizado pelos testes de handler.
type MockServiceInterface struct {
	ctrl     *gomock.Controller
	recorder *MockServiceInterfaceMockRecorder
}

// MockServiceInterfaceMockRecorder é o recorder de chamadas do mock de serviço.
type MockServiceInterfaceMockRecorder struct {
	mock *MockServiceInterface
}

// NewMockServiceInterface cria uma nova instância de MockServiceInterface.
func NewMockServiceInterface(ctrl *gomock.Controller) *MockServiceInterface {
	mock := &MockServiceInterface{ctrl: ctrl}
	mock.recorder = &MockServiceInterfaceMockRecorder{mock}
	return mock
}

// EXPECT retorna o recorder para configurar expectativas.
func (m *MockServiceInterface) EXPECT() *MockServiceInterfaceMockRecorder {
	return m.recorder
}

func (m *MockServiceInterface) CreateProduct(ctx context.Context, req domain.CreateProductRequest) (domain.ProductResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateProduct", ctx, req)
	ret0, _ := ret[0].(domain.ProductResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) CreateProduct(ctx, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateProduct", reflect.TypeOf((*MockServiceInterface)(nil).CreateProduct), ctx, req)
}

func (m *MockServiceInterface) DeleteProduct(ctx context.Context, req domain.DeleteProductRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteProduct", ctx, req)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockServiceInterfaceMockRecorder) DeleteProduct(ctx, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteProduct", reflect.TypeOf((*MockServiceInterface)(nil).DeleteProduct), ctx, req)
}

func (m *MockServiceInterface) GetProductByBarcode(ctx context.Context, barcode string) (domain.ProductResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProductByBarcode", ctx, barcode)
	ret0, _ := ret[0].(domain.ProductResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) GetProductByBarcode(ctx, barcode any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProductByBarcode", reflect.TypeOf((*MockServiceInterface)(nil).GetProductByBarcode), ctx, barcode)
}

func (m *MockServiceInterface) GetProductById(ctx context.Context, id uuid.UUID) (domain.ProductResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProductById", ctx, id)
	ret0, _ := ret[0].(domain.ProductResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) GetProductById(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProductById", reflect.TypeOf((*MockServiceInterface)(nil).GetProductById), ctx, id)
}

func (m *MockServiceInterface) GetProductByIdTx(ctx context.Context, tx db.DBTX, id uuid.UUID) (domain.ProductResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProductByIdTx", ctx, tx, id)
	ret0, _ := ret[0].(domain.ProductResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) GetProductByIdTx(ctx, tx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProductByIdTx", reflect.TypeOf((*MockServiceInterface)(nil).GetProductByIdTx), ctx, tx, id)
}

func (m *MockServiceInterface) ListProductsByCategoryId(ctx context.Context, categoryId, companyID uuid.UUID) ([]domain.ProductResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListProductsByCategoryId", ctx, categoryId, companyID)
	ret0, _ := ret[0].([]domain.ProductResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) ListProductsByCategoryId(ctx, categoryId, companyID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListProductsByCategoryId", reflect.TypeOf((*MockServiceInterface)(nil).ListProductsByCategoryId), ctx, categoryId, companyID)
}

func (m *MockServiceInterface) ListProductsByCompany(ctx context.Context, companyId uuid.UUID) ([]domain.ListProductsByCompanyRow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListProductsByCompany", ctx, companyId)
	ret0, _ := ret[0].([]domain.ListProductsByCompanyRow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) ListProductsByCompany(ctx, companyId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListProductsByCompany", reflect.TypeOf((*MockServiceInterface)(nil).ListProductsByCompany), ctx, companyId)
}

func (m *MockServiceInterface) ListProductsByCompanyPaginated(ctx context.Context, companyId uuid.UUID, pagination globalDomain.PaginationParams) (domain.ProductPaginatedResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListProductsByCompanyPaginated", ctx, companyId, pagination)
	ret0, _ := ret[0].(domain.ProductPaginatedResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) ListProductsByCompanyPaginated(ctx, companyId, pagination any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListProductsByCompanyPaginated", reflect.TypeOf((*MockServiceInterface)(nil).ListProductsByCompanyPaginated), ctx, companyId, pagination)
}

func (m *MockServiceInterface) UpdateProduct(ctx context.Context, id uuid.UUID, req domain.UpdateProductRequest) (domain.ProductResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateProduct", ctx, id, req)
	ret0, _ := ret[0].(domain.ProductResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) UpdateProduct(ctx, id, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateProduct", reflect.TypeOf((*MockServiceInterface)(nil).UpdateProduct), ctx, id, req)
}

func (m *MockServiceInterface) DecrementStock(ctx context.Context, req domain.DecrementStockRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DecrementStock", ctx, req)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockServiceInterfaceMockRecorder) DecrementStock(ctx, req any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DecrementStock", reflect.TypeOf((*MockServiceInterface)(nil).DecrementStock), ctx, req)
}

func (m *MockServiceInterface) CountProducts(ctx context.Context, companyId uuid.UUID) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CountProducts", ctx, companyId)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) CountProducts(ctx, companyId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CountProducts", reflect.TypeOf((*MockServiceInterface)(nil).CountProducts), ctx, companyId)
}

func (m *MockServiceInterface) GetProductsPerformanceSummary(ctx context.Context, companyId uuid.UUID) (float64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProductsPerformanceSummary", ctx, companyId)
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) GetProductsPerformanceSummary(ctx, companyId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProductsPerformanceSummary", reflect.TypeOf((*MockServiceInterface)(nil).GetProductsPerformanceSummary), ctx, companyId)
}

func (m *MockServiceInterface) GetCostTotalStock(ctx context.Context, companyId uuid.UUID) (float64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCostTotalStock", ctx, companyId)
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) GetCostTotalStock(ctx, companyId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCostTotalStock", reflect.TypeOf((*MockServiceInterface)(nil).GetCostTotalStock), ctx, companyId)
}

func (m *MockServiceInterface) GetTop5BestSellingProducts(ctx context.Context, companyId uuid.UUID) ([]domain.GetTop5BestSellingProductsRow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTop5BestSellingProducts", ctx, companyId)
	ret0, _ := ret[0].([]domain.GetTop5BestSellingProductsRow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) GetTop5BestSellingProducts(ctx, companyId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTop5BestSellingProducts", reflect.TypeOf((*MockServiceInterface)(nil).GetTop5BestSellingProducts), ctx, companyId)
}

func (m *MockServiceInterface) GetInventoryReport(ctx context.Context, companyId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.GetInventoryReportResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInventoryReport", ctx, companyId, startAt, startAt_2)
	ret0, _ := ret[0].([]domain.GetInventoryReportResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) GetInventoryReport(ctx, companyId, startAt, startAt_2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInventoryReport", reflect.TypeOf((*MockServiceInterface)(nil).GetInventoryReport), ctx, companyId, startAt, startAt_2)
}

func (m *MockServiceInterface) ListProductsByDate(ctx context.Context, companyId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.ListProductsByDateResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListProductsByDate", ctx, companyId, startAt, startAt_2)
	ret0, _ := ret[0].([]domain.ListProductsByDateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) ListProductsByDate(ctx, companyId, startAt, startAt_2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListProductsByDate", reflect.TypeOf((*MockServiceInterface)(nil).ListProductsByDate), ctx, companyId, startAt, startAt_2)
}

func (m *MockServiceInterface) ListProductBuCategoryIdAndDate(ctx context.Context, categoryId uuid.UUID, startAt time.Time, startAt_2 time.Time) ([]domain.ListProductsByCategoryAndDateResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListProductBuCategoryIdAndDate", ctx, categoryId, startAt, startAt_2)
	ret0, _ := ret[0].([]domain.ListProductsByCategoryAndDateResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockServiceInterfaceMockRecorder) ListProductBuCategoryIdAndDate(ctx, categoryId, startAt, startAt_2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListProductBuCategoryIdAndDate", reflect.TypeOf((*MockServiceInterface)(nil).ListProductBuCategoryIdAndDate), ctx, categoryId, startAt, startAt_2)
}
