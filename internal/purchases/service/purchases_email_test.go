package service

import (
	"strings"
	"testing"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/purchases/domain"
)

func TestBRL(t *testing.T) {
	for in, want := range map[float64]string{0: "R$ 0,00", 1234.5: "R$ 1.234,50", 1234567.891: "R$ 1.234.567,89", -10: "-R$ 10,00"} {
		if got := brl(in); got != want {
			t.Errorf("brl(%v) = %q, esperava %q", in, got, want)
		}
	}
}

func TestValidEmail(t *testing.T) {
	if got, ok := validEmail("  Fornecedor <vendas@malharia.com.br> "); !ok || got != "vendas@malharia.com.br" {
		t.Errorf("veio %q %v", got, ok)
	}
	for _, bad := range []string{"", "sem-arroba", "a@b", "x@y.com\r\nBcc: z@w.com"} {
		if _, ok := validEmail(bad); ok {
			t.Errorf("%q deveria ser inválido", bad)
		}
	}
}

func TestRenderOrderEmailEscapaHTML(t *testing.T) {
	notes := "<script>alert(1)</script>"
	expected := "2026-11-10"
	html, err := renderOrderEmail(domain.OrderDetail{
		Number:       7,
		ExpectedDate: &expected,
		Notes:        &notes,
		Total:        1530,
		Items:        []domain.OrderItem{{ProductName: "Camiseta Básica – Preta – M", Quantity: 15, UnitCost: 18, Total: 270}},
	}, db.GetPurchaseEmailContextRow{CompanyName: "Loja <X>", VendorName: "Malharia", UserName: "Ana"}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Pedido de compra nº 7 (atualizado)", "10/11/2026", "R$ 1.530,00", "Camiseta Básica – Preta – M", "Loja &lt;X&gt;", "&lt;script&gt;"} {
		if !strings.Contains(html, want) {
			t.Errorf("e-mail sem %q", want)
		}
	}
	if strings.Contains(html, "<script>") {
		t.Error("observação não foi escapada")
	}
}
