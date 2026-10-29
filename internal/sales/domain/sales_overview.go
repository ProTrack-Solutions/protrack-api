package domain

import (
	"errors"
	"fmt"
	"math"
	"time"
)

var ErrInvalidOverviewParams = errors.New("parâmetros inválidos")

// GetSalesOverviewParams período da visão geral de vendas, datas locais da
// empresa com o fim incluso. Sem start_date = desde a primeira venda;
// sem end_date = até hoje.
type GetSalesOverviewParams struct {
	StartDate string `json:"start_date" form:"start_date" binding:"omitempty,datetime=2006-01-02" example:"2026-10-01"`
	EndDate   string `json:"end_date" form:"end_date" binding:"omitempty,datetime=2006-01-02" example:"2026-10-29"`
}

type SalesOverviewTotals struct {
	SalesCount     int     `json:"sales_count"`
	Revenue        float64 `json:"revenue"`
	AverageTicket  float64 `json:"average_ticket"`
	Discounts      float64 `json:"discounts"`
	ItemsSold      int     `json:"items_sold"`
	CustomersCount int     `json:"customers_count"`
	CanceledCount  int     `json:"canceled_count"`
	CanceledAmount float64 `json:"canceled_amount"`
	// Devoluções registradas no período
	ReturnedAmount float64 `json:"returned_amount"`
	// Saldo em aberto das parcelas das vendas do período
	OpenReceivable    float64 `json:"open_receivable"`
	OverdueReceivable float64 `json:"overdue_receivable"`
}

type SalesOverviewPoint struct {
	Date       string  `json:"date"`
	SalesCount int     `json:"sales_count"`
	Revenue    float64 `json:"revenue"`
}

type GetSalesOverviewResponse struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	// Granularidade da série: day, week ou month, conforme o tamanho do período
	Period string              `json:"period"`
	Totals SalesOverviewTotals `json:"totals"`
	// Período anterior de mesmo tamanho; zerado quando o período é "desde o início"
	Previous SalesOverviewTotals  `json:"previous"`
	Series   []SalesOverviewPoint `json:"series"`
}

// AverageTicket faturamento / quantidade de vendas, com 2 casas
func AverageTicket(revenue float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return math.Round(revenue/float64(count)*100) / 100
}

// OverviewWindow intervalo [Start, End) em datas locais e a janela anterior do mesmo tamanho
type OverviewWindow struct {
	Start       time.Time
	End         time.Time
	PrevStart   time.Time
	Unit        string
	Step        string
	FirstBucket time.Time
	LastBucket  time.Time
}

// OverviewMaxDays tamanho máximo de um período escolhido pelo usuário
const OverviewMaxDays = 366

// BuildOverviewWindow valida o período e escolhe a granularidade da série:
// até 2 meses por dia, até ~6 meses por semana, acima disso por mês.
func BuildOverviewWindow(start, end time.Time) (OverviewWindow, error) {
	if end.Before(start) {
		return OverviewWindow{}, fmt.Errorf("%w: end_date antes de start_date", ErrInvalidOverviewParams)
	}
	days := int(end.Sub(start).Hours()/24) + 1

	w := OverviewWindow{
		Start:     start,
		End:       end.AddDate(0, 0, 1),
		PrevStart: start.AddDate(0, 0, -days),
	}

	switch {
	case days <= 62:
		w.Unit, w.Step = "day", "1 day"
		w.FirstBucket, w.LastBucket = start, end
	case days <= 186:
		// Semana começando na segunda, igual ao date_trunc('week') do Postgres
		w.Unit, w.Step = "week", "1 week"
		w.FirstBucket = start.AddDate(0, 0, -((int(start.Weekday()) + 6) % 7))
		w.LastBucket = end.AddDate(0, 0, -((int(end.Weekday()) + 6) % 7))
	default:
		w.Unit, w.Step = "month", "1 month"
		w.FirstBucket = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
		w.LastBucket = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	return w, nil
}
