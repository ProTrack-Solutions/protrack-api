package domain

type UpdateDefaultPaymentMethodRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
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
