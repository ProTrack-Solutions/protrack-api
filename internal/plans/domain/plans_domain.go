package domain

import (
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
)

type CreatePlanRequest struct {
	Name         string  `json:"name" validate:"required"`
	Description  string  `json:"description" validate:"required"`
	ValueAmount  float64 `json:"value_amount" validate:"required"`
	Currency     string  `json:"currency" validate:"required"`
	BillingCycle string  `json:"billing_cycle" validate:"required"`
}

type PlanResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	PriceCents   int32     `json:"price_cents"`
	Currency     string    `json:"currency"`
	BillingCycle string    `json:"billing_cycle"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ExternalId   string    `json:"external_id"`
}

type UpdatePlanParams struct {
	Name         string  `json:"name" validate:"required"`
	Description  string  `json:"description" validate:"required"`
	ValueAmount  float64 `json:"value_amount" validate:"required"`
	Currency     string  `json:"currency" validate:"required"`
	BillingCycle string  `json:"billing_cycle" validate:"required"`
}

func ApplyUpdatePlanParams(
	req UpdatePlanParams,
	arg *db.UpdatePlanParams,
) {
	if req.Name != "" {
		arg.Name = req.Name
	}

	if req.Description != "" {
		arg.Description = pgconv.ParseStringToPgText(req.Description)
	}

	if req.Currency != "" {
		arg.Currency = pgconv.ParseStringToPgText(req.Currency)
	}

	if req.BillingCycle != "" {
		arg.BillingCycle = req.BillingCycle
	}

	if req.ValueAmount != 0 {
		priceCents := req.ValueAmount * 100
		arg.PriceCents = int32(priceCents)
	}
}
