package domain

import (
	"fmt"
	"math/rand"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID `json:"id"`
	CompanyID   uuid.UUID `json:"company_id"`
	CategoryID  uuid.UUID `json:"category_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Barcode     string    `json:"barcode"`
	Quantity    int32     `json:"quantity"`
	Size        string    `json:"size"`
	CostPrice   float64   `json:"cost_price"`
	SalePrice   float64   `json:"sale_price"`
	CreatedBy   uuid.UUID `json:"created_by"`
	UpdatedBy   uuid.UUID `json:"updated_by"`
	DeletedBy   uuid.UUID `json:"deleted_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
}

type CreateProductRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CategoryID  uuid.UUID `json:"category_id"`
	Barcode     string    `json:"barcode"`
	Quantity    int32     `json:"quantity"`
	Size        string    `json:"size"`
	CostPrice   float64   `json:"cost_price"`
	SalePrice   float64   `json:"sale_price"`
	NotBarcode  bool      `json:"not_barcode"`
	SellInBulk  bool      `json:"sell_in_bulk"`
	Unit        string    `json:"unit"`
}

type DeleteProductRequest struct {
	ID        uuid.UUID `json:"id"`
	DeletedBy uuid.UUID `json:"deleted_by"`
}

type ListProductsByCategoryIdRequest struct {
	CategoryID uuid.UUID `json:"category_id" form:"category_id"`
	CompanyID  uuid.UUID `json:"company_id" form:"company_id"`
}

type UpdateProductRequest struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CategoryID  uuid.UUID `json:"category_id"`
	Barcode     string    `json:"barcode"`
	Quantity    int32     `json:"quantity"`
	Size        string    `json:"size"`
	CostPrice   float64   `json:"cost_price"`
	SalePrice   float64   `json:"sale_price"`
	UpdatedBy   uuid.UUID `json:"updated_by"`
	Unit        string    `json:"unit"`
}

type ProductResponse struct {
	ID          uuid.UUID `json:"id"`
	CompanyID   uuid.UUID `json:"company_id"`
	CategoryID  uuid.UUID `json:"category_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Barcode     string    `json:"barcode"`
	Quantity    int32     `json:"quantity"`
	Size        string    `json:"size"`
	CostPrice   float64   `json:"cost_price"`
	SalePrice   float64   `json:"sale_price"`
	CreatedBy   uuid.UUID `json:"created_by"`
	UpdatedBy   uuid.UUID `json:"updated_by"`
	DeletedBy   uuid.UUID `json:"deleted_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
}

type ListProductsByCompanyRow struct {
	ID           uuid.UUID `json:"id"`
	CompanyID    uuid.UUID `json:"company_id"`
	CategoryID   uuid.UUID `json:"category_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Barcode      string    `json:"barcode"`
	Quantity     int32     `json:"quantity"`
	Size         string    `json:"size"`
	CostPrice    float64   `json:"cost_price"`
	SalePrice    float64   `json:"sale_price"`
	CreatedBy    uuid.UUID `json:"created_by"`
	UpdatedBy    uuid.UUID `json:"updated_by"`
	DeletedBy    uuid.UUID `json:"deleted_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    time.Time `json:"deleted_at"`
	CategoryName string    `json:"category_name"`
	SellInBulk   bool      `json:"sell_in_bulk"`
	Unit         string    `json:"unit"`
}

type DecrementStockRequest struct {
	ID       uuid.UUID `json:"id"`
	Quantity int32     `json:"quantity"`
}

type GetTop5BestSellingProductsRow struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	TotalQuantitySold int32     `json:"total_quantity_sold"`
}

type GetInventoryReportResponse struct {
	Name         string    `json:"name" excel:"Nome do Produto"`
	CategoryName string    `json:"category_name" excel:"Categoria"`
	Quantity     int32     `json:"quantity" excel:"Quantidade"`
	SalePrice    float64   `json:"sale_price" excel:"Preço de Venda"`
	TotalValue   float64   `json:"total_value" excel:"Valor Total em Estoque"`
	CostPrice    float64   `json:"cost_price" excel:"Preço de Custo"`
	Barcode      string    `json:"barcode" excel:"Código de Barras"`
	CreatedAt    time.Time `json:"created_at" excel:"Data de Cadastro"`
}

type ListProductsByDateResponse struct {
	ID           uuid.UUID `json:"id"`
	CompanyID    uuid.UUID `json:"company_id"`
	CategoryID   uuid.UUID `json:"category_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Barcode      string    `json:"barcode"`
	Quantity     int32     `json:"quantity"`
	Size         string    `json:"size"`
	CostPrice    float64   `json:"cost_price"`
	SalePrice    float64   `json:"sale_price"`
	CreatedBy    uuid.UUID `json:"created_by"`
	UpdatedBy    uuid.UUID `json:"updated_by"`
	DeletedBy    uuid.UUID `json:"deleted_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    time.Time `json:"deleted_at"`
	CategoryName string    `json:"category_name"`
}

type ListProductsByCategoryAndDateResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	CostPrice    float64   `json:"cost_price"`
	Quantity     int32     `json:"quantity"`
	CategoryID   uuid.UUID `json:"category_id"`
	CreatedAt    time.Time `json:"created_at"`
	CategoryName string    `json:"category_name"`
}

type ProductPaginatedResponse struct {
	globalDomain.PaginatedResponse[ListProductsByCompanyRow]
	TotalValueInStock float64 `json:"total_value_in_stock"`
	ItensInStock      int32   `json:"itens_in_stock"`
	LowItensInStock   int32   `json:"low_itens_in_stock"`
}

func ApplyUpdateProductParams(
	req UpdateProductRequest,
	arg *db.UpdateProductParams,
) {
	if req.Name != "" {
		arg.Name = req.Name
	}

	if req.Description != "" {
		arg.Description = pgconv.ParseStringToPgText(req.Description)
	}

	if req.CategoryID != uuid.Nil {
		arg.CategoryID = pgconv.ParseUUIDToPgType(req.CategoryID)
	}

	if req.Barcode != "" {
		arg.Barcode = pgconv.ParseStringToPgText(req.Barcode)
	}

	if req.Quantity != 0 {
		arg.Quantity = pgconv.IntToPgInt4(int(req.Quantity))
	}

	if req.Size != "" {
		arg.Size = pgconv.ParseStringToPgText(req.Size)
	}

	if req.CostPrice != 0 {
		arg.CostPrice = pgconv.Float64ToPgNumeric(req.CostPrice)
	}

	if req.SalePrice != 0 {
		arg.SalePrice = pgconv.Float64ToPgNumeric(req.SalePrice)
	}

	if req.UpdatedBy != uuid.Nil {
		arg.UpdatedBy = pgconv.ParseUUIDToPgType(req.UpdatedBy)
	}

	if req.Unit != "" {
		arg.Unit = db.UnitOfMeasure(req.Unit)
	}
}

func GerarEAN13(base string) (string, error) {
	if base == "" {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < 12; i++ {
			base += fmt.Sprintf("%d", rng.Intn(10))
		}
	} else if len(base) != 12 {
		return "", fmt.Errorf("a base deve conter exatamente 12 dígitos")
	}

	// Cálculo do dígito verificador (EAN-13)
	soma := 0
	for i, r := range base {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("a base deve conter apenas números")
		}
		digito := int(r - '0')
		if i%2 == 0 {
			soma += digito * 1
		} else {
			soma += digito * 3
		}
	}

	digitoVerificador := (10 - (soma % 10)) % 10

	return fmt.Sprintf("%s%d", base, digitoVerificador), nil
}
