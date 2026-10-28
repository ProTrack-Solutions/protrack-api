package eligibility

import (
	"context"
	"testing"
	"time"

	companySettingsDomain "github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/demo"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	planFeaturesDomain "github.com/ProTrack-Solutions/protrack-api/internal/plan_features/domain"
	plansDomain "github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	subscriptionsDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	"github.com/google/uuid"
)

type fake struct {
	active, excess bool
	hasFeature     bool
	limit, sent    int64
	periodStart    time.Time
}

func (f *fake) GetSubscriptionByCompanyID(ctx context.Context, id uuid.UUID) (subscriptionsDomain.SubscriptionResponse, error) {
	return subscriptionsDomain.SubscriptionResponse{CurrentPeriodEnd: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)}, nil
}

func (f *fake) GetPlanByID(ctx context.Context, id uuid.UUID) (plansDomain.PlanResponse, error) {
	plan := plansDomain.PlanResponse{BillingCycle: "monthly"}
	if f.hasFeature {
		plan.Features = []planFeaturesDomain.PlanFeatureResponse{{FeatureKey: whatsAppFeature, LimitValue: f.limit}}
	}
	return plan, nil
}

func (f *fake) CountMessagesInPeriod(ctx context.Context, id uuid.UUID, start, end time.Time) (*int64, error) {
	f.periodStart = start
	return &f.sent, nil
}

func (f *fake) GetCompanySetting(ctx context.Context, id uuid.UUID, key enums.CompanySettingsKey) (companySettingsDomain.CompanySettingResponse, error) {
	value := map[enums.CompanySettingsKey]bool{enums.IsWhatsappActive: f.active, enums.IsExcessUsage: f.excess}[key]
	return companySettingsDomain.CompanySettingResponse{Value: value}, nil
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		f       fake
		allowed bool
	}{
		{"dentro da cota", fake{active: true, hasFeature: true, limit: 100, sent: 10}, true},
		{"cota esgotada sem excedente", fake{active: true, hasFeature: true, limit: 100, sent: 100}, false},
		{"cota esgotada com excedente", fake{active: true, hasFeature: true, limit: 100, sent: 150, excess: true}, true},
		{"WhatsApp desativado", fake{active: false, hasFeature: true, limit: 100}, false},
		{"plano sem WhatsApp", fake{active: true, hasFeature: false}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.f
			result, err := NewChecker(&f, &f, &f, &f).Check(context.Background(), uuid.New())
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if result.Allowed != tt.allowed {
				t.Fatalf("esperava allowed=%v, veio %+v", tt.allowed, result)
			}
			if !result.Allowed && result.Reason == "" {
				t.Error("recusa sem motivo")
			}
		})
	}
}

func TestCheckUsaPeriodoDoCiclo(t *testing.T) {
	f := fake{active: true, hasFeature: true, limit: 1}
	if _, err := NewChecker(&f, &f, &f, &f).Check(context.Background(), uuid.New()); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC); !f.periodStart.Equal(want) {
		t.Errorf("início do período: esperava %v, veio %v", want, f.periodStart)
	}
}

func TestCheckEmpresaDemo(t *testing.T) {
	f := fake{active: true, hasFeature: true, limit: 100}
	result, err := NewChecker(&f, &f, &f, &f).Check(context.Background(), demo.CompanyID)
	if err != nil || result.Allowed {
		t.Fatalf("empresa demo não pode enviar: %+v %v", result, err)
	}
}
