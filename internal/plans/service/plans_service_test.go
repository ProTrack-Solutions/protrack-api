package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"
)

var errDatabase = errors.New("database error")

func newService(t *testing.T, repo *mocks.MockRepositoryInterface) *service.Service {
	t.Helper()
	return service.NewServiceWithRepo(repo, nil)
}

func buildDbPlan(id uuid.UUID, name string, active bool) db.Plan {
	now := time.Now().UTC()
	return db.Plan{
		ID:           pgconv.ParseUUIDToPgType(id),
		Name:         name,
		Description:  pgconv.ParseStringToPgText("Descrição do plano"),
		PriceCents:   2990,
		Currency:     pgconv.ParseStringToPgText("BRL"),
		BillingCycle: "MONTHLY",
		Active:       pgconv.BoolToPgBool(active),
		CreatedAt:    pgconv.TimeToPgTimestamptz(now),
		UpdatedAt:    pgconv.TimeToPgTimestamptz(now),
		ExternalID:   "ext_123",
	}
}

// ---------------------------------------------------------------------------
// CreatePlans Tests
// ---------------------------------------------------------------------------

func TestCreatePlans_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	newID := uuid.New()
	req := domain.CreatePlanRequest{
		Name:         "Plano Básico",
		Description:  "Plano inicial",
		ValueAmount:  29.90,
		Currency:     "BRL",
		BillingCycle: "MONTHLY",
		ExternalID:   "ext_basico",
		Highlight:    true,
		Icon:         "star",
	}

	repo.EXPECT().
		CreatePlans(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.CreatePlanParams) (pgtype.UUID, error) {
			if arg.Name != "Plano Básico" {
				t.Errorf("Name incorreto: %s", arg.Name)
			}
			if arg.PriceCents != 2990 { // 29.90 * 100
				t.Errorf("PriceCents incorreto: esperava 2990, obteve %d", arg.PriceCents)
			}
			if arg.BillingCycle != "MONTHLY" {
				t.Errorf("BillingCycle incorreto: %s", arg.BillingCycle)
			}
			if arg.ExternalID != "ext_basico" {
				t.Errorf("ExternalID incorreto: %s", arg.ExternalID)
			}
			if !arg.Highlight {
				t.Errorf("Highlight deveria ser true")
			}
			if arg.Icon != "star" {
				t.Errorf("Icon incorreto: %s", arg.Icon)
			}
			return pgconv.ParseUUIDToPgType(newID), nil
		})

	err := svc.CreatePlans(context.Background(), req)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestCreatePlans_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		CreatePlans(gomock.Any(), gomock.Any()).
		Return(pgtype.UUID{}, errDatabase)

	err := svc.CreatePlans(context.Background(), domain.CreatePlanRequest{
		Name:        "Plano Erro",
		ValueAmount: 10.00,
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
	if !errors.Is(err, errDatabase) {
		t.Errorf("erro incorreto: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetPlanByID Tests
// ---------------------------------------------------------------------------

func TestGetPlanByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	planID := uuid.New()
	dbPlan := buildDbPlan(planID, "Plano Pro", true)

	repo.EXPECT().
		GetPlanByID(gomock.Any(), pgconv.ParseUUIDToPgType(planID)).
		Return(dbPlan, nil)

	resp, err := svc.GetPlanByID(context.Background(), planID)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}

	if resp.ID != planID {
		t.Errorf("ID incorreto: esperava %v, obteve %v", planID, resp.ID)
	}
	if resp.Name != "Plano Pro" {
		t.Errorf("Name incorreto: %s", resp.Name)
	}
	if resp.PriceCents != 2990 {
		t.Errorf("PriceCents incorreto: %d", resp.PriceCents)
	}
	if !resp.Active {
		t.Errorf("Active deveria ser true")
	}
	if resp.ExternalId != "ext_123" {
		t.Errorf("ExternalId incorreto: %s", resp.ExternalId)
	}
}

func TestGetPlanByID_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetPlanByID(gomock.Any(), gomock.Any()).
		Return(db.Plan{}, errDatabase)

	_, err := svc.GetPlanByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListPlans Tests
// ---------------------------------------------------------------------------

func TestListPlans_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	id1 := uuid.New()
	id2 := uuid.New()
	dbPlans := []db.Plan{
		buildDbPlan(id1, "Plano A", true),
		buildDbPlan(id2, "Plano B", false),
	}

	repo.EXPECT().
		ListPlans(gomock.Any()).
		Return(dbPlans, nil)

	resps, err := svc.ListPlans(context.Background())
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}

	if len(resps) != 2 {
		t.Fatalf("esperava 2 planos, obteve %d", len(resps))
	}
	if resps[0].Name != "Plano A" || resps[1].Name != "Plano B" {
		t.Errorf("Nomes dos planos incorretos")
	}
}

func TestListPlans_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		ListPlans(gomock.Any()).
		Return([]db.Plan{}, nil)

	resps, err := svc.ListPlans(context.Background())
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resps) != 0 {
		t.Errorf("esperava lista vazia, obteve %d", len(resps))
	}
}

func TestListPlans_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		ListPlans(gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListPlans(context.Background())
	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListPlansByActiveStatus Tests
// ---------------------------------------------------------------------------

func TestListPlansByActiveStatus_ActiveTrue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	id := uuid.New()
	dbPlans := []db.Plan{buildDbPlan(id, "Plano Ativo", true)}

	repo.EXPECT().
		ListPlansByActiveStatus(gomock.Any(), pgconv.BoolToPgBool(true)).
		Return(dbPlans, nil)

	resps, err := svc.ListPlansByActiveStatus(context.Background(), true)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resps) != 1 {
		t.Fatalf("esperava 1 plano, obteve %d", len(resps))
	}
	if !resps[0].Active {
		t.Errorf("Active deveria ser true")
	}
}

func TestListPlansByActiveStatus_ActiveFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	id := uuid.New()
	dbPlans := []db.Plan{buildDbPlan(id, "Plano Inativo", false)}

	repo.EXPECT().
		ListPlansByActiveStatus(gomock.Any(), pgconv.BoolToPgBool(false)).
		Return(dbPlans, nil)

	resps, err := svc.ListPlansByActiveStatus(context.Background(), false)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resps) != 1 {
		t.Fatalf("esperava 1 plano, obteve %d", len(resps))
	}
	if resps[0].Active {
		t.Errorf("Active deveria ser false")
	}
}

func TestListPlansByActiveStatus_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		ListPlansByActiveStatus(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListPlansByActiveStatus(context.Background(), true)
	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// UpdatePlan Tests
// ---------------------------------------------------------------------------

func TestUpdatePlan_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	planID := uuid.New()
	currentDbPlan := buildDbPlan(planID, "Plano Antigo", true)

	repo.EXPECT().
		GetPlanByID(gomock.Any(), pgconv.ParseUUIDToPgType(planID)).
		Return(currentDbPlan, nil)

	repo.EXPECT().
		UpdatePlan(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.UpdatePlanParams) error {
			if arg.Name != "Plano Atualizado" {
				t.Errorf("Name incorreto: %s", arg.Name)
			}
			if arg.PriceCents != 4990 { // 49.90 * 100
				t.Errorf("PriceCents incorreto: esperava 4990, obteve %d", arg.PriceCents)
			}
			return nil
		})

	err := svc.UpdatePlan(context.Background(), planID, domain.UpdatePlanParams{
		Name:        "Plano Atualizado",
		ValueAmount: 49.90,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestUpdatePlan_ZeroValueAmount_KeepsCurrentPrice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	planID := uuid.New()
	currentDbPlan := buildDbPlan(planID, "Plano Básico", true)
	currentDbPlan.PriceCents = 3500

	repo.EXPECT().
		GetPlanByID(gomock.Any(), pgconv.ParseUUIDToPgType(planID)).
		Return(currentDbPlan, nil)

	repo.EXPECT().
		UpdatePlan(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, arg db.UpdatePlanParams) error {
			if arg.PriceCents != 3500 {
				t.Errorf("PriceCents deveria manter 3500, obteve %d", arg.PriceCents)
			}
			return nil
		})

	err := svc.UpdatePlan(context.Background(), planID, domain.UpdatePlanParams{
		Name:        "Plano Novo Nome Apenas",
		ValueAmount: 0,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestUpdatePlan_GetPlanByIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		GetPlanByID(gomock.Any(), gomock.Any()).
		Return(db.Plan{}, errDatabase)

	err := svc.UpdatePlan(context.Background(), uuid.New(), domain.UpdatePlanParams{
		Name: "Plano Teste",
	})

	if err == nil {
		t.Fatal("esperava erro de busca de plano")
	}
}

func TestUpdatePlan_UpdateRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	planID := uuid.New()
	currentDbPlan := buildDbPlan(planID, "Plano Antigo", true)

	repo.EXPECT().
		GetPlanByID(gomock.Any(), gomock.Any()).
		Return(currentDbPlan, nil)

	repo.EXPECT().
		UpdatePlan(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.UpdatePlan(context.Background(), planID, domain.UpdatePlanParams{
		Name: "Plano Novo",
	})

	if err == nil {
		t.Fatal("esperava erro de atualização do repositório")
	}
}

// ---------------------------------------------------------------------------
// TogglePlanActiveStatus Tests
// ---------------------------------------------------------------------------

func TestTogglePlanActiveStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	planID := uuid.New()

	repo.EXPECT().
		TogglePlanActiveStatus(gomock.Any(), db.TogglePlanActiveStatusParams{
			ID:     pgconv.ParseUUIDToPgType(planID),
			Active: pgconv.BoolToPgBool(false),
		}).
		Return(nil)

	err := svc.TogglePlanActiveStatus(context.Background(), planID, false)
	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestTogglePlanActiveStatus_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newService(t, repo)

	repo.EXPECT().
		TogglePlanActiveStatus(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.TogglePlanActiveStatus(context.Background(), uuid.New(), true)
	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}
