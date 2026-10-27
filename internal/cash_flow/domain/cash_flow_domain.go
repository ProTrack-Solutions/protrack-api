package domain

import (
	"errors"
	"math"
	"time"
)

var ErrInvalidOverviewParams = errors.New("parâmetros inválidos")

type CashFlowSummaryResponse struct {
	TotalInflow  float64 `json:"total_inflow"`
	TotalOutflow float64 `json:"total_outflow"`
	NetBalance   float64 `json:"net_balance"`
}

type CashFlowSummaryRequest struct {
	StartAt time.Time `form:"startAt" time_format:"2006-01-02T15:04:05Z07:00"`
	EndAt   time.Time `form:"endAt"   time_format:"2006-01-02T15:04:05Z07:00"`
}

type GetCashFlowHistoryProjectionsResponse struct {
	Date               string  `json:"date"`
	TotalInflow        float64 `json:"total_inflow"`
	TotalOutflow       float64 `json:"total_outflow"`
	AccumulatedBalance float64 `json:"accumulated_balance"`
}

type GetCashInFlowByCategoryResponse struct {
	NameCategory     string  `json:"name_category"`
	TotalInFlow      float64 `json:"total_inflow"`
	PercentageInFlow float64 `json:"percentage_in_flow"`
}

type GetCashOutFlowByCategoryResponse struct {
	NameCategory     string  `json:"name_category"`
	TotalOutFlow     float64 `json:"total_outflow"`
	PercentageInFlow float64 `json:"percentage_in_flow"`
}

type GetCashFlowPeriodResponse struct {
	Month        string  `json:"mount"`
	TotalInflow  float64 `json:"total_inflow"`
	TotalOutflow float64 `json:"total_outflow"`
}

type GetCashFlowResponse struct {
	Date         string  `json:"date"`
	TotalInflow  float64 `json:"total_inflow"`
	TotalOutflow float64 `json:"total_outflow"`
}

type GetTotalSummaryParams struct {
	Quantity int64  `json:"quantity" form:"quantity"`
	Period   string `json:"period" form:"period"`
}

type TotalSummaty struct {
	Period             string  `json:"period"`
	TotalPeriodOutFlow float64 `json:"total_period_outflow"`
	TotalPeriodInFlow  float64 `json:"total_period_inflow"`
	TotalPeriod        float64 `json:"total_period"`
}

type GetTotalSummaryResponse struct {
	Summary                []TotalSummaty                     `json:"summary"`
	TotalOutFlow           float64                            `json:"total_outflow"`
	TotalInFlow            float64                            `json:"total_inflow"`
	Total                  float64                            `json:"total"`
	Projection             float64                            `json:"projection"`
	TotalCategoriesInFlow  []GetCashInFlowByCategoryResponse  `json:"total_categories_in_flow"`
	TotalCategoriesOutFlow []GetCashOutFlowByCategoryResponse `json:"total_categories_out_flow"`
}

func CalcularProjecao(historico []float64, periodoFuturo int) float64 {
	n := float64(len(historico))

	var sumX float64
	var sumY float64
	var sumXY float64
	var sumXX float64

	for index, y := range historico {
		x := float64(index + 1) // O período X começa em 1, 2, 3...

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += math.Pow(x, 2)
	}

	a := (n*sumXY - sumX*sumY) / (n*sumXX - math.Pow(sumX, 2))

	b := (sumY - a*sumX) / n

	projecao := (a * float64(periodoFuturo)) + b

	return math.Round(projecao*100) / 100
}

// GetOverviewParams filtros da visão geral: Quantity períodos de Period
// (dia, semana ou mês) terminando no período atual, mais ForecastDays de previsão.
type GetOverviewParams struct {
	Period       string `json:"period" form:"period" binding:"omitempty,oneof=day week month" example:"day"`
	Quantity     int    `json:"quantity" form:"quantity" binding:"omitempty,min=1" example:"30"`
	ForecastDays int    `json:"forecast_days" form:"forecast_days" binding:"omitempty,min=1,max=90" example:"30"`
}

type OverviewTotals struct {
	Inflow       float64 `json:"inflow"`
	Outflow      float64 `json:"outflow"`
	Net          float64 `json:"net"`
	InflowCount  int     `json:"inflow_count"`
	OutflowCount int     `json:"outflow_count"`
}

type OverviewPoint struct {
	Date    string  `json:"date"`
	Inflow  float64 `json:"inflow"`
	Outflow float64 `json:"outflow"`
	Net     float64 `json:"net"`
	// Balance resultado acumulado desde o início do intervalo
	Balance float64 `json:"balance"`
}

type OverviewShare struct {
	Name       string  `json:"name"`
	Total      float64 `json:"total"`
	Percentage float64 `json:"percentage"`
	Count      int     `json:"count,omitempty"`
}

type OverviewForecastPoint struct {
	Date       string  `json:"date"`
	Receivable float64 `json:"receivable"`
	Payable    float64 `json:"payable"`
	// Balance saldo previsto acumulado (a receber − a pagar) até o dia
	Balance float64 `json:"balance"`
}

type OverviewForecast struct {
	Days              int                     `json:"days"`
	Receivable        float64                 `json:"receivable"`
	Payable           float64                 `json:"payable"`
	Net               float64                 `json:"net"`
	OverdueReceivable float64                 `json:"overdue_receivable"`
	OverduePayable    float64                 `json:"overdue_payable"`
	Series            []OverviewForecastPoint `json:"series"`
}

type OverviewMovement struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind" enums:"inflow,outflow"`
	Description string    `json:"description"`
	Detail      string    `json:"detail"`
	Amount      float64   `json:"amount"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type GetOverviewResponse struct {
	Period    string `json:"period"`
	Quantity  int    `json:"quantity"`
	StartDate string `json:"start_date"`
	// EndDate último dia (inclusivo) do intervalo
	EndDate  string         `json:"end_date"`
	Totals   OverviewTotals `json:"totals"`
	Previous OverviewTotals `json:"previous"`
	// Projection tendência linear do resultado para o próximo período
	Projection            float64            `json:"projection"`
	Series                []OverviewPoint    `json:"series"`
	InflowByCategory      []OverviewShare    `json:"inflow_by_category"`
	OutflowByCategory     []OverviewShare    `json:"outflow_by_category"`
	InflowByPaymentMethod []OverviewShare    `json:"inflow_by_payment_method"`
	Forecast              OverviewForecast   `json:"forecast"`
	RecentMovements       []OverviewMovement `json:"recent_movements"`
}
