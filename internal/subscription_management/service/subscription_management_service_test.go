package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	metaWhatsappMocks "github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/mocks"
	planFeaturesDomain "github.com/ProTrack-Solutions/protrack-api/internal/plan_features/domain"
	plansDomain "github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_management/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_management/service"
	subscriptionsDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers e Fixtures
// ---------------------------------------------------------------------------

var errDatabase = errors.New("database error")

// deps agrupa os mocks das dependências do Service, para facilitar a
// configuração de cada cenário de teste.
type deps struct {
	repo         *mocks.MockRepositoryInterface
	subscription *mocks.MockSubscriptionsServiceInterface
	plans        *mocks.MockPlansServiceInterface
	metaWhatsapp *metaWhatsappMocks.MockServiceInterface
}

// newSvc cria um Service com os mocks injetados.
func newSvc(t *testing.T, d deps) *service.Service {
	t.Helper()
	return service.NewServiceWithDeps(d.repo, d.subscription, d.plans, d.metaWhatsapp)
}

func newDeps(ctrl *gomock.Controller) deps {
	return deps{
		repo:         mocks.NewMockRepositoryInterface(ctrl),
		subscription: mocks.NewMockSubscriptionsServiceInterface(ctrl),
		plans:        mocks.NewMockPlansServiceInterface(ctrl),
		metaWhatsapp: metaWhatsappMocks.NewMockServiceInterface(ctrl),
	}
}

// buildSubscription constrói uma assinatura ativa cujo período atual termina em currentPeriodEnd.
func buildSubscription(planID uuid.UUID, currentPeriodEnd time.Time) subscriptionsDomain.SubscriptionResponse {
	return subscriptionsDomain.SubscriptionResponse{
		ID:               uuid.New(),
		PlanID:           planID,
		Status:           "active",
		CurrentPeriodEnd: currentPeriodEnd,
	}
}

// buildPlan constrói um plano com a feature max_whatsapp_integration limitada a limit.
func buildPlan(billingCycle string, limit int64) plansDomain.PlanResponse {
	return plansDomain.PlanResponse{
		ID:           uuid.New(),
		BillingCycle: billingCycle,
		Features: []planFeaturesDomain.PlanFeatureResponse{
			{FeatureKey: "max_whatsapp_integration", LimitValue: limit},
		},
	}
}

// ---------------------------------------------------------------------------
// CountMenssageInvitAmount
// ---------------------------------------------------------------------------

func TestCountMenssageInvitAmount_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(t, d)

	companyID := uuid.New()
	periodEnd := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	sub := buildSubscription(uuid.New(), periodEnd)
	plan := buildPlan("monthly", 100)

	var count int64 = 40

	d.subscription.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), companyID).
		Return(sub, nil)

	d.plans.EXPECT().
		GetPlanByID(gomock.Any(), sub.PlanID).
		Return(plan, nil)

	d.metaWhatsapp.EXPECT().
		CountMessagesInPeriod(gomock.Any(), companyID, periodEnd.AddDate(0, -1, 0), periodEnd).
		Return(&count, nil)

	resp, err := svc.CountMenssageInvitAmount(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.LimitMenssage != 100 {
		t.Errorf("LimitMenssage incorreto: %d", resp.LimitMenssage)
	}
	if resp.MenssageAmount != 40 {
		t.Errorf("MenssageAmount incorreto: %d", resp.MenssageAmount)
	}
	if resp.Percentage != 40 {
		t.Errorf("Percentage incorreto: %v", resp.Percentage)
	}
}

func TestCountMenssageInvitAmount_YearlyBillingCycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(t, d)

	companyID := uuid.New()
	periodEnd := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	sub := buildSubscription(uuid.New(), periodEnd)
	plan := buildPlan("yearly", 100)

	var count int64 = 10

	d.subscription.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), companyID).
		Return(sub, nil)

	d.plans.EXPECT().
		GetPlanByID(gomock.Any(), sub.PlanID).
		Return(plan, nil)

	// para plano anual, o período de apuração deve começar 1 ano antes do fim do período atual.
	d.metaWhatsapp.EXPECT().
		CountMessagesInPeriod(gomock.Any(), companyID, periodEnd.AddDate(-1, 0, 0), periodEnd).
		Return(&count, nil)

	_, err := svc.CountMenssageInvitAmount(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestCountMenssageInvitAmount_NoMessagesSent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(t, d)

	companyID := uuid.New()
	periodEnd := time.Now().UTC()
	sub := buildSubscription(uuid.New(), periodEnd)
	plan := buildPlan("monthly", 100)

	d.subscription.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), companyID).
		Return(sub, nil)

	d.plans.EXPECT().
		GetPlanByID(gomock.Any(), sub.PlanID).
		Return(plan, nil)

	// nenhuma mensagem enviada ainda no período: repositório retorna nil.
	d.metaWhatsapp.EXPECT().
		CountMessagesInPeriod(gomock.Any(), companyID, gomock.Any(), gomock.Any()).
		Return(nil, nil)

	resp, err := svc.CountMenssageInvitAmount(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.MenssageAmount != 0 {
		t.Errorf("MenssageAmount incorreto: %d", resp.MenssageAmount)
	}
	if resp.Percentage != 0 {
		t.Errorf("Percentage incorreto: %v", resp.Percentage)
	}
}

func TestCountMenssageInvitAmount_PlanWithoutWhatsappFeature(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(t, d)

	companyID := uuid.New()
	periodEnd := time.Now().UTC()
	sub := buildSubscription(uuid.New(), periodEnd)
	plan := plansDomain.PlanResponse{ID: uuid.New(), BillingCycle: "monthly"} // sem feature max_whatsapp_integration

	var count int64 = 5

	d.subscription.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), companyID).
		Return(sub, nil)

	d.plans.EXPECT().
		GetPlanByID(gomock.Any(), sub.PlanID).
		Return(plan, nil)

	d.metaWhatsapp.EXPECT().
		CountMessagesInPeriod(gomock.Any(), companyID, gomock.Any(), gomock.Any()).
		Return(&count, nil)

	// sem limite configurado (limit=0), o percentual não pode ser calculado por
	// divisão por zero, então deve retornar 0 em vez de +Inf/NaN.
	resp, err := svc.CountMenssageInvitAmount(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.LimitMenssage != 0 {
		t.Errorf("LimitMenssage incorreto: %d", resp.LimitMenssage)
	}
	if resp.Percentage != 0 {
		t.Errorf("Percentage incorreto, esperava 0, obteve: %v", resp.Percentage)
	}
}

func TestCountMenssageInvitAmount_SubscriptionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(t, d)

	d.subscription.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(subscriptionsDomain.SubscriptionResponse{}, errDatabase)

	_, err := svc.CountMenssageInvitAmount(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro ao buscar assinatura")
	}
}

func TestCountMenssageInvitAmount_PlanError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(t, d)

	sub := buildSubscription(uuid.New(), time.Now().UTC())

	d.subscription.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(sub, nil)

	d.plans.EXPECT().
		GetPlanByID(gomock.Any(), sub.PlanID).
		Return(plansDomain.PlanResponse{}, errDatabase)

	_, err := svc.CountMenssageInvitAmount(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro ao buscar plano")
	}
}

func TestCountMenssageInvitAmount_CountMessagesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	d := newDeps(ctrl)
	svc := newSvc(t, d)

	sub := buildSubscription(uuid.New(), time.Now().UTC())
	plan := buildPlan("monthly", 100)

	d.subscription.EXPECT().
		GetSubscriptionByCompanyID(gomock.Any(), gomock.Any()).
		Return(sub, nil)

	d.plans.EXPECT().
		GetPlanByID(gomock.Any(), sub.PlanID).
		Return(plan, nil)

	d.metaWhatsapp.EXPECT().
		CountMessagesInPeriod(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.CountMenssageInvitAmount(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro ao contar mensagens")
	}
}
