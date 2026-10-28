package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestVariationName(t *testing.T) {
	tests := []struct{ model, color, size, want string }{
		{"Camiseta Básica", "Preta", "M", "Camiseta Básica – Preta – M"},
		{"Calça Jeans", "", "42", "Calça Jeans – 42"},
		{"Boné", "Azul", "", "Boné – Azul"},
		{"  Meia  ", " ", " ", "Meia"},
	}
	for _, tt := range tests {
		if got := VariationName(tt.model, tt.color, tt.size); got != tt.want {
			t.Errorf("VariationName(%q,%q,%q) = %q, esperava %q", tt.model, tt.color, tt.size, got, tt.want)
		}
	}
}

func TestNormalizeVariationsRecusaRepetidas(t *testing.T) {
	_, err := NormalizeVariations([]VariationRequest{{Color: "Preta", Size: "M"}, {Color: " preta ", Size: "m"}})
	if !errors.Is(err, ErrInvalidPurchase) {
		t.Fatalf("esperava ErrInvalidPurchase para combinação repetida, veio %v", err)
	}
	out, err := NormalizeVariations([]VariationRequest{{Color: " Preta ", Size: "M "}, {Color: "Preta", Size: "G"}})
	if err != nil || out[0].Color != "Preta" || out[0].Size != "M" {
		t.Fatalf("normalização: %+v %v", out, err)
	}
}

func TestInternalEAN13(t *testing.T) {
	for range 200 {
		code, err := InternalEAN13()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 13 || !strings.HasPrefix(code, "2") {
			t.Fatalf("código inválido: %s", code)
		}
		// Dígito verificador: soma ponderada (1,3,1,3...) dos 13 dígitos é múltipla de 10
		sum := 0
		for i, r := range code {
			w := 1
			if i%2 == 1 {
				w = 3
			}
			sum += int(r-'0') * w
		}
		if sum%10 != 0 {
			t.Fatalf("dígito verificador errado: %s", code)
		}
	}
}
