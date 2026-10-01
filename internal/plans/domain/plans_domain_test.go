package domain_test

import (
	"testing"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildOriginalDbUpdatePlanParams(id uuid.UUID) db.UpdatePlanParams {
	return db.UpdatePlanParams{
		ID:           pgconv.ParseUUIDToPgType(id),
		Name:         "Plano Básico",
		Description:  pgconv.ParseStringToPgText("Descrição antiga"),
		PriceCents:   2990,
		Currency:     pgconv.ParseStringToPgText("BRL"),
		BillingCycle: "MONTHLY",
	}
}

// ---------------------------------------------------------------------------
// ApplyUpdatePlanParams Tests
// ---------------------------------------------------------------------------

func TestApplyUpdatePlanParams_UpdatesAllFields(t *testing.T) {
	id := uuid.New()
	arg := buildOriginalDbUpdatePlanParams(id)

	req := domain.UpdatePlanParams{
		Name:         "Plano Pro",
		Description:  "Nova descrição do plano pro",
		ValueAmount:  49.90,
		Currency:     "USD",
		BillingCycle: "YEARLY",
	}

	domain.ApplyUpdatePlanParams(req, &arg)

	if arg.Name != "Plano Pro" {
		t.Errorf("Name: esperava 'Plano Pro', obteve '%s'", arg.Name)
	}
	if arg.Description.String != "Nova descrição do plano pro" {
		t.Errorf("Description: esperava 'Nova descrição do plano pro', obteve '%s'", arg.Description.String)
	}
	if arg.Currency.String != "USD" {
		t.Errorf("Currency: esperava 'USD', obteve '%s'", arg.Currency.String)
	}
	if arg.BillingCycle != "YEARLY" {
		t.Errorf("BillingCycle: esperava 'YEARLY', obteve '%s'", arg.BillingCycle)
	}
	// 49.90 * 100 = 4990 cents
	if arg.PriceCents != 4990 {
		t.Errorf("PriceCents: esperava 4990, obteve %d", arg.PriceCents)
	}
}

func TestApplyUpdatePlanParams_DoesNotOverwriteWithZeroValues(t *testing.T) {
	id := uuid.New()
	arg := buildOriginalDbUpdatePlanParams(id)

	req := domain.UpdatePlanParams{
		Name:         "",
		Description:  "",
		ValueAmount:  0,
		Currency:     "",
		BillingCycle: "",
	}

	domain.ApplyUpdatePlanParams(req, &arg)

	if arg.Name != "Plano Básico" {
		t.Errorf("Name não deveria mudar, obteve '%s'", arg.Name)
	}
	if arg.Description.String != "Descrição antiga" {
		t.Errorf("Description não deveria mudar, obteve '%s'", arg.Description.String)
	}
	if arg.Currency.String != "BRL" {
		t.Errorf("Currency não deveria mudar, obteve '%s'", arg.Currency.String)
	}
	if arg.BillingCycle != "MONTHLY" {
		t.Errorf("BillingCycle não deveria mudar, obteve '%s'", arg.BillingCycle)
	}
	if arg.PriceCents != 2990 {
		t.Errorf("PriceCents não deveria mudar, obteve %d", arg.PriceCents)
	}
}

func TestApplyUpdatePlanParams_PartialUpdate(t *testing.T) {
	id := uuid.New()
	arg := buildOriginalDbUpdatePlanParams(id)

	req := domain.UpdatePlanParams{
		Name:        "Plano Master",
		ValueAmount: 99.00,
	}

	domain.ApplyUpdatePlanParams(req, &arg)

	if arg.Name != "Plano Master" {
		t.Errorf("Name: esperava 'Plano Master', obteve '%s'", arg.Name)
	}
	if arg.PriceCents != 9900 {
		t.Errorf("PriceCents: esperava 9900, obteve %d", arg.PriceCents)
	}
	// Campos mantidos
	if arg.Description.String != "Descrição antiga" {
		t.Errorf("Description não deveria mudar, obteve '%s'", arg.Description.String)
	}
	if arg.BillingCycle != "MONTHLY" {
		t.Errorf("BillingCycle não deveria mudar, obteve '%s'", arg.BillingCycle)
	}
}

// ---------------------------------------------------------------------------
// Struct Field Assignment Tests
// ---------------------------------------------------------------------------

func TestCreatePlanRequest_FieldAssignment(t *testing.T) {
	req := domain.CreatePlanRequest{
		Name:         "Plano Enterprise",
		Description:  "Plano para grandes empresas",
		ValueAmount:  199.99,
		Currency:     "BRL",
		BillingCycle: "YEARLY",
	}

	if req.Name != "Plano Enterprise" {
		t.Errorf("Name incorreto")
	}
	if req.Description != "Plano para grandes empresas" {
		t.Errorf("Description incorreta")
	}
	if req.ValueAmount != 199.99 {
		t.Errorf("ValueAmount incorreto: %f", req.ValueAmount)
	}
	if req.Currency != "BRL" {
		t.Errorf("Currency incorreta")
	}
	if req.BillingCycle != "YEARLY" {
		t.Errorf("BillingCycle incorreto")
	}
}

func TestPlanResponse_FieldAssignment(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	resp := domain.PlanResponse{
		ID:           id,
		Name:         "Plano Start",
		Description:  "Plano de entrada",
		PriceCents:   1490,
		Currency:     "BRL",
		BillingCycle: "MONTHLY",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
	if resp.PriceCents != 1490 {
		t.Errorf("PriceCents incorreto: %d", resp.PriceCents)
	}
	if !resp.Active {
		t.Errorf("Active deveria ser true")
	}
}

// ---------------------------------------------------------------------------
// Preço original (desconto) e trial
// ---------------------------------------------------------------------------

func TestApplyUpdatePlanParams_NilPointersKeepCurrentValues(t *testing.T) {
	arg := buildOriginalDbUpdatePlanParams(uuid.New())
	arg.Highlight = true
	arg.Icon = "star"
	arg.OriginalPriceCents = pgconv.IntToPgInt4(5990)
	arg.TrialDays = 7

	domain.ApplyUpdatePlanParams(domain.UpdatePlanParams{Name: "Novo"}, &arg)

	if !arg.Highlight {
		t.Error("Highlight não deveria ser alterado quando não enviado")
	}
	if arg.Icon != "star" {
		t.Errorf("Icon não deveria ser alterado, obteve %q", arg.Icon)
	}
	if !arg.OriginalPriceCents.Valid || arg.OriginalPriceCents.Int32 != 5990 {
		t.Errorf("OriginalPriceCents não deveria ser alterado, obteve %+v", arg.OriginalPriceCents)
	}
	if arg.TrialDays != 7 {
		t.Errorf("TrialDays não deveria ser alterado, obteve %d", arg.TrialDays)
	}
}

func TestApplyUpdatePlanParams_SetsNewFields(t *testing.T) {
	arg := buildOriginalDbUpdatePlanParams(uuid.New())
	arg.Highlight = true

	highlight := false
	icon := "rocket"
	original := 59.90
	trial := int32(7)

	domain.ApplyUpdatePlanParams(domain.UpdatePlanParams{
		Highlight:           &highlight,
		Icon:                &icon,
		OriginalValueAmount: &original,
		TrialDays:           &trial,
	}, &arg)

	if arg.Highlight {
		t.Error("Highlight deveria ser false")
	}
	if arg.Icon != "rocket" {
		t.Errorf("Icon incorreto: %q", arg.Icon)
	}
	if !arg.OriginalPriceCents.Valid || arg.OriginalPriceCents.Int32 != 5990 {
		t.Errorf("OriginalPriceCents incorreto: %+v", arg.OriginalPriceCents)
	}
	if arg.TrialDays != 7 {
		t.Errorf("TrialDays incorreto: %d", arg.TrialDays)
	}
}

func TestApplyUpdatePlanParams_ZeroOriginalValueRemovesDiscount(t *testing.T) {
	arg := buildOriginalDbUpdatePlanParams(uuid.New())
	arg.OriginalPriceCents = pgconv.IntToPgInt4(5990)

	zero := 0.0
	domain.ApplyUpdatePlanParams(domain.UpdatePlanParams{OriginalValueAmount: &zero}, &arg)

	if arg.OriginalPriceCents.Valid {
		t.Errorf("OriginalPriceCents deveria ser NULL, obteve %+v", arg.OriginalPriceCents)
	}
}

func TestValidatePricing(t *testing.T) {
	tests := []struct {
		name     string
		price    int32
		original int
		trial    int32
		wantErr  error
	}{
		{"sem desconto e sem trial", 2990, 0, 0, nil},
		{"desconto válido com trial", 2990, 5990, 7, nil},
		{"preço original igual ao preço", 2990, 2990, 0, domain.ErrOriginalPriceNotGreater},
		{"preço original menor que o preço", 2990, 1990, 0, domain.ErrOriginalPriceNotGreater},
		{"trial negativo", 2990, 0, -1, domain.ErrInvalidTrialDays},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidatePricing(tt.price, pgconv.OptionalIntToPgInt4(tt.original), tt.trial)
			if err != tt.wantErr {
				t.Errorf("esperava %v, obteve %v", tt.wantErr, err)
			}
		})
	}
}
