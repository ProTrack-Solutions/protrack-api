package service

import (
	"context"
	"fmt"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
)

type col = domain.Column

func money(key, label string) col    { return col{Key: key, Label: label, Type: domain.ColumnMoney} }
func number(key, label string) col   { return col{Key: key, Label: label, Type: domain.ColumnNumber} }
func pct(key, label string) col      { return col{Key: key, Label: label, Type: domain.ColumnPercent} }
func text(key, label string) col     { return col{Key: key, Label: label, Type: domain.ColumnText} }
func date(key, label string) col     { return col{Key: key, Label: label, Type: domain.ColumnDate} }
func datetime(key, label string) col { return col{Key: key, Label: label, Type: domain.ColumnDateTime} }

func summary(label string, value float64, t domain.ColumnType) domain.SummaryItem {
	return domain.SummaryItem{Label: label, Value: value, Type: t}
}

// ─── Vendas ──────────────────────────────────────────────────────────────────

func (s *Service) salesReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportSales(ctx, db.ReportSalesParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var count, canceled int
	var subtotal, discount, total, cost float64
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		status := label(statusLabels, r.Status)
		if r.Canceled {
			status = statusLabels["canceled"]
			canceled++
		} else {
			count++
			subtotal += r.Subtotal
			discount += r.DiscountAmount
			total += r.TotalAmount
			cost += r.TotalCost
		}
		out = append(out, []any{
			p.localTime(r.SaleAt), r.CustomerName, r.SellerName,
			label(paymentMethodLabels, r.PaymentMethod), status,
			r.ItemsQuantity, r.Subtotal, r.DiscountAmount, r.TotalAmount,
			round2(r.TotalAmount - r.TotalCost),
		})
	}

	return domain.Report{
		Title:       "Vendas no período",
		Description: "Todas as vendas do período, uma por linha. Canceladas aparecem na lista mas ficam fora dos totais.",
		UsesPeriod:  true,
		Columns: []col{
			datetime("sale_at", "Data"), text("customer", "Cliente"), text("seller", "Vendedor"),
			text("payment_method", "Pagamento"), text("status", "Status"), number("items", "Itens"),
			money("subtotal", "Subtotal"), money("discount", "Desconto"), money("total", "Total"),
			money("profit", "Lucro bruto"),
		},
		Rows:   out,
		Totals: []any{"Total", nil, nil, nil, nil, nil, round2(subtotal), round2(discount), round2(total), round2(total - cost)},
		Summary: []domain.SummaryItem{
			summary("Faturamento", round2(total), domain.ColumnMoney),
			summary("Vendas", float64(count), domain.ColumnNumber),
			summary("Ticket médio", ticket(total, count), domain.ColumnMoney),
			summary("Descontos", round2(discount), domain.ColumnMoney),
			summary("Canceladas", float64(canceled), domain.ColumnNumber),
		},
	}, nil
}

func (s *Service) salesByDayReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportSalesByDay(ctx, db.ReportSalesByDayParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var count int
	var subtotal, discount, total, cost, best float64
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		count += int(r.SalesCount)
		subtotal += r.Subtotal
		discount += r.Discount
		total += r.Total
		cost += r.Cost
		best = max(best, r.Total)
		out = append(out, []any{
			p.localDate(r.Day), r.SalesCount, r.Subtotal, r.Discount, r.Total,
			ticket(r.Total, int(r.SalesCount)), round2(r.Total - r.Cost),
		})
	}

	days := len(rows)
	return domain.Report{
		Title:       "Faturamento por dia",
		Description: "Vendas consolidadas dia a dia, incluindo os dias sem movimento.",
		UsesPeriod:  true,
		Columns: []col{
			date("day", "Dia"), number("sales", "Vendas"), money("subtotal", "Bruto"),
			money("discount", "Descontos"), money("total", "Líquido"), money("ticket", "Ticket médio"),
			money("profit", "Lucro bruto"),
		},
		Rows:   out,
		Totals: []any{"Total", count, round2(subtotal), round2(discount), round2(total), ticket(total, count), round2(total - cost)},
		Summary: []domain.SummaryItem{
			summary("Faturamento", round2(total), domain.ColumnMoney),
			summary("Média por dia", ticket(total, days), domain.ColumnMoney),
			summary("Melhor dia", round2(best), domain.ColumnMoney),
			summary("Lucro bruto", round2(total-cost), domain.ColumnMoney),
		},
	}, nil
}

func (s *Service) salesByPaymentMethodReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportSalesByPaymentMethod(ctx, db.ReportSalesByPaymentMethodParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var count int
	var total, discount float64
	for _, r := range rows {
		count += int(r.SalesCount)
		total += r.Total
		discount += r.Discount
	}

	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{
			label(paymentMethodLabels, r.PaymentMethod), r.SalesCount, r.Discount, r.Total,
			ticket(r.Total, int(r.SalesCount)), percent(r.Total, total),
		})
	}

	return domain.Report{
		Title:       "Vendas por forma de pagamento",
		Description: "Quanto entrou por cada forma de pagamento e a participação de cada uma no faturamento.",
		UsesPeriod:  true,
		Columns: []col{
			text("payment_method", "Forma de pagamento"), number("sales", "Vendas"), money("discount", "Descontos"),
			money("total", "Total"), money("ticket", "Ticket médio"), pct("share", "Participação"),
		},
		Rows:   out,
		Totals: []any{"Total", count, round2(discount), round2(total), ticket(total, count), totalShare(total)},
		Summary: []domain.SummaryItem{
			summary("Faturamento", round2(total), domain.ColumnMoney),
			summary("Vendas", float64(count), domain.ColumnNumber),
			summary("Formas usadas", float64(len(rows)), domain.ColumnNumber),
		},
	}, nil
}

func (s *Service) salesBySellerReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportSalesBySeller(ctx, db.ReportSalesBySellerParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var count int
	var total, discount, cost float64
	for _, r := range rows {
		count += int(r.SalesCount)
		total += r.Total
		discount += r.Discount
		cost += r.Cost
	}

	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{
			r.SellerName, r.SalesCount, r.Discount, r.Total, ticket(r.Total, int(r.SalesCount)),
			round2(r.Total - r.Cost), percent(r.Total, total),
		})
	}

	return domain.Report{
		Title:       "Vendas por vendedor",
		Description: "Desempenho de cada usuário que registrou vendas no período. Base para cálculo de comissões.",
		UsesPeriod:  true,
		Columns: []col{
			text("seller", "Vendedor"), number("sales", "Vendas"), money("discount", "Descontos"),
			money("total", "Total vendido"), money("ticket", "Ticket médio"), money("profit", "Lucro bruto"),
			pct("share", "Participação"),
		},
		Rows:   out,
		Totals: []any{"Total", count, round2(discount), round2(total), ticket(total, count), round2(total - cost), totalShare(total)},
		Summary: []domain.SummaryItem{
			summary("Faturamento", round2(total), domain.ColumnMoney),
			summary("Vendedores", float64(len(rows)), domain.ColumnNumber),
			summary("Ticket médio", ticket(total, count), domain.ColumnMoney),
		},
	}, nil
}

func (s *Service) salesByCustomerReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportSalesByCustomer(ctx, db.ReportSalesByCustomerParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var count int
	var total float64
	for _, r := range rows {
		count += int(r.SalesCount)
		total += r.Total
	}

	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{
			r.CustomerName, r.Phone, r.SalesCount, r.Total, ticket(r.Total, int(r.SalesCount)),
			percent(r.Total, total), p.localTime(r.LastSaleAt), r.BalanceDue,
		})
	}

	return domain.Report{
		Title:       "Vendas por cliente",
		Description: "Ranking dos clientes que mais compraram no período, com o saldo devedor atual de cada um.",
		UsesPeriod:  true,
		Columns: []col{
			text("customer", "Cliente"), text("phone", "Telefone"), number("sales", "Compras"),
			money("total", "Total comprado"), money("ticket", "Ticket médio"), pct("share", "Participação"),
			datetime("last_sale_at", "Última compra"), money("balance_due", "Saldo devedor"),
		},
		Rows:   out,
		Totals: []any{"Total", nil, count, round2(total), ticket(total, count), totalShare(total), nil, nil},
		Summary: []domain.SummaryItem{
			summary("Clientes atendidos", float64(len(rows)), domain.ColumnNumber),
			summary("Faturamento", round2(total), domain.ColumnMoney),
			summary("Média por cliente", ticket(total, len(rows)), domain.ColumnMoney),
		},
	}, nil
}

// ─── Produtos e estoque ──────────────────────────────────────────────────────

// abcClass classifica pela participação acumulada antes do item: os primeiros
// itens que somam até 80% da receita são A, até 95% são B e o resto C. Assim o
// item que cruza a linha dos 80% ainda é A.
func abcClass(cumulativeBefore float64) string {
	switch {
	case cumulativeBefore < 80:
		return "A"
	case cumulativeBefore < 95:
		return "B"
	default:
		return "C"
	}
}

func (s *Service) productsABCReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportSalesByProduct(ctx, db.ReportSalesByProductParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var quantity int
	var gross, discount, net, cost float64
	for _, r := range rows {
		quantity += int(r.Quantity)
		gross += r.Gross
		discount += r.Discount
		net += r.Net
		cost += r.Cost
	}

	classes := map[string]int{}
	var cumulative float64
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		share := percent(r.Net, net)
		class := abcClass(cumulative)
		cumulative += share
		classes[class]++
		out = append(out, []any{
			class, r.ProductName, r.CategoryName, r.Quantity, r.Net, r.Cost,
			round2(r.Net - r.Cost), percent(r.Net-r.Cost, r.Net), share, round2(min(cumulative, 100)),
		})
	}

	return domain.Report{
		Title:       "Curva ABC e margem por produto",
		Description: "Produtos ordenados pela receita. Classe A soma 80% do faturamento, B os próximos 15% e C o restante.",
		UsesPeriod:  true,
		Columns: []col{
			text("class", "Classe"), text("product", "Produto"), text("category", "Categoria"),
			number("quantity", "Qtd. vendida"), money("net", "Receita líquida"), money("cost", "Custo (CMV)"),
			money("profit", "Lucro bruto"), pct("margin", "Margem"), pct("share", "Participação"),
			pct("cumulative", "Acumulado"),
		},
		Rows:   out,
		Totals: []any{"Total", nil, nil, quantity, round2(net), round2(cost), round2(net - cost), percent(net-cost, net), totalShare(net), nil},
		Summary: []domain.SummaryItem{
			summary("Receita líquida", round2(net), domain.ColumnMoney),
			summary("Lucro bruto", round2(net-cost), domain.ColumnMoney),
			summary("Margem média", percent(net-cost, net), domain.ColumnPercent),
			summary("Produtos classe A", float64(classes["A"]), domain.ColumnNumber),
		},
		Notes: costNotes(),
	}, nil
}

func (s *Service) profitCategoryReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportSalesByCategory(ctx, db.ReportSalesByCategoryParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var quantity int
	var net, cost float64
	for _, r := range rows {
		quantity += int(r.Quantity)
		net += r.Net
		cost += r.Cost
	}

	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, []any{
			r.CategoryName, r.ProductsCount, r.Quantity, r.Net, r.Cost, round2(r.Net - r.Cost),
			percent(r.Net-r.Cost, r.Net), percent(r.Net, net),
		})
	}

	return domain.Report{
		Title:       "Margem por categoria",
		Description: "Receita, custo e margem de cada categoria de produto no período.",
		UsesPeriod:  true,
		Columns: []col{
			text("category", "Categoria"), number("products", "Produtos vendidos"), number("quantity", "Qtd. vendida"),
			money("net", "Receita líquida"), money("cost", "Custo (CMV)"), money("profit", "Lucro bruto"),
			pct("margin", "Margem"), pct("share", "Participação"),
		},
		Rows:   out,
		Totals: []any{"Total", nil, quantity, round2(net), round2(cost), round2(net - cost), percent(net-cost, net), totalShare(net)},
		Summary: []domain.SummaryItem{
			summary("Receita líquida", round2(net), domain.ColumnMoney),
			summary("Lucro bruto", round2(net-cost), domain.ColumnMoney),
			summary("Margem média", percent(net-cost, net), domain.ColumnPercent),
		},
		Notes: costNotes(),
	}, nil
}

func (s *Service) inventoryReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportInventory(ctx, p.companyID)
	if err != nil {
		return domain.Report{}, err
	}

	var quantity, zeroed int
	var costValue, saleValue float64
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		var qty any = r.Quantity
		var rowCost, rowSale any
		if r.SellInBulk {
			// Venda a granel não controla saldo: não entra na valorização
			qty = "Granel"
		} else {
			quantity += int(r.Quantity)
			if r.Quantity <= 0 {
				zeroed++
			}
			c := round2(float64(r.Quantity) * r.CostPrice)
			v := round2(float64(r.Quantity) * r.SalePrice)
			costValue += c
			saleValue += v
			rowCost, rowSale = c, v
		}
		out = append(out, []any{
			r.ProductName, r.CategoryName, r.Barcode, r.Unit, qty, r.CostPrice, r.SalePrice,
			percent(r.SalePrice-r.CostPrice, r.SalePrice), rowCost, rowSale,
		})
	}

	return domain.Report{
		Title:       "Posição de estoque",
		Description: "Saldo atual de cada produto e o valor do estoque a preço de custo e de venda. Não depende do período.",
		UsesPeriod:  false,
		Columns: []col{
			text("product", "Produto"), text("category", "Categoria"), text("barcode", "Código de barras"),
			text("unit", "Unidade"), number("quantity", "Saldo"), money("cost_price", "Custo unitário"),
			money("sale_price", "Preço de venda"), pct("markup_margin", "Margem"),
			money("cost_value", "Valor a custo"), money("sale_value", "Valor a venda"),
		},
		Rows:   out,
		Totals: []any{"Total", nil, nil, nil, quantity, nil, nil, nil, round2(costValue), round2(saleValue)},
		Summary: []domain.SummaryItem{
			summary("Produtos", float64(len(rows)), domain.ColumnNumber),
			summary("Valor a custo", round2(costValue), domain.ColumnMoney),
			summary("Valor a venda", round2(saleValue), domain.ColumnMoney),
			summary("Sem saldo", float64(zeroed), domain.ColumnNumber),
		},
	}, nil
}

func (s *Service) stockMovementsReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportStockMovements(ctx, db.ReportStockMovementsParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var entries, exits int
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		if r.Quantity > 0 {
			entries += int(r.Quantity)
		} else {
			exits -= int(r.Quantity)
		}
		out = append(out, []any{
			p.localTime(r.CreatedAt), r.ProductName, label(stockMovementLabels, r.Type), r.Quantity, r.Reason, r.UserName,
		})
	}

	return domain.Report{
		Title:       "Movimentações de estoque",
		Description: "Entradas, saídas por venda, cancelamentos e ajustes de estoque no período (kardex).",
		UsesPeriod:  true,
		Columns: []col{
			datetime("created_at", "Data"), text("product", "Produto"), text("type", "Tipo"),
			number("quantity", "Quantidade"), text("reason", "Motivo"), text("user", "Usuário"),
		},
		Rows:   out,
		Totals: []any{"Saldo", nil, nil, entries - exits, nil, nil},
		Summary: []domain.SummaryItem{
			summary("Movimentações", float64(len(rows)), domain.ColumnNumber),
			summary("Unidades que entraram", float64(entries), domain.ColumnNumber),
			summary("Unidades que saíram", float64(exits), domain.ColumnNumber),
		},
	}, nil
}

// ─── Financeiro ──────────────────────────────────────────────────────────────

func (s *Service) dreReport(ctx context.Context, p period) (domain.Report, error) {
	revenue, err := s.repo.ReportDreRevenue(ctx, db.ReportDreRevenueParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}
	expenses, err := s.repo.ReportDreExpenses(ctx, db.ReportDreExpensesParams{
		CompanyID: p.companyID, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	return buildDRE(revenue, expenses), nil
}

// buildDRE monta a DRE gerencial. Percentuais são sobre a receita líquida.
func buildDRE(revenue db.ReportDreRevenueRow, expenses []db.ReportDreExpensesRow) domain.Report {
	net := revenue.Net
	grossProfit := round2(net - revenue.Cost)

	var totalExpenses float64
	for _, e := range expenses {
		totalExpenses += e.Total
	}
	result := round2(grossProfit - totalExpenses)

	line := func(desc string, value float64) []any {
		return []any{desc, round2(value), percent(value, net)}
	}

	rows := [][]any{
		line("(+) Receita bruta de vendas", revenue.Gross),
		line("(-) Descontos concedidos", -revenue.Discount),
		line("(=) Receita líquida", net),
		line("(-) Custo das mercadorias vendidas (CMV)", -revenue.Cost),
		line("(=) Lucro bruto", grossProfit),
		line("(-) Despesas operacionais", -totalExpenses),
	}
	for _, e := range expenses {
		rows = append(rows, line("      "+e.CategoryName, -e.Total))
	}
	rows = append(rows, line("(=) Resultado do período", result))

	return domain.Report{
		Title:       "DRE gerencial",
		Description: "Demonstração do resultado pelo regime de competência: vendas pela data da venda e despesas pelo vencimento.",
		UsesPeriod:  true,
		Columns: []col{
			text("description", "Descrição"), money("value", "Valor"), pct("share", "% da receita líquida"),
		},
		Rows: rows,
		Summary: []domain.SummaryItem{
			summary("Receita líquida", round2(net), domain.ColumnMoney),
			summary("Lucro bruto", grossProfit, domain.ColumnMoney),
			summary("Margem bruta", percent(grossProfit, net), domain.ColumnPercent),
			summary("Resultado", result, domain.ColumnMoney),
			summary("Margem líquida", percent(result, net), domain.ColumnPercent),
		},
		Notes: append(costNotes(), "Despesas vêm das contas a pagar com vencimento no período, exceto as canceladas."),
	}
}

func (s *Service) paymentsReceivedReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportPaymentsReceived(ctx, db.ReportPaymentsReceivedParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var total float64
	customers := map[string]struct{}{}
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		total += r.AmountPaid
		customers[r.CustomerName] = struct{}{}
		out = append(out, []any{
			p.localTime(r.PaymentDate), r.CustomerName, r.PaymentMethodName, r.UserName, r.AmountPaid, r.Notes,
		})
	}

	return domain.Report{
		Title:       "Recebimentos de crediário",
		Description: "Pagamentos de parcelas e vendas a prazo recebidos no período.",
		UsesPeriod:  true,
		Columns: []col{
			datetime("payment_date", "Data"), text("customer", "Cliente"), text("payment_method", "Forma de pagamento"),
			text("user", "Recebido por"), money("amount", "Valor"), text("notes", "Observação"),
		},
		Rows:   out,
		Totals: []any{"Total", nil, nil, nil, round2(total), nil},
		Summary: []domain.SummaryItem{
			summary("Total recebido", round2(total), domain.ColumnMoney),
			summary("Recebimentos", float64(len(rows)), domain.ColumnNumber),
			summary("Clientes", float64(len(customers)), domain.ColumnNumber),
		},
	}, nil
}

func (s *Service) openInstallmentsReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportOpenInstallments(ctx, db.ReportOpenInstallmentsParams{
		CompanyID: p.companyID, StartDate: p.startDate(), EndDate: p.endDate(), Today: p.todayDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var balance, overdue float64
	var overdueCount int
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		balance += r.Balance
		situation := "A vencer"
		if r.DaysOverdue > 0 {
			situation = "Vencida"
			overdue += r.Balance
			overdueCount++
		}
		out = append(out, []any{
			p.localDate(r.DueDate), r.CustomerName, r.Phone, installmentLabel(r.InstallmentNumber, r.TotalInstallments),
			situation, r.DaysOverdue, r.TotalAmount, r.Balance,
		})
	}

	return domain.Report{
		Title:       "Parcelas em aberto e inadimplência",
		Description: "Parcelas do crediário com vencimento no período que ainda têm saldo, com os dias de atraso.",
		UsesPeriod:  true,
		Columns: []col{
			date("due_date", "Vencimento"), text("customer", "Cliente"), text("phone", "Telefone"),
			text("installment", "Parcela"), text("situation", "Situação"), number("days_overdue", "Dias de atraso"),
			money("amount", "Valor da parcela"), money("balance", "Saldo em aberto"),
		},
		Rows:   out,
		Totals: []any{"Total", nil, nil, nil, nil, nil, nil, round2(balance)},
		Summary: []domain.SummaryItem{
			summary("Saldo em aberto", round2(balance), domain.ColumnMoney),
			summary("Vencido", round2(overdue), domain.ColumnMoney),
			summary("Parcelas vencidas", float64(overdueCount), domain.ColumnNumber),
			summary("Inadimplência", percent(overdue, balance), domain.ColumnPercent),
		},
	}, nil
}

func (s *Service) billsPayableReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportBillsPayable(ctx, db.ReportBillsPayableParams{
		CompanyID: p.companyID, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var amount, paid, open, overdue float64
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		status := r.Status
		if status != "canceled" {
			amount += r.Amount
			paid += r.AmountPaid
			if status != "paid" {
				open += r.Amount - r.AmountPaid
				if r.DueDate.Valid && r.DueDate.Time.Before(p.today) {
					overdue += r.Amount - r.AmountPaid
				}
			}
		}
		out = append(out, []any{
			p.localDate(r.DueDate), r.VendorName, r.CategoryName, r.Description, label(statusLabels, status),
			r.Amount, r.AmountPaid, p.localDate(r.PaymentDate),
		})
	}

	return domain.Report{
		Title:       "Contas a pagar",
		Description: "Contas com vencimento no período, pagas ou em aberto. Canceladas ficam fora dos totais.",
		UsesPeriod:  true,
		Columns: []col{
			date("due_date", "Vencimento"), text("vendor", "Fornecedor"), text("category", "Categoria"),
			text("description", "Descrição"), text("status", "Status"), money("amount", "Valor"),
			money("amount_paid", "Valor pago"), date("payment_date", "Pago em"),
		},
		Rows:   out,
		Totals: []any{"Total", nil, nil, nil, nil, round2(amount), round2(paid), nil},
		Summary: []domain.SummaryItem{
			summary("Total no período", round2(amount), domain.ColumnMoney),
			summary("Pago", round2(paid), domain.ColumnMoney),
			summary("Em aberto", round2(open), domain.ColumnMoney),
			summary("Vencido", round2(overdue), domain.ColumnMoney),
		},
	}, nil
}

func (s *Service) cashSessionsReport(ctx context.Context, p period) (domain.Report, error) {
	rows, err := s.repo.ReportCashSessions(ctx, db.ReportCashSessionsParams{
		CompanyID: p.companyID, Tz: p.tz, StartDate: p.startDate(), EndDate: p.endDate(),
	})
	if err != nil {
		return domain.Report{}, err
	}

	var sales, received, supplies, withdrawals, difference float64
	var shortages int
	out := make([][]any, 0, len(rows))
	for _, r := range rows {
		sales += r.SalesTotal
		received += r.ReceivedTotal
		supplies += r.SuppliesTotal
		withdrawals += r.WithdrawalsTotal
		difference += r.Difference
		if r.Difference < 0 {
			shortages++
		}

		// Caixa ainda aberto não tem fechamento: os totais só existem após fechar
		var expected, counted, diff any
		if r.Status == "closed" {
			expected, counted, diff = r.ExpectedCash, r.CountedCash, r.Difference
		}
		out = append(out, []any{
			p.localTime(r.OpenedAt), p.localTime(r.ClosedAt), r.UserName, label(statusLabels, r.Status),
			r.OpeningAmount, r.SalesCount, r.SalesTotal, r.ReceivedTotal, r.SuppliesTotal, r.WithdrawalsTotal,
			expected, counted, diff,
		})
	}

	return domain.Report{
		Title:       "Fechamentos de caixa",
		Description: "Turnos de caixa abertos no período com vendas, sangrias, suprimentos e a diferença na conferência.",
		UsesPeriod:  true,
		Columns: []col{
			datetime("opened_at", "Abertura"), datetime("closed_at", "Fechamento"), text("user", "Operador"),
			text("status", "Status"), money("opening", "Fundo de troco"), number("sales_count", "Vendas"),
			money("sales_total", "Total vendido"), money("received", "Recebimentos"), money("supplies", "Suprimentos"),
			money("withdrawals", "Sangrias"), money("expected", "Esperado em dinheiro"),
			money("counted", "Contado"), money("difference", "Diferença"),
		},
		Rows: out,
		Totals: []any{"Total", nil, nil, nil, nil, nil, round2(sales), round2(received), round2(supplies),
			round2(withdrawals), nil, nil, round2(difference)},
		Summary: []domain.SummaryItem{
			summary("Turnos", float64(len(rows)), domain.ColumnNumber),
			summary("Total vendido", round2(sales), domain.ColumnMoney),
			summary("Diferença acumulada", round2(difference), domain.ColumnMoney),
			summary("Turnos com falta", float64(shortages), domain.ColumnNumber),
		},
	}, nil
}

// ─── Auxiliares ──────────────────────────────────────────────────────────────

func ticket(total float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return round2(total / float64(count))
}

// totalShare participação da linha de total: 100% quando houve movimento
func totalShare(total float64) any {
	if total == 0 {
		return nil
	}
	return 100.0
}

func installmentLabel(number, total int32) string {
	if total <= 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", number, total)
}

func costNotes() []string {
	return []string{"O custo (CMV) usa o custo do produto registrado no momento da venda; vendas anteriores a esse registro usam o custo atual do cadastro."}
}
