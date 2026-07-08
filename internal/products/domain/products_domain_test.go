package domain_test

import (
	"testing"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/products/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newPgUUID converte um uuid.UUID para pgtype.UUID sem depender de pgconv nos testes.
func newPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// newPgText converte uma string para pgtype.Text.
func newPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

// newPgNumeric cria um pgtype.Numeric a partir de um float64.
func newPgNumeric(v float64) pgtype.Numeric {
	return pgconv.Float64ToPgNumeric(v)
}

// ---------------------------------------------------------------------------
// TestApplyUpdateProductParams
// ---------------------------------------------------------------------------

func TestApplyUpdateProductParams_UpdatesOnlyNonZeroFields(t *testing.T) {
	originalID := uuid.New()
	originalCategoryID := uuid.New()
	updatedBy := uuid.New()

	// arg representa o estado atual do produto no banco
	arg := db.UpdateProductParams{
		ID:          newPgUUID(originalID),
		Name:        "Produto Original",
		Description: newPgText("Descrição original"),
		CategoryID:  newPgUUID(originalCategoryID),
		Barcode:     newPgText("1234567890"),
		Quantity:    10,
		Size:        newPgText("M"),
		CostPrice:   newPgNumeric(50.00),
		SalePrice:   newPgNumeric(100.00),
		UpdatedBy:   newPgUUID(uuid.Nil),
	}

	newCategoryID := uuid.New()
	req := domain.UpdateProductRequest{
		Name:        "Novo Nome",
		Description: "Nova descrição",
		CategoryID:  newCategoryID,
		Barcode:     "0987654321",
		Quantity:    20,
		Size:        "L",
		CostPrice:   75.00,
		SalePrice:   150.00,
		UpdatedBy:   updatedBy,
	}

	domain.ApplyUpdateProductParams(req, &arg)

	if arg.Name != "Novo Nome" {
		t.Errorf("esperava Name='Novo Nome', obteve '%s'", arg.Name)
	}
	if arg.Description.String != "Nova descrição" {
		t.Errorf("esperava Description='Nova descrição', obteve '%s'", arg.Description.String)
	}
	if arg.Barcode.String != "0987654321" {
		t.Errorf("esperava Barcode='0987654321', obteve '%s'", arg.Barcode.String)
	}
	if arg.Quantity != 20 {
		t.Errorf("esperava Quantity=20, obteve %d", arg.Quantity)
	}
	if arg.Size.String != "L" {
		t.Errorf("esperava Size='L', obteve '%s'", arg.Size.String)
	}
	if pgconv.PgUUIDToUUID(arg.CategoryID) != newCategoryID {
		t.Errorf("esperava CategoryID=%v, obteve %v", newCategoryID, pgconv.PgUUIDToUUID(arg.CategoryID))
	}
	if pgconv.PgUUIDToUUID(arg.UpdatedBy) != updatedBy {
		t.Errorf("esperava UpdatedBy=%v, obteve %v", updatedBy, pgconv.PgUUIDToUUID(arg.UpdatedBy))
	}
	if pgconv.PgNumericToFloat64(arg.CostPrice) != 75.00 {
		t.Errorf("esperava CostPrice=75.00, obteve %f", pgconv.PgNumericToFloat64(arg.CostPrice))
	}
	if pgconv.PgNumericToFloat64(arg.SalePrice) != 150.00 {
		t.Errorf("esperava SalePrice=150.00, obteve %f", pgconv.PgNumericToFloat64(arg.SalePrice))
	}
}

func TestApplyUpdateProductParams_DoesNotOverwriteWithZeroValues(t *testing.T) {
	originalID := uuid.New()
	originalCategoryID := uuid.New()

	arg := db.UpdateProductParams{
		ID:          newPgUUID(originalID),
		Name:        "Produto Original",
		Description: newPgText("Descrição original"),
		CategoryID:  newPgUUID(originalCategoryID),
		Barcode:     newPgText("1234567890"),
		Quantity:    10,
		Size:        newPgText("M"),
		CostPrice:   newPgNumeric(50.00),
		SalePrice:   newPgNumeric(100.00),
		UpdatedBy:   newPgUUID(uuid.Nil),
	}

	// Requisição com campos zerados: nenhum campo deve sobrescrever o arg
	req := domain.UpdateProductRequest{
		Name:        "",
		Description: "",
		CategoryID:  uuid.Nil,
		Barcode:     "",
		Quantity:    0,
		Size:        "",
		CostPrice:   0,
		SalePrice:   0,
		UpdatedBy:   uuid.Nil,
	}

	domain.ApplyUpdateProductParams(req, &arg)

	// Todos os campos originais devem ser preservados
	if arg.Name != "Produto Original" {
		t.Errorf("Name não deve ter sido alterado; obteve '%s'", arg.Name)
	}
	if arg.Description.String != "Descrição original" {
		t.Errorf("Description não deve ter sido alterada; obteve '%s'", arg.Description.String)
	}
	if pgconv.PgUUIDToUUID(arg.CategoryID) != originalCategoryID {
		t.Errorf("CategoryID não deve ter sido alterado")
	}
	if arg.Quantity != 10 {
		t.Errorf("Quantity não deve ter sido alterada; obteve %d", arg.Quantity)
	}
	if arg.Barcode.String != "1234567890" {
		t.Errorf("Barcode não deve ter sido alterado; obteve '%s'", arg.Barcode.String)
	}
	if arg.Size.String != "M" {
		t.Errorf("Size não deve ter sido alterado; obteve '%s'", arg.Size.String)
	}
}

func TestApplyUpdateProductParams_PartialUpdate(t *testing.T) {
	arg := db.UpdateProductParams{
		Name:      "Original",
		Quantity:  5,
		CostPrice: newPgNumeric(10.00),
		SalePrice: newPgNumeric(20.00),
	}

	// Atualiza apenas Name e SalePrice
	req := domain.UpdateProductRequest{
		Name:      "Atualizado",
		SalePrice: 35.00,
	}

	domain.ApplyUpdateProductParams(req, &arg)

	if arg.Name != "Atualizado" {
		t.Errorf("esperava Name='Atualizado', obteve '%s'", arg.Name)
	}
	if arg.Quantity != 5 {
		t.Errorf("Quantity não deve ter sido alterada; obteve %d", arg.Quantity)
	}
	if pgconv.PgNumericToFloat64(arg.CostPrice) != 10.00 {
		t.Errorf("CostPrice não deve ter sido alterado; obteve %f", pgconv.PgNumericToFloat64(arg.CostPrice))
	}
	if pgconv.PgNumericToFloat64(arg.SalePrice) != 35.00 {
		t.Errorf("esperava SalePrice=35.00, obteve %f", pgconv.PgNumericToFloat64(arg.SalePrice))
	}
}

// ---------------------------------------------------------------------------
// TestProductStructFields — validação básica das structs de domínio
// ---------------------------------------------------------------------------

func TestCreateProductRequest_FieldAssignment(t *testing.T) {
	companyID := uuid.New()
	categoryID := uuid.New()
	createdBy := uuid.New()

	req := domain.CreateProductRequest{
		CompanyID:   companyID,
		Name:        "Produto Teste",
		Description: "Descrição teste",
		CategoryID:  categoryID,
		Barcode:     "ABC123",
		Quantity:    100,
		Size:        "G",
		CostPrice:   25.50,
		SalePrice:   59.90,
		CreatedBy:   createdBy,
	}

	if req.CompanyID != companyID {
		t.Errorf("CompanyID incorreto")
	}
	if req.Name != "Produto Teste" {
		t.Errorf("Name incorreto: '%s'", req.Name)
	}
	if req.Quantity != 100 {
		t.Errorf("Quantity incorreta: %d", req.Quantity)
	}
	if req.CostPrice != 25.50 {
		t.Errorf("CostPrice incorreto: %f", req.CostPrice)
	}
	if req.SalePrice != 59.90 {
		t.Errorf("SalePrice incorreto: %f", req.SalePrice)
	}
}

func TestDeleteProductRequest_FieldAssignment(t *testing.T) {
	id := uuid.New()
	deletedBy := uuid.New()

	req := domain.DeleteProductRequest{
		ID:        id,
		DeletedBy: deletedBy,
	}

	if req.ID != id {
		t.Errorf("ID incorreto")
	}
	if req.DeletedBy != deletedBy {
		t.Errorf("DeletedBy incorreto")
	}
}

func TestDecrementStockRequest_FieldAssignment(t *testing.T) {
	id := uuid.New()

	req := domain.DecrementStockRequest{
		ID:       id,
		Quantity: 5,
	}

	if req.ID != id {
		t.Errorf("ID incorreto")
	}
	if req.Quantity != 5 {
		t.Errorf("Quantity incorreta: %d", req.Quantity)
	}
}
