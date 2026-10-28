package domain

import (
	"errors"
	"testing"
)

func TestAllocateLandedCosts(t *testing.T) {
	// Itens de R$ 300 e R$ 100; frete 50 + despesas 10 - desconto 20 = +40 rateado 75%/25%
	landed, itemsTotal, total, err := AllocateLandedCosts(
		[]CostLine{{Quantity: 10, UnitCost: 30}, {Quantity: 5, UnitCost: 20}},
		50, 10, 20,
	)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if itemsTotal != 400 || total != 440 {
		t.Fatalf("totais: esperava 400/440, veio %v/%v", itemsTotal, total)
	}
	// 30 + 40*0.75/10 = 33 ; 20 + 40*0.25/5 = 22
	if landed[0] != 33 || landed[1] != 22 {
		t.Errorf("custo rateado: esperava 33 e 22, veio %v", landed)
	}

	// A soma dos custos rateados reconstitui o total da nota
	if got := Round2(landed[0]*10 + landed[1]*5); got != total {
		t.Errorf("rateio não fecha com o total: %v != %v", got, total)
	}
}

func TestAllocateLandedCostsItensSemCusto(t *testing.T) {
	// Bonificação: itens a custo zero recebem o frete pela quantidade
	landed, _, total, err := AllocateLandedCosts([]CostLine{{Quantity: 3, UnitCost: 0}, {Quantity: 1, UnitCost: 0}}, 8, 0, 0)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if total != 8 || landed[0] != 2 || landed[1] != 2 {
		t.Errorf("esperava 2/2 e total 8, veio %v total %v", landed, total)
	}
}

func TestAllocateLandedCostsDescontoMaiorQueNota(t *testing.T) {
	_, _, _, err := AllocateLandedCosts([]CostLine{{Quantity: 1, UnitCost: 10}}, 0, 0, 11)
	if !errors.Is(err, ErrInvalidPurchase) {
		t.Fatalf("esperava ErrInvalidPurchase, veio %v", err)
	}
}

func TestWeightedAverageCost(t *testing.T) {
	tests := []struct {
		name     string
		stock    int32
		current  float64
		qty      int32
		landed   float64
		bulk     bool
		expected float64
	}{
		{"média ponderada", 10, 20, 10, 30, false, 25},
		{"estoque zerado assume o custo da compra", 0, 20, 5, 31.5, false, 31.5},
		{"estoque negativo não pesa", -3, 20, 5, 31.5, false, 31.5},
		{"granel fica com o último custo", 100, 20, 5, 31.5, true, 31.5},
		{"arredonda em centavos", 3, 10, 1, 11, false, 10.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WeightedAverageCost(tt.stock, tt.current, tt.qty, tt.landed, tt.bulk); got != tt.expected {
				t.Errorf("esperava %v, veio %v", tt.expected, got)
			}
		})
	}
}

func TestReverseAverageCostDesfazAEntrada(t *testing.T) {
	// Entrada: 10 a R$ 20 + 10 a R$ 30 => 20 a R$ 25. Estornar volta para R$ 20.
	if got := ReverseAverageCost(20, 25, 10, 30, 20, false); got != 20 {
		t.Errorf("esperava 20, veio %v", got)
	}
	// Saldo zera com o estorno: volta ao custo anterior gravado
	if got := ReverseAverageCost(10, 30, 10, 30, 18, false); got != 18 {
		t.Errorf("esperava 18, veio %v", got)
	}
	// Resultado negativo (vendas no meio do caminho) cai no custo anterior
	if got := ReverseAverageCost(12, 10, 10, 50, 9, false); got != 9 {
		t.Errorf("esperava 9, veio %v", got)
	}
	if got := ReverseAverageCost(0, 31.5, 5, 31.5, 20, true); got != 20 {
		t.Errorf("granel: esperava 20, veio %v", got)
	}
}

func TestStatusAfterReceiving(t *testing.T) {
	tests := []struct {
		current  string
		qty      int32
		received int32
		expected string
	}{
		{StatusSent, 10, 4, StatusPartial},
		{StatusPartial, 10, 10, StatusReceived},
		{StatusDraft, 10, 10, StatusReceived},
		{StatusReceived, 10, 6, StatusPartial}, // estorno parcial
		{StatusPartial, 10, 0, StatusSent},     // estornou tudo
		{StatusClosed, 10, 0, StatusClosed},    // encerrado continua encerrado
	}
	for _, tt := range tests {
		if got := StatusAfterReceiving(tt.current, tt.qty, tt.received); got != tt.expected {
			t.Errorf("%s %d/%d: esperava %s, veio %s", tt.current, tt.received, tt.qty, tt.expected, got)
		}
	}
}

func TestValidateInstallments(t *testing.T) {
	three := []InstallmentRequest{{Amount: 33.33}, {Amount: 33.33}, {Amount: 33.34}}
	if err := ValidateInstallments(three, 100); err != nil {
		t.Errorf("parcelas que fecham em centavos: %v", err)
	}
	if err := ValidateInstallments([]InstallmentRequest{{Amount: 50}}, 100); !errors.Is(err, ErrInstallmentsMismatch) {
		t.Errorf("soma diferente: esperava ErrInstallmentsMismatch, veio %v", err)
	}
	if err := ValidateInstallments(nil, 100); !errors.Is(err, ErrInvalidPurchase) {
		t.Errorf("sem parcelas: esperava ErrInvalidPurchase, veio %v", err)
	}
	if err := ValidateInstallments(nil, 0); err != nil {
		t.Errorf("nota zerada sem parcelas deve passar: %v", err)
	}
}

func TestActionsFor(t *testing.T) {
	draft := ActionsFor(StatusDraft, 0)
	if !draft.CanEdit || !draft.CanSend || !draft.CanReceive || !draft.CanCancel || draft.CanClose {
		t.Errorf("rascunho: %+v", draft)
	}
	partial := ActionsFor(StatusPartial, 3)
	if partial.CanEdit || partial.CanCancel || !partial.CanReceive || !partial.CanClose {
		t.Errorf("parcial: %+v", partial)
	}
	for _, status := range []string{StatusReceived, StatusClosed, StatusCanceled} {
		if a := ActionsFor(status, 0); a != (OrderActions{}) {
			t.Errorf("%s não deveria liberar ações: %+v", status, a)
		}
	}
}
