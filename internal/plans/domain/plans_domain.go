package domain

import (
	"errors"
	"math"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	plansFeatures "github.com/ProTrack-Solutions/protrack-api/internal/plan_features/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrOriginalPriceNotGreater = errors.New("o preço original deve ser maior que o preço do plano")
	ErrInvalidTrialDays        = errors.New("trial_days não pode ser negativo")
)

type CreatePlanRequest struct {
	Name                string                                   `json:"name" validate:"required"`
	Description         string                                   `json:"description" validate:"required"`
	ValueAmount         float64                                  `json:"value_amount" validate:"required"`
	Currency            string                                   `json:"currency" validate:"required"`
	BillingCycle        string                                   `json:"billing_cycle" validate:"required"`
	Highlight           bool                                     `json:"highlight" validate:"required"`
	Icon                string                                   `json:"icon" validate:"required"`
	OriginalValueAmount float64                                  `json:"original_value_amount"` // Preço "de" exibido riscado no front (opcional, maior que value_amount)
	TrialDays           int32                                    `json:"trial_days" validate:"gte=0"`
	Features            []plansFeatures.CreatePlanFeatureRequest `json:"features" validate:"required"`
}

type PlanResponse struct {
	ID                 uuid.UUID                           `json:"id"`
	Name               string                              `json:"name"`
	Description        string                              `json:"description"`
	PriceCents         int32                               `json:"price_cents"`
	Currency           string                              `json:"currency"`
	BillingCycle       string                              `json:"billing_cycle"`
	Active             bool                                `json:"active"`
	CreatedAt          time.Time                           `json:"created_at"`
	UpdatedAt          time.Time                           `json:"updated_at"`
	ExternalId         string                              `json:"external_id"`
	ExternalPriceId    string                              `json:"external_price_id"`
	Highlight          bool                                `json:"highlight"`
	Icon               string                              `json:"icon"`
	OriginalPriceCents *int32                              `json:"original_price_cents"` // Preço "de" (riscado); null quando não há desconto
	TrialDays          int32                               `json:"trial_days"`
	Features           []plansFeatures.PlanFeatureResponse `json:"features"`
}

type UpdatePlanParams struct {
	Name         string  `json:"name" validate:"required"`
	Description  string  `json:"description" validate:"required"`
	ValueAmount  float64 `json:"value_amount" validate:"required"`
	Currency     string  `json:"currency" validate:"required"`
	BillingCycle string  `json:"billing_cycle" validate:"required"`
	// Ponteiros diferenciam "não enviado" (mantém o valor atual) de um valor explícito (ex: highlight=false)
	Highlight           *bool    `json:"highlight"`
	Icon                *string  `json:"icon"`
	OriginalValueAmount *float64 `json:"original_value_amount"` // Enviar 0 remove o preço "de" do plano
	TrialDays           *int32   `json:"trial_days" validate:"omitempty,gte=0"`
}

// ToPriceCents converte um valor em reais para centavos.
func ToPriceCents(value float64) int32 {
	return int32(math.Round(value * 100))
}

// ValidatePricing garante que o preço "de" (quando informado) seja maior que o preço cobrado
// e que os dias de teste não sejam negativos.
func ValidatePricing(priceCents int32, originalPriceCents pgtype.Int4, trialDays int32) error {
	if originalPriceCents.Valid && originalPriceCents.Int32 <= priceCents {
		return ErrOriginalPriceNotGreater
	}
	if trialDays < 0 {
		return ErrInvalidTrialDays
	}
	return nil
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

	if req.Highlight != nil {
		arg.Highlight = *req.Highlight
	}

	if req.Icon != nil && *req.Icon != "" {
		arg.Icon = *req.Icon
	}

	if req.OriginalValueAmount != nil {
		arg.OriginalPriceCents = pgconv.OptionalIntToPgInt4(int(ToPriceCents(*req.OriginalValueAmount)))
	}

	if req.TrialDays != nil {
		arg.TrialDays = *req.TrialDays
	}
}
