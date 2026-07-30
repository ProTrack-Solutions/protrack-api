package domain

type CreateSubscriptionInput struct {
	Email     string
	Name      string
	CardToken string // pm_... (Payment Method Token)
	PriceID   string // price_... (ID do Preço/Plano cadastrado no Stripe ou ID dinâmico)
}

type CreateSubscriptionOutput struct {
	CustomerID     string `json:"customer_id"`
	SubscriptionID string `json:"subscription_id"`
	Status         string `json:"status"`
}
