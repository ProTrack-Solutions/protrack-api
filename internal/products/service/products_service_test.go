package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Fixtures e helpers
// ---------------------------------------------------------------------------

// newService cria um novo Service injetando o mock de repositório via NewServiceWithRepo.
func newService(t *testing.T, repo *mocks.MockRepositoryInterface) *service.Service {
	t.Helper()
	return service.NewServiceWithRepo(repo)
}

// buildDbProduct cria uma db.Product de exemplo totalmente preenchido.
func buildDbProduct(id, companyID, categoryID, createdBy uuid.UUID) db.Product {
	now := time.Now().UTC()
	return db.Product{
		ID:          pgconv.ParseUUIDToPgType(id),
		CompanyID:   pgconv.ParseUUIDToPgType(companyID),
		CategoryID:  pgconv.ParseUUIDToPgType(categoryID),
		Name:        "Camiseta Polo",
		Description: pgconv.ParseStringToPgText("Camiseta de algodão"),
		Barcode:     pgconv.ParseStringToPgText("BAR0001"),
		Quantity:    50,
		Size:        pgconv.ParseStringToPgText("M"),
		CostPrice:   pgconv.Float64ToPgNumeric(30.00),
		SalePrice:   pgconv.Float64ToPgNumeric(79.90),
		CreatedBy:   pgconv.ParseUUIDToPgType(createdBy),
		UpdatedBy:   pgconv.ParseUUIDToPgType(uuid.Nil),
		DeletedBy:   pgconv.ParseUUIDToPgType(uuid.Nil),
		CreatedAt:   pgconv.TimeToPgTimestamptz(now),
		UpdatedAt:   pgconv.TimeToPgTimestamptz(now),
		DeletedAt:   pgtype.Timestamptz{Valid: false},
	}
}

// buildDbProductCompanyRow cria uma db.ListProductsByCompanyRow de exemplo.
func buildDbProductCompanyRow(id, companyID, categoryID uuid.UUID) db.ListProductsByCompanyRow {
	now := time.Now().UTC()
	return db.ListProductsByCompanyRow{
		ID:           pgconv.ParseUUIDToPgType(id),
		CompanyID:    pgconv.ParseUUIDToPgType(companyID),
		CategoryID:   pgconv.ParseUUIDToPgType(categoryID),
		Name:         "Calça Jeans",
		Description:  pgconv.ParseStringToPgText("Calça slim"),
		Barcode:      pgconv.ParseStringToPgText("BAR0002"),
		Quantity:     20,
		Size:         pgconv.ParseStringToPgText("42"),
		CostPrice:    pgconv.Float64ToPgNumeric(45.00),
		SalePrice:    pgconv.Float64ToPgNumeric(99.90),
		CreatedBy:    pgconv.ParseUUIDToPgType(uuid.New()),
		UpdatedBy:    pgconv.ParseUUIDToPgType(uuid.Nil),
		DeletedBy:    pgconv.ParseUUIDToPgType(uuid.Nil),
		CreatedAt:    pgconv.TimeToPgTimestamptz(now),
		UpdatedAt:    pgconv.TimeToPgTimestamptz(now),
		DeletedAt:    pgtype.Timestamptz{Valid: false},
		CategoryName: "Roupas",
	}
}

// buildDbProductPaginatedRow cria uma db.ListProductsByCompanyPaginatedRow de exemplo.
func buildDbProductPaginatedRow(id, companyID, categoryID uuid.UUID, qty int32) db.ListProductsByCompanyPaginatedRow {
	now := time.Now().UTC()
	return db.ListProductsByCompanyPaginatedRow{
		ID:           pgconv.ParseUUIDToPgType(id),
		CompanyID:    pgconv.ParseUUIDToPgType(companyID),
		CategoryID:   pgconv.ParseUUIDToPgType(categoryID),
		Name:         "Produto Paginado",
		Description:  pgconv.ParseStringToPgText("Desc paginada"),
		Barcode:      pgconv.ParseStringToPgText("BAR0003"),
		Quantity:     qty,
		Size:         pgconv.ParseStringToPgText("G"),
		CostPrice:    pgconv.Float64ToPgNumeric(20.00),
		SalePrice:    pgconv.Float64ToPgNumeric(40.00),
		CreatedBy:    pgconv.ParseUUIDToPgType(uuid.New()),
		UpdatedBy:    pgconv.ParseUUIDToPgType(uuid.Nil),
		DeletedBy:    pgconv.ParseUUIDToPgType(uuid.Nil),
		CreatedAt:    pgconv.TimeToPgTimestamptz(now),
		UpdatedAt:    pgconv.TimeToPgTimestamptz(now),
		DeletedAt:    pgtype.Timestamptz{Valid: false},
		CategoryName: "Eletrônicos",
	}
}

var errDatabase = errors.New("database error")

// ---------------------------------------------------------------------------
// CreateProduct
// ---------------------------------------------------------------------------

func TestCreateProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()
	categoryID := uuid.New()
	createdBy := uuid.New()
	productID := uuid.New()

	expectedProduct := buildDbProduct(productID, companyID, categoryID, createdBy)

	repo.EXPECT().
		CreateProduct(gomock.Any(), gomock.Any()).
		Return(expectedProduct, nil)

	req := domain.CreateProductRequest{
		CompanyID:   companyID,
		Name:        "Camiseta Polo",
		Description: "Camiseta de algodão",
		CategoryID:  categoryID,
		Barcode:     "BAR0001",
		Quantity:    50,
		Size:        "M",
		CostPrice:   30.00,
		SalePrice:   79.90,
		CreatedBy:   createdBy,
	}

	resp, err := svc.CreateProduct(context.Background(), req)

	if err != nil {
		t.Fatalf("esperava sem erro, obteve: %v", err)
	}
	if resp.Name != "Camiseta Polo" {
		t.Errorf("esperava Name='Camiseta Polo', obteve '%s'", resp.Name)
	}
	if resp.CompanyID != companyID {
		t.Errorf("CompanyID não confere")
	}
	if resp.CostPrice != 30.00 {
		t.Errorf("esperava CostPrice=30.00, obteve %f", resp.CostPrice)
	}
	if resp.SalePrice != 79.90 {
		t.Errorf("esperava SalePrice=79.90, obteve %f", resp.SalePrice)
	}
}

func TestCreateProduct_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		CreateProduct(gomock.Any(), gomock.Any()).
		Return(db.Product{}, errDatabase)

	_, err := svc.CreateProduct(context.Background(), domain.CreateProductRequest{
		CompanyID: uuid.New(),
		Name:      "X",
	})

	if err == nil {
		t.Fatal("esperava erro do repositório, mas não houve")
	}
}

// ---------------------------------------------------------------------------
// DeleteProduct
// ---------------------------------------------------------------------------

func TestDeleteProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	productID := uuid.New()
	deletedBy := uuid.New()

	repo.EXPECT().
		DeleteProduct(gomock.Any(), db.DeleteProductParams{
			ID:        pgconv.ParseUUIDToPgType(productID),
			DeletedBy: pgconv.ParseUUIDToPgType(deletedBy),
		}).
		Return(nil)

	err := svc.DeleteProduct(context.Background(), domain.DeleteProductRequest{
		ID:        productID,
		DeletedBy: deletedBy,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestDeleteProduct_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		DeleteProduct(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.DeleteProduct(context.Background(), domain.DeleteProductRequest{
		ID:        uuid.New(),
		DeletedBy: uuid.New(),
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetProductByBarcode
// ---------------------------------------------------------------------------

func TestGetProductByBarcode_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	categoryID := uuid.New()
	createdBy := uuid.New()

	expectedProduct := buildDbProduct(id, companyID, categoryID, createdBy)
	barcode := "BAR0001"

	repo.EXPECT().
		GetProductByBarcode(gomock.Any(), pgconv.ParseStringToPgText(barcode)).
		Return(expectedProduct, nil)

	resp, err := svc.GetProductByBarcode(context.Background(), barcode)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Barcode != barcode {
		t.Errorf("esperava Barcode='%s', obteve '%s'", barcode, resp.Barcode)
	}
	if resp.ID != id {
		t.Errorf("ID não confere")
	}
}

func TestGetProductByBarcode_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetProductByBarcode(gomock.Any(), gomock.Any()).
		Return(db.Product{}, errors.New("no rows in result set"))

	_, err := svc.GetProductByBarcode(context.Background(), "INEXISTENTE")

	if err == nil {
		t.Fatal("esperava erro de produto não encontrado")
	}
}

// ---------------------------------------------------------------------------
// GetProductById
// ---------------------------------------------------------------------------

func TestGetProductById_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	categoryID := uuid.New()
	createdBy := uuid.New()

	expectedProduct := buildDbProduct(id, companyID, categoryID, createdBy)

	repo.EXPECT().
		GetProductById(gomock.Any(), pgconv.ParseUUIDToPgType(id)).
		Return(expectedProduct, nil)

	resp, err := svc.GetProductById(context.Background(), id)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID não confere")
	}
	if resp.Name != "Camiseta Polo" {
		t.Errorf("Name incorreto: '%s'", resp.Name)
	}
}

func TestGetProductById_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetProductById(gomock.Any(), gomock.Any()).
		Return(db.Product{}, errDatabase)

	_, err := svc.GetProductById(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListProductsByCategoryId
// ---------------------------------------------------------------------------

func TestListProductsByCategoryId_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	categoryID := uuid.New()
	companyID := uuid.New()
	productID := uuid.New()
	createdBy := uuid.New()

	dbProducts := []db.Product{
		buildDbProduct(productID, companyID, categoryID, createdBy),
	}

	repo.EXPECT().
		ListProductsByCategoryId(gomock.Any(), db.ListProductsByCategoryIdParams{
			CategoryID: pgconv.ParseUUIDToPgType(categoryID),
			CompanyID:  pgconv.ParseUUIDToPgType(companyID),
		}).
		Return(dbProducts, nil)

	resp, err := svc.ListProductsByCategoryId(context.Background(), categoryID, companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("esperava 1 produto, obteve %d", len(resp))
	}
	if resp[0].ID != productID {
		t.Errorf("ID do produto incorreto")
	}
}

func TestListProductsByCategoryId_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		ListProductsByCategoryId(gomock.Any(), gomock.Any()).
		Return([]db.Product{}, nil)

	resp, err := svc.ListProductsByCategoryId(context.Background(), uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("esperava lista vazia, obteve %d itens", len(resp))
	}
}

func TestListProductsByCategoryId_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		ListProductsByCategoryId(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListProductsByCategoryId(context.Background(), uuid.New(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListProductsByCompany
// ---------------------------------------------------------------------------

func TestListProductsByCompany_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()
	categoryID := uuid.New()

	dbRows := []db.ListProductsByCompanyRow{
		buildDbProductCompanyRow(uuid.New(), companyID, categoryID),
		buildDbProductCompanyRow(uuid.New(), companyID, categoryID),
	}

	repo.EXPECT().
		ListProductsByCompany(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(dbRows, nil)

	resp, err := svc.ListProductsByCompany(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("esperava 2 produtos, obteve %d", len(resp))
	}
	for _, p := range resp {
		if p.CategoryName != "Roupas" {
			t.Errorf("CategoryName incorreto: '%s'", p.CategoryName)
		}
	}
}

func TestListProductsByCompany_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		ListProductsByCompany(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListProductsByCompany(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListProductsByCompanyPaginated
// ---------------------------------------------------------------------------

func TestListProductsByCompanyPaginated_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()
	categoryID := uuid.New()
	pagination := globalDomain.PaginationParams{Page: 1, PerPage: 10}

	// Produto com estoque alto
	row1 := buildDbProductPaginatedRow(uuid.New(), companyID, categoryID, 10)
	// Produto com estoque baixo (< 5)
	row2 := buildDbProductPaginatedRow(uuid.New(), companyID, categoryID, 3)

	dbRows := []db.ListProductsByCompanyPaginatedRow{row1, row2}

	repo.EXPECT().
		CountProductsByCompany(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(int64(2), nil)

	repo.EXPECT().
		ListProductsByCompanyPaginated(gomock.Any(), db.ListProductsByCompanyPaginatedParams{
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
			Limit:     10,
			Offset:    0,
		}).
		Return(dbRows, nil)

	resp, err := svc.ListProductsByCompanyPaginated(context.Background(), companyID, pagination)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("esperava 2 produtos, obteve %d", len(resp.Data))
	}
	if resp.TotalRows != 2 {
		t.Errorf("esperava TotalRows=2, obteve %d", resp.TotalRows)
	}
	if resp.LowItensInStock != 1 {
		t.Errorf("esperava LowItensInStock=1, obteve %d", resp.LowItensInStock)
	}
	// itensInStock = 10 + 3 = 13
	if resp.ItensInStock != 13 {
		t.Errorf("esperava ItensInStock=13, obteve %d", resp.ItensInStock)
	}
	// totalValueInStock = (20.00 * 10) + (20.00 * 3) = 260.00
	expectedTotal := 20.00*10 + 20.00*3
	if resp.TotalValueInStock != expectedTotal {
		t.Errorf("esperava TotalValueInStock=%f, obteve %f", expectedTotal, resp.TotalValueInStock)
	}
}

func TestListProductsByCompanyPaginated_CountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		CountProductsByCompany(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	_, err := svc.ListProductsByCompanyPaginated(context.Background(), uuid.New(), globalDomain.PaginationParams{Page: 1, PerPage: 10})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestListProductsByCompanyPaginated_ListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		CountProductsByCompany(gomock.Any(), gomock.Any()).
		Return(int64(5), nil)

	repo.EXPECT().
		ListProductsByCompanyPaginated(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListProductsByCompanyPaginated(context.Background(), uuid.New(), globalDomain.PaginationParams{Page: 1, PerPage: 10})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestListProductsByCompanyPaginated_SecondPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()
	categoryID := uuid.New()
	pagination := globalDomain.PaginationParams{Page: 2, PerPage: 5}

	repo.EXPECT().
		CountProductsByCompany(gomock.Any(), gomock.Any()).
		Return(int64(12), nil)

	repo.EXPECT().
		ListProductsByCompanyPaginated(gomock.Any(), db.ListProductsByCompanyPaginatedParams{
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
			Limit:     5,
			Offset:    5, // (page-1) * perPage = (2-1) * 5
		}).
		Return([]db.ListProductsByCompanyPaginatedRow{
			buildDbProductPaginatedRow(uuid.New(), companyID, categoryID, 8),
		}, nil)

	resp, err := svc.ListProductsByCompanyPaginated(context.Background(), companyID, pagination)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Page != 2 {
		t.Errorf("esperava Page=2, obteve %d", resp.Page)
	}
	if resp.TotalPages != 3 { // ceil(12/5) = 3
		t.Errorf("esperava TotalPages=3, obteve %d", resp.TotalPages)
	}
}

// ---------------------------------------------------------------------------
// UpdateProduct
// ---------------------------------------------------------------------------

func TestUpdateProduct_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	categoryID := uuid.New()
	createdBy := uuid.New()
	updatedBy := uuid.New()

	currentProduct := buildDbProduct(id, companyID, categoryID, createdBy)
	updatedProduct := currentProduct
	updatedProduct.Name = "Nome Atualizado"
	updatedProduct.UpdatedBy = pgconv.ParseUUIDToPgType(updatedBy)

	repo.EXPECT().
		GetProductById(gomock.Any(), pgconv.ParseUUIDToPgType(id)).
		Return(currentProduct, nil)

	repo.EXPECT().
		UpdateProduct(gomock.Any(), gomock.Any()).
		Return(updatedProduct, nil)

	resp, err := svc.UpdateProduct(context.Background(), id, domain.UpdateProductRequest{
		Name:      "Nome Atualizado",
		UpdatedBy: updatedBy,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Name != "Nome Atualizado" {
		t.Errorf("esperava Name='Nome Atualizado', obteve '%s'", resp.Name)
	}
}

func TestUpdateProduct_GetByIdError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetProductById(gomock.Any(), gomock.Any()).
		Return(db.Product{}, errors.New("not found"))

	_, err := svc.UpdateProduct(context.Background(), uuid.New(), domain.UpdateProductRequest{
		Name: "Qualquer",
	})

	if err == nil {
		t.Fatal("esperava erro ao buscar produto")
	}
}

func TestUpdateProduct_UpdateRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	id := uuid.New()
	repo.EXPECT().
		GetProductById(gomock.Any(), gomock.Any()).
		Return(buildDbProduct(id, uuid.New(), uuid.New(), uuid.New()), nil)

	repo.EXPECT().
		UpdateProduct(gomock.Any(), gomock.Any()).
		Return(db.Product{}, errDatabase)

	_, err := svc.UpdateProduct(context.Background(), id, domain.UpdateProductRequest{Name: "Teste"})

	if err == nil {
		t.Fatal("esperava erro do repositório ao atualizar")
	}
}

// ---------------------------------------------------------------------------
// DecrementStock
// ---------------------------------------------------------------------------

func TestDecrementStock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	productID := uuid.New()

	repo.EXPECT().
		DecrementStock(gomock.Any(), db.DecrementStockParams{
			ID:       pgconv.ParseUUIDToPgType(productID),
			Quantity: 3,
		}).
		Return(nil)

	err := svc.DecrementStock(context.Background(), domain.DecrementStockRequest{
		ID:       productID,
		Quantity: 3,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestDecrementStock_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		DecrementStock(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.DecrementStock(context.Background(), domain.DecrementStockRequest{
		ID:       uuid.New(),
		Quantity: 1,
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// CountProducts
// ---------------------------------------------------------------------------

func TestCountProducts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		CountProducts(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(int64(42), nil)

	count, err := svc.CountProducts(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 42 {
		t.Errorf("esperava 42, obteve %d", count)
	}
}

func TestCountProducts_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		CountProducts(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	_, err := svc.CountProducts(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetProductsPerformanceSummary
// ---------------------------------------------------------------------------

func TestGetProductsPerformanceSummary_PositiveGrowth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		GetProductsPerformanceSummary(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(db.GetProductsPerformanceSummaryRow{
			CurrentMonthQty: 150,
			LastMonthQty:    100,
		}, nil)

	// percentage = ((150 - 100) / 100) * 100 = 50%
	percentage, err := svc.GetProductsPerformanceSummary(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 50.0 {
		t.Errorf("esperava 50%%, obteve %f%%", percentage)
	}
}

func TestGetProductsPerformanceSummary_NegativeGrowth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		GetProductsPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetProductsPerformanceSummaryRow{
			CurrentMonthQty: 80,
			LastMonthQty:    100,
		}, nil)

	// percentage = ((80 - 100) / 100) * 100 = -20%
	percentage, err := svc.GetProductsPerformanceSummary(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != -20.0 {
		t.Errorf("esperava -20%%, obteve %f%%", percentage)
	}
}

func TestGetProductsPerformanceSummary_NoLastMonth_HasCurrentMonth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetProductsPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetProductsPerformanceSummaryRow{
			CurrentMonthQty: 50,
			LastMonthQty:    0,
		}, nil)

	// lastMonthQty == 0 e currentMonthQty > 0 → 100%
	percentage, err := svc.GetProductsPerformanceSummary(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 100.0 {
		t.Errorf("esperava 100%%, obteve %f%%", percentage)
	}
}

func TestGetProductsPerformanceSummary_BothZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetProductsPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetProductsPerformanceSummaryRow{
			CurrentMonthQty: 0,
			LastMonthQty:    0,
		}, nil)

	percentage, err := svc.GetProductsPerformanceSummary(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 0.0 {
		t.Errorf("esperava 0%%, obteve %f%%", percentage)
	}
}

func TestGetProductsPerformanceSummary_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetProductsPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetProductsPerformanceSummaryRow{}, errDatabase)

	_, err := svc.GetProductsPerformanceSummary(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetCostTotalStock
// ---------------------------------------------------------------------------

func TestGetCostTotalStock_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		GetCostTotalStock(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(5000.50, nil)

	total, err := svc.GetCostTotalStock(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if total != 5000.50 {
		t.Errorf("esperava 5000.50, obteve %f", total)
	}
}

func TestGetCostTotalStock_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetCostTotalStock(gomock.Any(), gomock.Any()).
		Return(float64(0), errDatabase)

	_, err := svc.GetCostTotalStock(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetTop5BestSellingProducts
// ---------------------------------------------------------------------------

func TestGetTop5BestSellingProducts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()

	dbRows := []db.GetTop5BestSellingProductsRow{
		{ID: pgconv.ParseUUIDToPgType(productID1), Name: "Produto A", TotalQuantitySold: 200},
		{ID: pgconv.ParseUUIDToPgType(productID2), Name: "Produto B", TotalQuantitySold: 150},
	}

	repo.EXPECT().
		GetTop5BestSellingProducts(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(dbRows, nil)

	resp, err := svc.GetTop5BestSellingProducts(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("esperava 2 produtos, obteve %d", len(resp))
	}
	if resp[0].Name != "Produto A" {
		t.Errorf("esperava 'Produto A', obteve '%s'", resp[0].Name)
	}
	if resp[0].TotalQuantitySold != 200 {
		t.Errorf("esperava 200, obteve %d", resp[0].TotalQuantitySold)
	}
}

func TestGetTop5BestSellingProducts_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetTop5BestSellingProducts(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.GetTop5BestSellingProducts(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetInventoryReport
// ---------------------------------------------------------------------------

func TestGetInventoryReport_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()
	startAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	now := time.Now().UTC()

	dbRows := []db.GetInventoryReportRow{
		{
			Name:         "Produto Inventário",
			CategoryName: pgconv.ParseStringToPgText("Categoria A"),
			Quantity:     30,
			SalePrice:    pgconv.Float64ToPgNumeric(50.00),
			TotalValue:   pgconv.Float64ToPgNumeric(1500.00),
			CostPrice:    pgconv.Float64ToPgNumeric(25.00),
			Barcode:      pgconv.ParseStringToPgText("INV001"),
			CreatedAt:    pgconv.TimeToPgTimestamptz(now),
		},
	}

	repo.EXPECT().
		GetInventoryReport(gomock.Any(), db.GetInventoryReportParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyID),
			CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
		}).
		Return(dbRows, nil)

	resp, err := svc.GetInventoryReport(context.Background(), companyID, startAt, endAt)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("esperava 1 item, obteve %d", len(resp))
	}
	if resp[0].Name != "Produto Inventário" {
		t.Errorf("Name incorreto: '%s'", resp[0].Name)
	}
	if resp[0].Barcode != "INV001" {
		t.Errorf("Barcode incorreto: '%s'", resp[0].Barcode)
	}
}

func TestGetInventoryReport_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetInventoryReport(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	startAt := time.Now()
	_, err := svc.GetInventoryReport(context.Background(), uuid.New(), startAt, startAt)

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListProductsByDate
// ---------------------------------------------------------------------------

func TestListProductsByDate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	companyID := uuid.New()
	categoryID := uuid.New()
	startAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	now := time.Now().UTC()

	dbRows := []db.ListProductsByDateRow{
		{
			ID:           pgconv.ParseUUIDToPgType(uuid.New()),
			Name:         "Produto Data",
			CostPrice:    pgconv.Float64ToPgNumeric(40.00),
			Quantity:     15,
			CategoryID:   pgconv.ParseUUIDToPgType(categoryID),
			CreatedAt:    pgconv.TimeToPgTimestamptz(now),
			CategoryName: "Data Categoria",
		},
	}

	repo.EXPECT().
		ListProductsByDate(gomock.Any(), db.ListProductsByDateParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyID),
			CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
		}).
		Return(dbRows, nil)

	resp, err := svc.ListProductsByDate(context.Background(), companyID, startAt, endAt)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("esperava 1 produto, obteve %d", len(resp))
	}
	if resp[0].Name != "Produto Data" {
		t.Errorf("Name incorreto: '%s'", resp[0].Name)
	}
	if resp[0].CategoryName != "Data Categoria" {
		t.Errorf("CategoryName incorreto: '%s'", resp[0].CategoryName)
	}
}

func TestListProductsByDate_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		ListProductsByDate(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	startAt := time.Now()
	_, err := svc.ListProductsByDate(context.Background(), uuid.New(), startAt, startAt)

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListProductBuCategoryIdAndDate
// ---------------------------------------------------------------------------

func TestListProductBuCategoryIdAndDate_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	categoryID := uuid.New()
	startAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	now := time.Now().UTC()

	dbRows := []db.ListProductsByCategoryAndDateRow{
		{
			ID:           pgconv.ParseUUIDToPgType(uuid.New()),
			Name:         "Produto Cat+Data",
			CostPrice:    pgconv.Float64ToPgNumeric(30.00),
			Quantity:     8,
			CategoryID:   pgconv.ParseUUIDToPgType(categoryID),
			CreatedAt:    pgconv.TimeToPgTimestamptz(now),
			CategoryName: "Categoria Filtrada",
		},
	}

	repo.EXPECT().
		ListProductBuCategoryIdAndDate(gomock.Any(), db.ListProductsByCategoryAndDateParams{
			CategoryID:  pgconv.ParseUUIDToPgType(categoryID),
			CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
		}).
		Return(dbRows, nil)

	resp, err := svc.ListProductBuCategoryIdAndDate(context.Background(), categoryID, startAt, endAt)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("esperava 1 produto, obteve %d", len(resp))
	}
	if resp[0].CategoryName != "Categoria Filtrada" {
		t.Errorf("CategoryName incorreto: '%s'", resp[0].CategoryName)
	}
	if resp[0].Quantity != 8 {
		t.Errorf("Quantity incorreta: %d", resp[0].Quantity)
	}
}

func TestListProductBuCategoryIdAndDate_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		ListProductBuCategoryIdAndDate(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	startAt := time.Now()
	_, err := svc.ListProductBuCategoryIdAndDate(context.Background(), uuid.New(), startAt, startAt)

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}
