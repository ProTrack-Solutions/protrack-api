package domain

import "time"

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
	TotalCategoriesInFlow  []GetCashInFlowByCategoryResponse  `json:"total_categories_in_flow"`
	TotalCategoriesOutFlow []GetCashOutFlowByCategoryResponse `json:"total_categories_out_flow"`
}
