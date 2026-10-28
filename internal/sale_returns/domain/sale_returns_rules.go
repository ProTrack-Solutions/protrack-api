package domain

import (
	"math"
	"time"
)

func Round2(v float64) float64 { return math.Round(v*100) / 100 }

func cents(v float64) int64 { return int64(math.Round(v * 100)) }

// UnitAmount é o valor pago por unidade de uma linha da venda (preço menos o
// desconto rateado), com 4 casas para não perder centavos na multiplicação.
func UnitAmount(netTotal float64, quantity int32) float64 {
	if quantity <= 0 {
		return 0
	}
	return math.Round(netTotal/float64(quantity)*10000) / 10000
}

// LineReturnAmount é o valor devolvido ao devolver qty unidades de uma linha.
// Quando a devolução fecha a linha, devolve exatamente o que faltava (net_total
// menos o já devolvido), para a soma das devoluções bater com o valor pago.
func LineReturnAmount(netTotal, returnedTotal float64, quantity, returnedQty, qty int32) float64 {
	if returnedQty+qty >= quantity {
		return Round2(math.Max(netTotal-returnedTotal, 0))
	}
	return Round2(netTotal / float64(quantity) * float64(qty))
}

// Allocation é como o valor devolvido se distribui
type Allocation struct {
	// Parte dos itens novos paga com o valor devolvido
	ExchangeCredit float64
	// O que o cliente paga quando os itens novos valem mais
	Difference float64
	// Abatido das parcelas em aberto da venda
	DebtReduction float64
	// Sobra: vira vale-troca ou dinheiro de volta
	Rest float64
}

// Allocate distribui o valor devolvido: primeiro cobre os itens novos da troca,
// depois abate as parcelas em aberto (o cliente não recebe de volta o que ainda
// deve) e o resto fica para vale-troca ou reembolso.
func Allocate(returned, exchangeTotal, openDebt float64) Allocation {
	r, n, d := cents(returned), cents(exchangeTotal), cents(openDebt)

	used := min(r, n)
	leftover := r - used
	debt := min(leftover, max(d, 0))

	return Allocation{
		ExchangeCredit: float64(used) / 100,
		Difference:     float64(n-used) / 100,
		DebtReduction:  float64(debt) / 100,
		Rest:           float64(leftover-debt) / 100,
	}
}

// Deadline é o último instante para trocar. days = 0 é sem prazo (nil).
func Deadline(saleCreatedAt time.Time, days int) *time.Time {
	if days <= 0 {
		return nil
	}
	d := saleCreatedAt.AddDate(0, 0, days)
	return &d
}

// WithinWindow diz se a venda ainda está no prazo de troca
func WithinWindow(saleCreatedAt time.Time, days int, now time.Time) bool {
	d := Deadline(saleCreatedAt, days)
	return d == nil || !now.After(*d)
}

// ReduceInstallments abate amount das parcelas (já ordenadas da última para a
// primeira) e devolve o novo saldo de cada uma.
func ReduceInstallments(balances []float64, amount float64) []float64 {
	remaining := cents(amount)
	out := make([]float64, len(balances))
	for i, b := range balances {
		bc := cents(b)
		cut := min(bc, remaining)
		remaining -= cut
		out[i] = float64(bc-cut) / 100
	}
	return out
}
