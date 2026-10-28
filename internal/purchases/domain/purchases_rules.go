package domain

import (
	"fmt"
	"math"
)

func Round2(v float64) float64 { return math.Round(v*100) / 100 }
func Round4(v float64) float64 { return math.Round(v*10000) / 10000 }

// CostLine item da nota usado no rateio
type CostLine struct {
	Quantity int32
	UnitCost float64
}

// AllocateLandedCosts rateia frete e outras despesas (somam) e o desconto
// (subtrai) entre os itens, proporcional ao valor de cada um. Devolve o custo
// unitário final de cada item, o total dos produtos e o total da nota. Se a
// nota só tem itens a custo zero, o rateio é pela quantidade.
func AllocateLandedCosts(lines []CostLine, freight, otherExpenses, discount float64) ([]float64, float64, float64, error) {
	var itemsTotal float64
	var quantity int64
	for _, l := range lines {
		itemsTotal += float64(l.Quantity) * l.UnitCost
		quantity += int64(l.Quantity)
	}
	itemsTotal = Round2(itemsTotal)

	total := Round2(itemsTotal + freight + otherExpenses - discount)
	if total < 0 {
		return nil, 0, 0, fmt.Errorf("%w: o desconto não pode ser maior que o total da nota", ErrInvalidPurchase)
	}

	extra := freight + otherExpenses - discount
	landed := make([]float64, len(lines))
	for i, l := range lines {
		var weight float64
		switch {
		case itemsTotal > 0:
			weight = float64(l.Quantity) * l.UnitCost / itemsTotal
		case quantity > 0:
			weight = float64(l.Quantity) / float64(quantity)
		}
		unit := l.UnitCost + extra*weight/float64(l.Quantity)
		landed[i] = Round4(math.Max(unit, 0))
	}

	return landed, itemsTotal, total, nil
}

// WeightedAverageCost custo médio ponderado após a entrada. Saldo negativo ou
// zerado não pesa: o custo passa a ser o da compra. Produto a granel não
// controla saldo, então fica com o último custo pago.
func WeightedAverageCost(stock int32, currentCost float64, quantity int32, landedCost float64, bulk bool) float64 {
	base := max(stock, 0)
	if bulk || base == 0 {
		return Round2(landedCost)
	}
	return Round2((float64(base)*currentCost + float64(quantity)*landedCost) / float64(base+quantity))
}

// ReverseAverageCost desfaz a entrada no custo médio. Quando a conta não é
// possível (saldo zerado após o estorno, resultado negativo ou granel), volta
// ao custo que o produto tinha antes do recebimento.
func ReverseAverageCost(stock int32, currentCost float64, quantity int32, landedCost, previousCost float64, bulk bool) float64 {
	remaining := stock - quantity
	if bulk || remaining <= 0 {
		return Round2(previousCost)
	}
	cost := (float64(stock)*currentCost - float64(quantity)*landedCost) / float64(remaining)
	if cost < 0 {
		return Round2(previousCost)
	}
	return Round2(cost)
}

// StatusAfterReceiving status do pedido depois de um recebimento ou estorno.
// Pedido encerrado continua encerrado.
func StatusAfterReceiving(current string, quantity, received int32) string {
	switch {
	case current == StatusClosed:
		return StatusClosed
	case received >= quantity:
		return StatusReceived
	case received > 0:
		return StatusPartial
	default:
		return StatusSent
	}
}

// ValidateInstallments confere se as parcelas fecham com o total da nota
// (em centavos). Nota de valor zero não gera parcelas.
func ValidateInstallments(installments []InstallmentRequest, total float64) error {
	if total == 0 {
		if len(installments) > 0 {
			return fmt.Errorf("%w: recebimento sem valor não gera parcelas", ErrInvalidPurchase)
		}
		return nil
	}
	if len(installments) == 0 {
		return fmt.Errorf("%w: informe ao menos uma parcela", ErrInvalidPurchase)
	}

	var cents int64
	for _, inst := range installments {
		cents += int64(math.Round(inst.Amount * 100))
	}
	if cents != int64(math.Round(total*100)) {
		return fmt.Errorf("%w (R$ %.2f)", ErrInstallmentsMismatch, total)
	}
	return nil
}

// OrderActions ações liberadas para o pedido no status atual
type OrderActions struct {
	CanEdit, CanSend, CanReceive, CanClose, CanCancel, CanEmail bool
}

func ActionsFor(status string, receivedQuantity int32) OrderActions {
	open := status == StatusDraft || status == StatusSent
	return OrderActions{
		CanEdit:    open && receivedQuantity == 0,
		CanSend:    status == StatusDraft,
		CanReceive: open || status == StatusPartial,
		CanClose:   status == StatusPartial,
		CanCancel:  open && receivedQuantity == 0,
		// E-mail só depois de enviado e enquanto ainda falta mercadoria
		CanEmail: status == StatusSent || status == StatusPartial,
	}
}
