package domain

import (
	"math"
	"time"
)

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
