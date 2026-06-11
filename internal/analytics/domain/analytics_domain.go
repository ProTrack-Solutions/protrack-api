package domain

type ProfitMarginProductsResponse struct {
	Name      string  `json:"name" excel:"name"`
	CostPrice float64 `json:"cost_price" excel:"cost_price"`
	SalePrice float64 `json:"sale_price" excel:"sale_price"`
	Profit    float64 `json:"profit" excel:"profit"`
}

type ProfitMarginCategoryResponse struct {
	Name      string  `json:"name" excel:"name"`
	TotalCost float64 `json:"total_cost" excel:"total_cost"`
	TotalSale float64 `json:"total_price" excel:"total_price"`
	Profit    float64 `json:"profit" excel:"profit"`
}

type CalculateMarginDistributionResponse struct {
	ProductName     string  `json:"product_name" excel:"product_name"`
	Quantity        int64   `json:"quantity" excel:"quantity"`
	CostPrice       float64 `json:"cost_price" excel:"cost_price"`
	TotalInvestment float64 `json:"total_investment" excel:"total_investment"`
}
