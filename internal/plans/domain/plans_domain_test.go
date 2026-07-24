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
