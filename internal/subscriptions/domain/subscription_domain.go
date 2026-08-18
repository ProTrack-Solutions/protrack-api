package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateSubscriptionRequest struct {
	PlanId                 uuid.UUID
	PaymentMethodsId       uuid.UUID
	ExternalSubscriptionID string
	Status                 string
	CurrentPeriodStart     time.Time
	CurrentPeriodEnd       time.Time
}

type SubscriptionResponse struct {
	ID                     uuid.UUID `json:"id"`
	CompanyID              uuid.UUID `json:"company_id"`
	PlanID                 uuid.UUID `json:"plan_id"`
	PaymentMethodID        uuid.UUID `json:"payment_method_id"`
	ExternalSubscriptionID string    `json:"external_subscription_id"`
	Status                 string    `json:"status"`
	CurrentPeriodStart     time.Time `json:"current_period_start"`
	CurrentPeriodEnd       time.Time `json:"current_period_end"`
	CanceledAt             time.Time `json:"canceled_at"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type UpdateSubscriptionPlanRequest struct {
	PlanId uuid.UUID `json:"plan_id"`
}

type UpdateSubscriptionMethodRequest struct {
	PaymentMethodId uuid.UUID `json:"payment_mathod_id"`
}

type UpdateSubscriptionStatusRequest struct {
	Status           string    `json:"status"`
	CurrentPeriodEnd time.Time `json:"current_period_end"`
}

type ListSubscriptionsDueOnRequest struct {
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	CurrentPeriodEnd_2 time.Time `json:"current_period_end_2"`
}
