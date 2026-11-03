package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeGridVariations(t *testing.T) {
	t.Run("limpa espaços e mantém a ordem", func(t *testing.T) {
		out, err := NormalizeGridVariations("Camiseta", []GridVariationInput{
			{Color: "  Preta ", Size: " M", Quantity: 2},
			{Color: "Preta", Size: "G", Quantity: 1},
		})
		if err != nil {
			t.Fatal(err)
		}
		if out[0].Color != "Preta" || out[0].Size != "M" || out[1].Size != "G" {
			t.Fatalf("inesperado: %+v", out)
		}
	})

	// Cada caso tem uma variação válida (com estoque) e o problema que deve barrar
	cases := []struct {
		name  string
		model string
		in    []GridVariationInput
		want  string
	}{
		{"vazia", "Camiseta", nil, "ao menos uma variação"},
		{"sem cor nem tamanho", "Camiseta", []GridVariationInput{{Color: " ", Quantity: 1}}, "cor ou tamanho"},
		{"estoque zerado", "Camiseta", []GridVariationInput{{Color: "Preta", Size: "M", Quantity: 1}, {Color: "Preta", Size: "G"}}, "sem estoque"},
		{"estoque negativo", "Camiseta", []GridVariationInput{{Color: "Preta", Size: "M", Quantity: -1}}, "sem estoque"},
		{"repetida sem maiúsculas", "Camiseta", []GridVariationInput{{Color: "Preta", Size: "M", Quantity: 1}, {Color: "preta", Size: "m", Quantity: 1}}, "mais de uma vez"},
		{"código repetido", "Camiseta", []GridVariationInput{{Color: "A", Barcode: "123", Quantity: 1}, {Color: "B", Barcode: "123", Quantity: 1}}, "código 123"},
		{"nome passa de 100 chars", strings.Repeat("M", 60), []GridVariationInput{{Color: strings.Repeat("x", 50), Size: "GG", Quantity: 1}}, "100 caracteres"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NormalizeGridVariations(tc.model, tc.in)
			if !errors.Is(err, ErrInvalidGrid) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("esperava ErrInvalidGrid com %q, veio %v", tc.want, err)
			}
		})
	}
}

func TestNormalizeGroupItems(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	if _, err := NormalizeGroupItems("Camiseta", []GroupItem{{ProductID: a, Size: "P"}, {ProductID: b, Size: "M"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeGroupItems("Camiseta", []GroupItem{{ProductID: a, Size: "P"}, {ProductID: a, Size: "M"}}); !errors.Is(err, ErrInvalidGrid) {
		t.Fatalf("produto repetido deveria falhar: %v", err)
	}
	if _, err := NormalizeGroupItems("Camiseta", []GroupItem{{ProductID: a, Size: "P"}, {ProductID: b, Size: "p"}}); !errors.Is(err, ErrInvalidGrid) {
		t.Fatalf("combinação repetida deveria falhar: %v", err)
	}
	if _, err := NormalizeGroupItems("Camiseta", []GroupItem{{ProductID: a}}); !errors.Is(err, ErrInvalidGrid) {
		t.Fatalf("sem cor nem tamanho deveria falhar: %v", err)
	}
}

func TestValidateGridStockMovement(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	ok := []GridStockMovementRequest{
		{Type: "ENTRY", Items: []GridStockItem{{ProductID: a, Quantity: 3}, {ProductID: b, Quantity: 0}}},
		{Type: "ADJUSTMENT", Reason: "avaria", Items: []GridStockItem{{ProductID: a, Quantity: -1}}},
	}
	for _, req := range ok {
		if err := ValidateGridStockMovement(req); err != nil {
			t.Fatalf("%+v: %v", req, err)
		}
	}
	bad := map[string]GridStockMovementRequest{
		"tudo zero":         {Type: "ENTRY", Items: []GridStockItem{{ProductID: a}}},
		"entrada negativa":  {Type: "ENTRY", Items: []GridStockItem{{ProductID: a, Quantity: -2}}},
		"ajuste sem motivo": {Type: "ADJUSTMENT", Items: []GridStockItem{{ProductID: a, Quantity: 2}}},
		"variação repetida": {Type: "ENTRY", Items: []GridStockItem{{ProductID: a, Quantity: 1}, {ProductID: a, Quantity: 1}}},
		"tipo desconhecido": {Type: "SALE", Items: []GridStockItem{{ProductID: a, Quantity: 1}}},
	}
	for name, req := range bad {
		if err := ValidateGridStockMovement(req); err == nil {
			t.Fatalf("%s: esperava erro", name)
		}
	}
}

func TestPriceOrDefault(t *testing.T) {
	own := 79.999
	if PriceOrDefault(&own, 10) != 80 || PriceOrDefault(nil, 59.9) != 59.9 {
		t.Fatal("preço da variação ou do modelo")
	}
}
