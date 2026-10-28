package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrCategoryNotFound        = errors.New("categoria de produto não encontrada")
	ErrProductLimitReached     = errors.New("limite de produtos do plano atingido")
	ErrVariationsExist         = errors.New("já existem produtos cadastrados com estas variações")
	ErrBarcodeGenerationFailed = errors.New("não foi possível gerar um código de barras único")
)

// Separador do nome gerado: "Camiseta Básica – Preta – M"
const nameSeparator = " – "

type CatalogSearchParams struct {
	Search  string `form:"search" binding:"max=100"`
	Page    int32  `form:"page" binding:"omitempty,gte=1"`
	PerPage int32  `form:"per_page" binding:"omitempty,gte=1,lte=50"`
}

type CatalogProduct struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	ModelName  string    `json:"model_name"`
	Color      string    `json:"color"`
	Size       string    `json:"size"`
	Barcode    string    `json:"barcode"`
	Unit       string    `json:"unit"`
	SellInBulk bool      `json:"sell_in_bulk"`
	Quantity   int32     `json:"quantity"`
	CostPrice  float64   `json:"cost_price"`
	SalePrice  float64   `json:"sale_price"`
}

type CatalogModel struct {
	ModelName  string `json:"model_name"`
	Variations int32  `json:"variations"`
}

type CatalogSearchResponse struct {
	Products   []CatalogProduct `json:"products"`
	Models     []CatalogModel   `json:"models"`
	Page       int32            `json:"page"`
	PerPage    int32            `json:"per_page"`
	TotalRows  int32            `json:"total_rows"`
	TotalPages int32            `json:"total_pages"`
}

type CatalogCategory struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type VariationRequest struct {
	Color string `json:"color" binding:"max=50"`
	Size  string `json:"size" binding:"max=20"`
	// Custo e preço próprios da variação; omitidos usam os do modelo
	CostPrice *float64 `json:"cost_price" binding:"omitempty,gte=0"`
	SalePrice *float64 `json:"sale_price" binding:"omitempty,gte=0"`
}

type CreateVariationsRequest struct {
	ModelName   string             `json:"model_name" binding:"required,min=2,max=120"`
	CategoryID  uuid.UUID          `json:"category_id" binding:"required"`
	Description string             `json:"description" binding:"max=500"`
	CostPrice   float64            `json:"cost_price" binding:"gte=0"`
	SalePrice   float64            `json:"sale_price" binding:"gte=0"`
	Variations  []VariationRequest `json:"variations" binding:"required,min=1,max=100,dive"`
}

// VariationName monta o nome do produto pulando cor/tamanho vazios
func VariationName(model, color, size string) string {
	parts := []string{strings.TrimSpace(model)}
	for _, p := range []string{color, size} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, nameSeparator)
}

// VariationKey identifica a combinação sem diferenciar maiúsculas (mesma regra do índice único)
func VariationKey(color, size string) string {
	return strings.ToLower(strings.TrimSpace(color)) + "|" + strings.ToLower(strings.TrimSpace(size))
}

// NormalizeVariations limpa espaços e recusa combinações repetidas na mesma requisição
func NormalizeVariations(variations []VariationRequest) ([]VariationRequest, error) {
	seen := map[string]bool{}
	out := make([]VariationRequest, 0, len(variations))
	for _, v := range variations {
		v.Color = strings.TrimSpace(v.Color)
		v.Size = strings.TrimSpace(v.Size)
		key := VariationKey(v.Color, v.Size)
		if seen[key] {
			return nil, fmt.Errorf("%w: a variação %q aparece mais de uma vez", ErrInvalidPurchase, VariationName("", v.Color, v.Size))
		}
		seen[key] = true
		out = append(out, v)
	}
	return out, nil
}

// InternalEAN13 gera um EAN-13 de circulação interna: o prefixo 2 é a faixa
// que o GS1 reserva para uso dentro da loja, então não colide com o código de
// fábrica de nenhum produto.
func InternalEAN13() (string, error) {
	digits := make([]byte, 12)
	digits[0] = '2'
	for i := 1; i < 12; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = byte('0' + n.Int64())
	}

	sum := 0
	for i, d := range digits {
		weight := 1
		if i%2 == 1 {
			weight = 3
		}
		sum += int(d-'0') * weight
	}
	check := (10 - sum%10) % 10
	return string(digits) + string(rune('0'+check)), nil
}
