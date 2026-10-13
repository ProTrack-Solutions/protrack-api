package generator

import (
	"bytes"
	"testing"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/google/uuid"
)

func TestFormatBRL(t *testing.T) {
	cases := map[float64]string{
		0:          "0,00",
		5.5:        "5,50",
		999.80:     "999,80",
		1234.567:   "1.234,57",
		1000000:    "1.000.000,00",
		-1234.5:    "-1.234,50",
		283.266667: "283,27",
	}
	for value, want := range cases {
		if got := formatBRL(value); got != want {
			t.Errorf("formatBRL(%v) = %q, want %q", value, got, want)
		}
	}
}

func TestGenerateSaleReceiptPDF(t *testing.T) {
	items := make([]domain.SaleReceiptItem, 0, 80)
	for range 80 { // força quebra de página na tabela de itens
		items = append(items, domain.SaleReceiptItem{Code: "00114", Description: "Maiô tradicional", Unit: "UN", Quantity: 1, UnitPrice: 529.90, Total: 529.90})
	}

	buf, err := GenerateSaleReceiptPDF(domain.SaleReceipt{
		SaleID:        uuid.New(),
		SaleAt:        time.Now(),
		PrintedAt:     time.Now(),
		TotalAmount:   529.90 * 80,
		PaymentMethod: "credit_card",
		Status:        "paid",
		Items:         items,
	})
	if err != nil {
		t.Fatalf("erro ao gerar recibo: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Fatal("o buffer gerado não é um PDF")
	}
}
