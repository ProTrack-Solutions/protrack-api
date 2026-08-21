package domain

import "time"

type UpdateDefaultPaymentMethodRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
}

type SubscriptionDetailsResponse struct {
	SubscriptionID         string                `json:"subscription_id"`
	CompanyID              string                `json:"company_id"`
	Status                 string                `json:"status"`
	CurrentPeriodStart     time.Time             `json:"current_period_start"`
	CurrentPeriodEnd       time.Time             `json:"current_period_end"`
	CanceledAt             time.Time             `json:"canceled_at,omitempty"`
	ExternalSubscriptionID string                `json:"external_subscription_id"`
	CreatedAt              time.Time             `json:"created_at"`
	UpdatedAt              time.Time             `json:"updated_at"`
	Plan                   PlanDetailsResponse   `json:"plan"`
	PaymentMethod          *PaymentMethodDetails `json:"payment_method"`
	Features               []PlanFeatureResponse `json:"features"`
}

type PlanDetailsResponse struct {
	ID              string `json:"id"`
	ExternalID      string `json:"external_id"`
	ExternalPriceID string `json:"external_price_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	PriceCents      int32  `json:"price_cents"`
	Currency        string `json:"currency"`
	BillingCycle    string `json:"billing_cycle"`
	Active          bool   `json:"active"`
	Highlight       bool   `json:"highlight"`
	Icon            string `json:"icon"`
}

type PaymentMethodDetails struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	CardBrand    string `json:"card_brand"`
	CardLast4    string `json:"card_last4"`
	CardExpMonth int32  `json:"card_exp_month"`
	CardExpYear  int32  `json:"card_exp_year"`
	IsDefault    bool   `json:"is_default"`
}

type PlanFeatureResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IsEnabled    bool   `json:"is_enabled"`
	DisplayOrder int    `json:"display_order"`
}

type CancelSubscriptionRequest struct {
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
	IdempotencyKey    string `json:"idempotency_key" binding:"required"`
	Reason            string `json:"reason,omitempty"`
}

type AddPaymentMethodRequest struct {
	StripePaymentMethodID string `json:"stripe_payment_method_id" binding:"required"`
	SetAsDefault          bool   `json:"set_as_default"`
	IdempotencyKey        string `json:"idempotency_key" binding:"required"`
}

type CountMenssageInvitAmountResponse struct {
	LimitMenssage  int     `json:"limit_menssage"`
	MenssageAmount int     `json:"menssage_amount"`
	Percentage     float64 `json:"percentage"`
}
