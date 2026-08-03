package domain

import (
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
)

type CreateSubscriptionPaymentMethodRequest struct {
	GatewayPaymentMethodId string `json:"gateway_payment_method_id" validate:"required"`
	Type                   string `json:"type" validate:"required"`
	CardBrand              string `json:"card_brand" validate:"required"`
	CardLastFour           string `json:"card_last_four" validate:"required"`
	CardExpMonth           int32  `json:"card_exp_month" validate:"required"`
	CardExpYear            int32  `json:"card_exp_year" validate:"required"`
	IsDefault              bool   `json:"is_default"`
}

type SubscriptionPaymentMethodResponse struct {
	ID                     uuid.UUID `json:"id"`
	CompanyID              uuid.UUID `json:"company_id"`
	GatewayPaymentMethodID string    `json:"gateway_payment_method_id"`
	Type                   string    `json:"type"`
	CardBrand              string    `json:"card_brand"`
	CardLast4              string    `json:"card_last4"`
	CardExpMonth           int64     `json:"card_exp_month"`
	CardExpYear            int64     `json:"card_exp_year"`
	IsDefault              bool      `json:"is_default"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type UpdateSubscriptionPaymentMethodRequest struct {
	GatewayPaymentMethodID string `json:"gateway_payment_method_id"`
	Type                   string `json:"type"`
	CardBrand              string `json:"card_brand"`
	CardLastFour           string `json:"card_last_four"`
	CardExpMonth           int64  `json:"card_exp_month"`
	CardExpYear            int64  `json:"card_exp_year"`
}

func ApplyUpdateSubscriptionPaymentMethodParams(req UpdateSubscriptionPaymentMethodRequest, arg *db.UpdateSubscriptionPaymentMethodParams) {
	if req.GatewayPaymentMethodID != "" {
		arg.GatewayPaymentMethodID = req.GatewayPaymentMethodID
	}

	if req.Type != "" {
		arg.Type = req.Type
	}

	if req.CardBrand != "" {
		arg.CardBrand = pgconv.ParseStringToPgText(req.CardBrand)
	}

	if req.CardLastFour != "" {
		arg.CardLast4 = pgconv.ParseStringToPgText(req.CardLastFour)
	}

	if req.CardExpMonth != 0 {
		arg.CardExpMonth = pgconv.IntToPgInt4(int(req.CardExpMonth))
	}

	if req.CardExpYear != 0 {
		arg.CardExpYear = pgconv.IntToPgInt4(int(req.CardExpYear))
	}
}
