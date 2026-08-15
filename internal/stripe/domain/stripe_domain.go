package domain

type CreateSubscriptionInput struct {
	Email          string
	Name           string
	CardToken      string // pm_... (Payment Method Token)
	PriceID        string // price_... (ID do Preço/Plano cadastrado no Stripe ou ID dinâmico)
	IdempotencyKey string // Chave de idempotência para garantir que a operação seja executada apenas uma vez
}

type CreateSubscriptionOutput struct {
	CustomerID             string `json:"customer_id"`
	SubscriptionID         string `json:"subscription_id"`
	Status                 string `json:"status"`
	ClientSecret           string `json:"client_secret,omitempty"`
	GatewayPaymentMethodId string `json:"gateway_payment_method_id"`
}
