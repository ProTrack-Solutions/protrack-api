package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	discordDomain "github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_management/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_management/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	subPaymentMethodDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/domain"
	subPaymentMethod "github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/service"
	subscriptionService "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/service"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/customer"
	"github.com/stripe/stripe-go/v86/paymentmethod"
	"github.com/stripe/stripe-go/v86/setupintent"
	"github.com/stripe/stripe-go/v86/subscription"
)

type RepositoryInterface interface {
	GetSubscriptionDetailsByCompanyID(ctx context.Context, companyId pgtype.UUID) (db.GetSubscriptionDetailsByCompanyIDRow, error)
	WithTx(tx db.DBTX) *repository.Repository
}

type Service struct {
	repo                RepositoryInterface
	subPaymentMethod    *subPaymentMethod.Service
	subscriptionService *subscriptionService.Service
	loggerDiscord       *discord.DiscordLogger
}

func NewService(cfg *config.Config, subscriptionService *subscriptionService.Service, subPaymentMethod *subPaymentMethod.Service, loggerDiscord *discord.DiscordLogger, repo *repository.Repository) *Service {
	stripe.Key = cfg.StripeSecretKey
	return &Service{
		subscriptionService: subscriptionService,
		subPaymentMethod:    subPaymentMethod,
		loggerDiscord:       loggerDiscord,
		repo:                repo,
	}
}

func (s *Service) GetSubscriptionDetails(ctx context.Context, companyID uuid.UUID) (domain.SubscriptionDetailsResponse, error) {
	row, err := s.repo.GetSubscriptionDetailsByCompanyID(ctx, pgconv.ParseUUIDToPgType(companyID))
	if err != nil {
		return domain.SubscriptionDetailsResponse{}, fmt.Errorf("erro ao buscar dados da assinatura: %w", err)
	}

	var features []domain.PlanFeatureResponse
	if row.Features != nil {
		raw, err := json.Marshal(row.Features)
		if err != nil {
			return domain.SubscriptionDetailsResponse{}, fmt.Errorf("erro ao processar features do plano: %w", err)
		}

		if err := json.Unmarshal(raw, &features); err != nil {
			return domain.SubscriptionDetailsResponse{}, fmt.Errorf("erro ao processar features do plano: %w", err)
		}
	}

	var paymentMethod *domain.PaymentMethodDetails
	if row.PaymentMethodID.Valid {
		paymentMethod = &domain.PaymentMethodDetails{
			ID:           pgconv.PgUUIDToUUID(row.PaymentMethodID).String(),
			Type:         pgconv.ParsePgTextToString(row.PaymentMethodType),
			CardBrand:    pgconv.ParsePgTextToString(row.PaymentMethodCardBrand),
			CardLast4:    pgconv.ParsePgTextToString(row.PaymentMethodCardLast4),
			CardExpMonth: int32(pgconv.PgInt4ToInt(row.PaymentMethodCardExpMonth)),
			CardExpYear:  int32(pgconv.PgInt4ToInt(row.PaymentMethodCardExpYear)),
			IsDefault:    pgconv.PgBoolToBool(row.PaymentMethodIsDefault),
		}
	}

	return domain.SubscriptionDetailsResponse{
		SubscriptionID:         pgconv.PgUUIDToUUID(row.SubscriptionID).String(),
		CompanyID:              pgconv.PgUUIDToUUID(row.CompanyID).String(),
		Status:                 row.SubscriptionStatus,
		CurrentPeriodStart:     pgconv.PgTimestamptzToTime(row.CurrentPeriodStart),
		CurrentPeriodEnd:       pgconv.PgTimestamptzToTime(row.CurrentPeriodEnd),
		CanceledAt:             pgconv.PgTimestamptzToTime(row.CanceledAt),
		ExternalSubscriptionID: pgconv.ParsePgTextToString(row.ExternalSubscriptionID),
		CreatedAt:              pgconv.PgTimestamptzToTime(row.SubscriptionCreatedAt),
		UpdatedAt:              pgconv.PgTimestamptzToTime(row.SubscriptionUpdatedAt),
		Plan: domain.PlanDetailsResponse{
			ID:              pgconv.PgUUIDToUUID(row.PlanID).String(),
			ExternalID:      row.PlanExternalID,
			ExternalPriceID: row.PlanExternalPriceID,
			Name:            row.PlanName,
			Description:     pgconv.ParsePgTextToString(row.PlanDescription),
			PriceCents:      row.PlanPriceCents,
			Currency:        pgconv.ParsePgTextToString(row.PlanCurrency),
			BillingCycle:    row.PlanBillingCycle,
			Active:          pgconv.PgBoolToBool(row.PlanActive),
			Highlight:       row.PlanHighlight,
			Icon:            row.PlanIcon,
		},
		PaymentMethod: paymentMethod,
		Features:      features,
	}, nil
}

func (s *Service) UpdateDefaultPaymentMethod(ctx context.Context, companyID uuid.UUID, subPaymentMethod uuid.UUID, req domain.UpdateDefaultPaymentMethodRequest) error {
	if req.IdempotencyKey == "" {
		return fmt.Errorf("idempotency key é obrigatória")
	}

	sub, err := s.subscriptionService.GetSubscriptionByCompanyID(ctx, companyID)
	if err != nil {
		return err
	}

	paymentMethod, err := s.subPaymentMethod.GetSubscriptionPaymentMethodById(ctx, subPaymentMethod)
	if err != nil {
		return err
	}

	if paymentMethod.CompanyID != companyID {
		return fmt.Errorf("payment method não pertence à empresa informada")
	}

	subStripe, err := subscription.Get(sub.ExternalSubscriptionID, nil)
	if err != nil {
		return err
	}

	attachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(subStripe.Customer.ID),
	}

	_, err = paymentmethod.Attach(paymentMethod.GatewayPaymentMethodID, attachParams)
	if err != nil {
		var stripeErr *stripe.Error

		if !errors.As(err, &stripeErr) || stripeErr.Code != stripe.ErrorCodeResourceMissing {
			if stripeErr == nil || stripeErr.Type != stripe.ErrorTypeInvalidRequest {
				return fmt.Errorf("erro ao anexar payment method: %w", err)
			}
		}
	}
	subParams := &stripe.SubscriptionParams{
		DefaultPaymentMethod: stripe.String(paymentMethod.GatewayPaymentMethodID),
	}
	subParams.SetIdempotencyKey(req.IdempotencyKey + "-sub")
	_, err = subscription.Update(sub.ExternalSubscriptionID, subParams)
	if err != nil {
		return fmt.Errorf("erro ao atualizar payment method da assinatura: %w", err)
	}

	custParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethod.GatewayPaymentMethodID),
		},
	}

	custParams.SetIdempotencyKey(req.IdempotencyKey + "-customer")
	if _, err := customer.Update(subStripe.Customer.ID, custParams); err != nil {
		return fmt.Errorf("erro ao atualizar payment method padrão do customer: %w", err)
	}

	return nil
}

func (s *Service) CancelSubscription(ctx context.Context, companyID uuid.UUID, req domain.CancelSubscriptionRequest) error {
	if req.IdempotencyKey == "" {
		return errors.New("idempotency key é obrigatória")
	}

	sub, err := s.subscriptionService.GetSubscriptionByCompanyID(ctx, companyID)
	if err != nil {
		return err
	}

	if req.CancelAtPeriodEnd {
		subParams := &stripe.SubscriptionParams{
			CancelAtPeriodEnd: stripe.Bool(true),
		}
		subParams.SetIdempotencyKey(req.IdempotencyKey + "-cancel-at-period-end")

		_, err = subscription.Update(sub.ExternalSubscriptionID, subParams)
		if err != nil {
			return fmt.Errorf("erro ao agendar cancelamento da assinatura: %w", err)
		}

		return nil
	}

	cancelParams := &stripe.SubscriptionCancelParams{}
	cancelParams.SetIdempotencyKey(req.IdempotencyKey + "-cancel")

	_, err = subscription.Cancel(sub.ExternalSubscriptionID, cancelParams)
	if err != nil {
		return fmt.Errorf("erro ao cancelar assinatura: %w", err)
	}

	return s.subscriptionService.CancelSubscription(ctx, sub.ID)
}

func (s *Service) AddPaymentMethod(ctx context.Context, companyID uuid.UUID, userID uuid.UUID, req domain.AddPaymentMethodRequest) error {
	if req.IdempotencyKey == "" {
		return errors.New("idempotency key é obrigatória")
	}

	sub, err := s.subscriptionService.GetSubscriptionByCompanyID(ctx, companyID)
	if err != nil {
		return err
	}

	subStripe, err := subscription.Get(sub.ExternalSubscriptionID, nil)
	if err != nil {
		return fmt.Errorf("erro ao buscar assinatura: %w", err)
	}

	if subStripe.Customer == nil {
		return errors.New("assinatura sem customer associado")
	}

	siParams := &stripe.SetupIntentParams{
		Customer:      stripe.String(subStripe.Customer.ID),
		PaymentMethod: stripe.String(req.StripePaymentMethodID),
		Confirm:       stripe.Bool(true),
		Usage:         stripe.String("off_session"),
	}
	siParams.SetIdempotencyKey(req.IdempotencyKey + "-setup-intent")

	si, err := setupintent.New(siParams)
	if err != nil {
		return fmt.Errorf("erro ao validar payment method: %w", err)
	}

	if si.Status != stripe.SetupIntentStatusSucceeded {
		return fmt.Errorf("payment method não pôde ser confirmado (status: %s)", si.Status)
	}

	pm, err := paymentmethod.Get(req.StripePaymentMethodID, nil)
	if err != nil {
		return fmt.Errorf("erro ao buscar payment method confirmado: %w", err)
	}

	var brand, last4 string
	var expMonth, expYear int32
	if pm.Card != nil {
		brand = string(pm.Card.Brand)
		last4 = pm.Card.Last4
		expMonth = int32(pm.Card.ExpMonth)
		expYear = int32(pm.Card.ExpYear)
	}

	if err := s.subPaymentMethod.CreateSubscriptionPaymentMethod(ctx, companyID, userID, subPaymentMethodDomain.CreateSubscriptionPaymentMethodRequest{
		GatewayPaymentMethodId: pm.ID,
		Type:                   string(pm.Type),
		CardBrand:              brand,
		CardLastFour:           last4,
		CardExpMonth:           expMonth,
		CardExpYear:            expYear,
		IsDefault:              req.SetAsDefault,
	}); err != nil {
		if _, detachErr := paymentmethod.Detach(pm.ID, nil); detachErr != nil {
			s.loggerDiscord.Send(discordDomain.LevelError, "falha ao reverter attach após erro de save", detachErr.Error())
		}
		return fmt.Errorf("erro ao salvar payment method: %w", err)
	}

	if !req.SetAsDefault {
		return nil
	}

	// --- Opção 5: captura estado anterior para permitir rollback em caso de falha parcial ---
	var previousSubDefaultPM string
	if subStripe.DefaultPaymentMethod != nil {
		previousSubDefaultPM = subStripe.DefaultPaymentMethod.ID
	}

	var previousCustomerDefaultPM string
	if subStripe.Customer.InvoiceSettings != nil && subStripe.Customer.InvoiceSettings.DefaultPaymentMethod != nil {
		previousCustomerDefaultPM = subStripe.Customer.InvoiceSettings.DefaultPaymentMethod.ID
	}

	subParams := &stripe.SubscriptionParams{
		DefaultPaymentMethod: stripe.String(pm.ID),
	}
	subParams.SetIdempotencyKey(req.IdempotencyKey + "-sub-default")

	if _, err := subscription.Update(sub.ExternalSubscriptionID, subParams); err != nil {
		return fmt.Errorf("payment method salvo, mas erro ao definir como padrão da assinatura: %w", err)
	}

	custParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	custParams.SetIdempotencyKey(req.IdempotencyKey + "-customer-default")

	_, custErr := customer.Update(subStripe.Customer.ID, custParams)
	if custErr == nil {
		return nil
	}

	// Retry único antes de assumir falha definitiva
	retryParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	retryParams.SetIdempotencyKey(req.IdempotencyKey + "-customer-default-retry")

	if _, retryErr := customer.Update(subStripe.Customer.ID, retryParams); retryErr == nil {
		return nil
	}

	// Falhou de novo: reverte a assinatura para o default anterior, para não deixar
	// subscription e customer apontando para payment methods diferentes.
	rollbackParams := &stripe.SubscriptionParams{
		DefaultPaymentMethod: stripe.String(previousSubDefaultPM),
	}
	rollbackParams.SetIdempotencyKey(req.IdempotencyKey + "-sub-default-rollback")

	if previousSubDefaultPM != "" {
		if _, rollbackErr := subscription.Update(sub.ExternalSubscriptionID, rollbackParams); rollbackErr != nil {
			s.loggerDiscord.Send(discordDomain.LevelError,
				"falha crítica: subscription e customer com default divergentes, rollback falhou",
				fmt.Sprintf("company=%s pm=%s subErr=%v customerErr=%v previousCustomerDefault=%s",
					companyID, pm.ID, rollbackErr, custErr, previousCustomerDefaultPM))
			return fmt.Errorf("payment method salvo, mas estado inconsistente entre subscription e customer: %w", custErr)
		}
	}

	s.loggerDiscord.Send(discordDomain.LevelWarning,
		"payment method salvo, default revertido após falha ao atualizar customer",
		fmt.Sprintf("company=%s pm=%s customerErr=%v", companyID, pm.ID, custErr))

	return fmt.Errorf("payment method salvo, mas erro ao definir como padrão do cliente: %w", custErr)
}
