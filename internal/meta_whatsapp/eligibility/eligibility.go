// Package eligibility decide se uma empresa pode enviar mensagens de WhatsApp
// agora. É a mesma regra para o envio automático (vendas vencidas) e para o
// manual (lembrete de cobrança), para os dois nunca divergirem.
package eligibility

import (
	"context"
	"fmt"
	"strings"
	"time"

	companySettingsDomain "github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/demo"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	plansDomain "github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	subscriptionsDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	"github.com/google/uuid"
)

// Feature do plano que libera o WhatsApp; o limite é a cota de mensagens do período
const whatsAppFeature = "max_whatsapp_integration"

type Subscriptions interface {
	GetSubscriptionByCompanyID(ctx context.Context, companyID uuid.UUID) (subscriptionsDomain.SubscriptionResponse, error)
}

type Plans interface {
	GetPlanByID(ctx context.Context, planId uuid.UUID) (plansDomain.PlanResponse, error)
}

type Messages interface {
	CountMessagesInPeriod(ctx context.Context, companyId uuid.UUID, periodStart, periodEnd time.Time) (*int64, error)
}

type Settings interface {
	GetCompanySetting(ctx context.Context, companyId uuid.UUID, key enums.CompanySettingsKey) (companySettingsDomain.CompanySettingResponse, error)
}

type Checker struct {
	subscriptions Subscriptions
	plans         Plans
	messages      Messages
	settings      Settings
}

func NewChecker(subscriptions Subscriptions, plans Plans, messages Messages, settings Settings) *Checker {
	return &Checker{subscriptions: subscriptions, plans: plans, messages: messages, settings: settings}
}

// Result Allowed=false traz em Reason o motivo, em texto para o usuário
type Result struct {
	Allowed bool
	Reason  string
}

func denied(reason string) Result { return Result{Reason: reason} }

func (c *Checker) Check(ctx context.Context, companyID uuid.UUID) (Result, error) {
	// Os clientes da empresa demo são fictícios: nunca envia WhatsApp
	if demo.IsDemoCompany(companyID) {
		return denied("o envio de WhatsApp fica desativado na empresa de demonstração"), nil
	}

	active, err := c.boolSetting(ctx, companyID, enums.IsWhatsappActive)
	if err != nil {
		return Result{}, err
	}
	if !active {
		return denied("o WhatsApp está desativado nas configurações da empresa"), nil
	}

	sub, err := c.subscriptions.GetSubscriptionByCompanyID(ctx, companyID)
	if err != nil {
		return Result{}, fmt.Errorf("assinatura da empresa %s: %w", companyID, err)
	}
	plan, err := c.plans.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return Result{}, fmt.Errorf("plano %s da empresa %s: %w", sub.PlanID, companyID, err)
	}

	var limit int64
	hasFeature := false
	for _, ft := range plan.Features {
		if ft.FeatureKey == whatsAppFeature {
			hasFeature = true
			limit = ft.LimitValue
		}
	}
	if !hasFeature {
		return denied("o plano atual não inclui envio de WhatsApp"), nil
	}

	periodStart := PeriodStart(sub.CurrentPeriodEnd, plan.BillingCycle)
	sent, err := c.messages.CountMessagesInPeriod(ctx, companyID, periodStart, sub.CurrentPeriodEnd)
	if err != nil {
		return Result{}, fmt.Errorf("contagem de mensagens da empresa %s: %w", companyID, err)
	}
	if sent == nil {
		return Result{}, fmt.Errorf("contagem de mensagens indisponível para a empresa %s", companyID)
	}
	if *sent < limit {
		return Result{Allowed: true}, nil
	}

	// Cota esgotada: só segue se a empresa aceitou pagar o excedente
	excess, err := c.boolSetting(ctx, companyID, enums.IsExcessUsage)
	if err != nil {
		return Result{}, err
	}
	if !excess {
		return denied("o limite de mensagens do plano foi atingido; ative o uso excedente para continuar enviando"), nil
	}
	return Result{Allowed: true}, nil
}

func (c *Checker) boolSetting(ctx context.Context, companyID uuid.UUID, key enums.CompanySettingsKey) (bool, error) {
	setting, err := c.settings.GetCompanySetting(ctx, companyID, key)
	if err != nil {
		return false, fmt.Errorf("configuração %s da empresa %s: %w", key, companyID, err)
	}
	value, _ := setting.Value.(bool)
	return value, nil
}

// PeriodStart início do ciclo de cobrança que termina em currentPeriodEnd
func PeriodStart(currentPeriodEnd time.Time, billingCycle string) time.Time {
	if strings.ToLower(billingCycle) == "yearly" {
		return currentPeriodEnd.AddDate(-1, 0, 0)
	}
	// "monthly" e qualquer valor não reconhecido
	return currentPeriodEnd.AddDate(0, -1, 0)
}
