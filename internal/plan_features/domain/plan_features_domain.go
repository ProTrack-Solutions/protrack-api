package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreatePlanFeatureRequest struct {
	Name         string `json:"name"`
	IsEnabled    bool   `json:"is_enabled"`
	DisplayOrder int32  `json:"display_order"`
	FeatureKey   string `json:"feature_key"`
	LimitValue   int64  `json:"limit_value"`
}

type PlanFeatureResponse struct {
	ID           uuid.UUID `json:"id"`
	PlanID       uuid.UUID `json:"plan_id"`
	Name         string    `json:"name"`
	IsEnabled    bool      `json:"is_enabled"`
	DisplayOrder int32     `json:"display_order"`
	FeatureKey   string    `json:"feature_key"`
	LimitValue   int64     `json:"limit_value"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
