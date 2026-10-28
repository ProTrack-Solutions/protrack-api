package domain

import (
	"testing"
	"time"
)

func TestLineReturnAmount(t *testing.T) {
	// Linha: 3 x R$ 10,00 com R$ 1,00 de desconto rateado = R$ 29,00
	if got := LineReturnAmount(29, 0, 3, 0, 1); got != 9.67 {
		t.Errorf("1 de 3: esperava 9.67, veio %v", got)
	}
	if got := LineReturnAmount(29, 9.67, 3, 1, 1); got != 9.67 {
		t.Errorf("2 de 3: esperava 9.67, veio %v", got)
	}
	// A última unidade fecha a linha: 29 - 19.34 = 9.66, a soma bate com o pago
	if got := LineReturnAmount(29, 19.34, 3, 2, 1); got != 9.66 {
		t.Errorf("3 de 3: esperava 9.66, veio %v", got)
	}
	if got := LineReturnAmount(29, 0, 3, 0, 3); got != 29 {
		t.Errorf("linha inteira: esperava 29, veio %v", got)
	}
}

func TestAllocate(t *testing.T) {
	cases := []struct {
		name                            string
		returned, exchange, debt        float64
		credit, diff, reduction, remain float64
	}{
		{"devolução à vista", 100, 0, 0, 0, 0, 0, 100},
		{"devolução a prazo abate a dívida primeiro", 100, 0, 60, 0, 0, 60, 40},
		{"devolução a prazo com dívida maior", 100, 0, 250, 0, 0, 100, 0},
		{"troca com diferença a pagar", 80, 120, 0, 80, 40, 0, 0},
		{"troca com sobra vira crédito", 120, 80, 0, 80, 0, 0, 40},
		{"troca de venda a prazo mantém a dívida", 100, 100, 300, 100, 0, 0, 0},
		{"troca com sobra abate a dívida", 150, 100, 30, 100, 0, 30, 20},
		{"centavos", 0.3, 0.1, 0.1, 0.1, 0, 0.1, 0.1},
	}
	for _, c := range cases {
		a := Allocate(c.returned, c.exchange, c.debt)
		if a.ExchangeCredit != c.credit || a.Difference != c.diff || a.DebtReduction != c.reduction || a.Rest != c.remain {
			t.Errorf("%s: veio %+v", c.name, a)
		}
	}
}

func TestWithinWindow(t *testing.T) {
	sale := time.Date(2026, 10, 1, 15, 0, 0, 0, time.UTC)

	if !WithinWindow(sale, 30, time.Date(2026, 10, 31, 15, 0, 0, 0, time.UTC)) {
		t.Error("no último instante do prazo ainda vale")
	}
	if WithinWindow(sale, 30, time.Date(2026, 10, 31, 15, 0, 1, 0, time.UTC)) {
		t.Error("um segundo depois do prazo não vale")
	}
	if !WithinWindow(sale, 0, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("prazo 0 é sem prazo")
	}
	if Deadline(sale, 0) != nil {
		t.Error("sem prazo não tem data limite")
	}
}

func TestReduceInstallments(t *testing.T) {
	// Parcelas da última para a primeira
	got := ReduceInstallments([]float64{50, 50, 33.33}, 70)
	want := []float64{0, 30, 33.33}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("esperava %v, veio %v", want, got)
		}
	}
}

func TestUnitAmount(t *testing.T) {
	if got := UnitAmount(29, 3); got != 9.6667 {
		t.Errorf("esperava 9.6667, veio %v", got)
	}
	if got := UnitAmount(10, 0); got != 0 {
		t.Errorf("quantidade zero: esperava 0, veio %v", got)
	}
}
