package domain

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/shared/variation"
	"github.com/google/uuid"
)

// Grade de produtos: cada combinação cor × tamanho é um produto normal,
// agrupado por model_name. Vendas, trocas e compras continuam olhando o
// produto (a variação); a grade só organiza cadastro, estoque e etiquetas.

var (
	ErrInvalidGrid          = errors.New("grade inválida")
	ErrModelNotFound        = errors.New("modelo não encontrado")
	ErrModelAlreadyExists   = errors.New("já existe um modelo com esse nome")
	ErrVariationsExist      = errors.New("já existem variações cadastradas com essa cor e tamanho")
	ErrVariationHasStock    = errors.New("só é possível remover uma variação sem saldo em estoque")
	ErrProductLimitReached  = errors.New("limite de produtos do plano atingido")
	ErrBarcodeGeneration    = errors.New("não foi possível gerar um código de barras único")
	ErrBarcodeAlreadyExists = errors.New("código de barras já usado por outro produto")
)

const (
	MaxGridVariations = 200
	minModelName      = 2
)

// GridVariationInput uma combinação a cadastrar
type GridVariationInput struct {
	Color string `json:"color" binding:"max=50"`
	Size  string `json:"size" binding:"max=20"`
	// Estoque inicial; precisa ser maior que zero
	Quantity int32 `json:"quantity" binding:"gte=0"`
	// Código de fábrica; vazio gera um código interno
	Barcode string `json:"barcode" binding:"max=50"`
	// Custo e preço próprios da variação; omitidos usam os do modelo
	CostPrice *float64 `json:"cost_price" binding:"omitempty,gte=0"`
	SalePrice *float64 `json:"sale_price" binding:"omitempty,gte=0"`
}

// CreateModelRequest cadastra um modelo novo com a grade
type CreateModelRequest struct {
	ModelName   string               `json:"model_name" binding:"required,max=100"`
	CategoryID  uuid.UUID            `json:"category_id" binding:"required"`
	Description string               `json:"description" binding:"max=1000"`
	CostPrice   float64              `json:"cost_price" binding:"gte=0"`
	SalePrice   float64              `json:"sale_price" binding:"gte=0"`
	Variations  []GridVariationInput `json:"variations" binding:"required,min=1,dive"`
}

// AddVariationsRequest adiciona cores/tamanhos a um modelo existente; categoria
// e descrição vêm do próprio modelo
type AddVariationsRequest struct {
	CostPrice  float64              `json:"cost_price" binding:"gte=0"`
	SalePrice  float64              `json:"sale_price" binding:"gte=0"`
	Variations []GridVariationInput `json:"variations" binding:"required,min=1,dive"`
}

type VariationPriceInput struct {
	ID        uuid.UUID `json:"id" binding:"required"`
	CostPrice float64   `json:"cost_price" binding:"gte=0"`
	SalePrice float64   `json:"sale_price" binding:"gte=0"`
	// Troca o código (ex.: interno pelo de fábrica); omitido mantém o atual
	Barcode *string `json:"barcode" binding:"omitempty,max=50"`
}

// UpdateModelRequest edita o modelo inteiro. Nome, categoria e descrição valem
// para todas as variações; os preços vêm por variação (o front decide se o
// preço do modelo sobrescreve as exceções). Variações fora da lista mantêm o preço.
type UpdateModelRequest struct {
	ModelName   string                `json:"model_name" binding:"required,max=100"`
	CategoryID  uuid.UUID             `json:"category_id" binding:"required"`
	Description string                `json:"description" binding:"max=1000"`
	Variations  []VariationPriceInput `json:"variations" binding:"dive"`
}

type GridStockItem struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int32     `json:"quantity"`
}

// GridStockMovementRequest entrada ou ajuste de várias variações de uma vez
type GridStockMovementRequest struct {
	Type   string          `json:"type" binding:"required"`
	Reason string          `json:"reason" binding:"max=500"`
	Items  []GridStockItem `json:"items" binding:"required,min=1,dive"`
}

type GroupItem struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Color     string    `json:"color" binding:"max=50"`
	Size      string    `json:"size" binding:"max=20"`
}

// GroupProductsRequest transforma produtos soltos (legado) em variações de um
// modelo, novo ou existente. O produto continua o mesmo: saldo, histórico e
// vendas são preservados.
type GroupProductsRequest struct {
	ModelName string `json:"model_name" binding:"required,max=100"`
	// Obrigatória para modelo novo; num modelo existente vale a dele
	CategoryID uuid.UUID   `json:"category_id"`
	Items      []GroupItem `json:"items" binding:"required,min=1,dive"`
}

// ModelQuery identifica o modelo pela query string (nomes podem ter "/")
type ModelQuery struct {
	Name string `form:"name" binding:"required,max=100"`
}

// StockEntry uma linha do estoque agrupado: um modelo ou um produto sem grade
type StockEntry struct {
	Kind string `json:"kind"` // "model" | "product"
	Key  string `json:"key"`
	// Produto sem grade: id e dados do próprio produto
	ProductID   *uuid.UUID `json:"product_id"`
	Name        string     `json:"name"`
	ModelName   string     `json:"model_name"`
	Description string     `json:"description"`
	Barcode     string     `json:"barcode"`
	Size        string     `json:"size"`
	Color       string     `json:"color"`
	CategoryID  uuid.UUID  `json:"category_id"`
	Category    string     `json:"category_name"`
	Unit        string     `json:"unit"`
	SellInBulk  bool       `json:"sell_in_bulk"`
	// Modelo: totais das variações
	Variations         int32     `json:"variations"`
	Colors             int32     `json:"colors"`
	Sizes              int32     `json:"sizes"`
	Quantity           int32     `json:"quantity"`
	LowStockVariations int32     `json:"low_stock_variations"`
	MinCostPrice       float64   `json:"min_cost_price"`
	MaxCostPrice       float64   `json:"max_cost_price"`
	MinSalePrice       float64   `json:"min_sale_price"`
	MaxSalePrice       float64   `json:"max_sale_price"`
	TotalRegistered    int32     `json:"total_registered"`
	TotalSold          int32     `json:"total_sold"`
	CreatedAt          time.Time `json:"created_at"`
}

type StockEntriesResponse struct {
	Data              []StockEntry `json:"data"`
	Page              int32        `json:"page"`
	PerPage           int32        `json:"per_page"`
	TotalRows         int64        `json:"total_rows"`
	TotalPages        int32        `json:"total_pages"`
	ProductsCount     int64        `json:"products_count"`
	TotalValueInStock float64      `json:"total_value_in_stock"`
	ItensInStock      int32        `json:"itens_in_stock"`
	LowItensInStock   int32        `json:"low_itens_in_stock"`
}

// GridVariation uma variação com tudo que as telas da grade precisam
type GridVariation struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	ModelName       string    `json:"model_name"`
	Color           string    `json:"color"`
	Size            string    `json:"size"`
	Barcode         string    `json:"barcode"`
	Description     string    `json:"description"`
	CategoryID      uuid.UUID `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	Unit            string    `json:"unit"`
	SellInBulk      bool      `json:"sell_in_bulk"`
	Quantity        int32     `json:"quantity"`
	CostPrice       float64   `json:"cost_price"`
	SalePrice       float64   `json:"sale_price"`
	TotalRegistered int32     `json:"total_registered"`
	TotalSold       int32     `json:"total_sold"`
	CreatedAt       time.Time `json:"created_at"`
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// NormalizeModelName limpa espaços repetidos e valida o tamanho
func NormalizeModelName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if len([]rune(name)) < minModelName {
		return "", fmt.Errorf("%w: informe o nome do modelo", ErrInvalidGrid)
	}
	return name, nil
}

// checkVariationName garante que "Modelo – Cor – Tamanho" cabe na coluna name
func checkVariationName(model, color, size string) error {
	name := variation.Name(model, color, size)
	if len([]rune(name)) > variation.MaxProductName {
		return fmt.Errorf("%w: o nome %q passa de %d caracteres; encurte o modelo ou a cor", ErrInvalidGrid, name, variation.MaxProductName)
	}
	return nil
}

// NormalizeGridVariations limpa espaços, recusa combinações repetidas e valida o nome gerado
func NormalizeGridVariations(model string, in []GridVariationInput) ([]GridVariationInput, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("%w: informe ao menos uma variação", ErrInvalidGrid)
	}
	if len(in) > MaxGridVariations {
		return nil, fmt.Errorf("%w: no máximo %d variações por vez", ErrInvalidGrid, MaxGridVariations)
	}
	seen := map[string]bool{}
	barcodes := map[string]bool{}
	out := make([]GridVariationInput, 0, len(in))
	for _, v := range in {
		v.Color = strings.Join(strings.Fields(v.Color), " ")
		v.Size = strings.Join(strings.Fields(v.Size), " ")
		v.Barcode = strings.TrimSpace(v.Barcode)
		if v.Color == "" && v.Size == "" {
			return nil, fmt.Errorf("%w: toda variação precisa de cor ou tamanho", ErrInvalidGrid)
		}
		// Só entra na grade o que tem peça: variação zerada não é cadastrada
		// (o cadastro rápido de Compras é outro fluxo e cria sem estoque)
		if v.Quantity <= 0 {
			return nil, fmt.Errorf("%w: a variação %q está sem estoque; informe a quantidade ou tire-a da grade", ErrInvalidGrid, variation.Name(model, v.Color, v.Size))
		}
		key := variation.Key(v.Color, v.Size)
		if seen[key] {
			return nil, fmt.Errorf("%w: a variação %q aparece mais de uma vez", ErrInvalidGrid, variation.Name("", v.Color, v.Size))
		}
		seen[key] = true
		if v.Barcode != "" {
			if barcodes[v.Barcode] {
				return nil, fmt.Errorf("%w: o código %s aparece em mais de uma variação", ErrInvalidGrid, v.Barcode)
			}
			barcodes[v.Barcode] = true
		}
		if err := checkVariationName(model, v.Color, v.Size); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// NormalizeGroupItems valida o agrupamento de produtos soltos
func NormalizeGroupItems(model string, in []GroupItem) ([]GroupItem, error) {
	if len(in) > MaxGridVariations {
		return nil, fmt.Errorf("%w: no máximo %d produtos por vez", ErrInvalidGrid, MaxGridVariations)
	}
	seenKey := map[string]bool{}
	seenID := map[uuid.UUID]bool{}
	out := make([]GroupItem, 0, len(in))
	for _, it := range in {
		it.Color = strings.Join(strings.Fields(it.Color), " ")
		it.Size = strings.Join(strings.Fields(it.Size), " ")
		if it.Color == "" && it.Size == "" {
			return nil, fmt.Errorf("%w: informe a cor ou o tamanho de cada produto", ErrInvalidGrid)
		}
		if seenID[it.ProductID] {
			return nil, fmt.Errorf("%w: o mesmo produto aparece mais de uma vez", ErrInvalidGrid)
		}
		seenID[it.ProductID] = true
		key := variation.Key(it.Color, it.Size)
		if seenKey[key] {
			return nil, fmt.Errorf("%w: dois produtos com a mesma cor e tamanho (%s)", ErrInvalidGrid, variation.Name("", it.Color, it.Size))
		}
		seenKey[key] = true
		if err := checkVariationName(model, it.Color, it.Size); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

// ValidateGridStockMovement mesmas regras do lançamento avulso, item a item
func ValidateGridStockMovement(req GridStockMovementRequest) error {
	seen := map[uuid.UUID]bool{}
	nonZero := 0
	for _, it := range req.Items {
		if seen[it.ProductID] {
			return fmt.Errorf("%w: a mesma variação aparece mais de uma vez", ErrInvalidGrid)
		}
		seen[it.ProductID] = true
		if it.Quantity != 0 {
			nonZero++
		}
		if err := ValidateCreateStockMovementRequest(CreateStockMovementRequest{
			Type:     req.Type,
			Quantity: it.Quantity,
			Reason:   req.Reason,
		}); err != nil && !(errors.Is(err, ErrInvalidStockMovementQuantity) && it.Quantity == 0) {
			return err
		}
	}
	if nonZero == 0 {
		return ErrInvalidStockMovementQuantity
	}
	return nil
}

// PriceOrDefault preço próprio da variação ou o do modelo
func PriceOrDefault(own *float64, model float64) float64 {
	if own != nil {
		return round2(*own)
	}
	return round2(model)
}

// Round2 arredonda valores monetários
func Round2(v float64) float64 { return round2(v) }
