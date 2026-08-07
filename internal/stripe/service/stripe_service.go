package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/stripe/domain"
	"github.com/google/uuid"

	subPaymentMethodDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/domain"
	subPaymentMethod "github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/service"
	subscriptionDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	subscriptionService "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/service"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/customer"
	"github.com/stripe/stripe-go/v86/paymentmethod"
	"github.com/stripe/stripe-go/v86/subscription"
)

type Service struct {
	secretKey           string
	subPaymentMethod    *subPaymentMethod.Service
	subscriptionService *subscriptionService.Service
}

func NewService(cfg *config.Config, subscriptionService *subscriptionService.Service, subPaymentMethod *subPaymentMethod.Service) *Service {
	stripe.Key = cfg.StripeSecretKey
	return &Service{
		secretKey:           cfg.StripeSecretKey,
		subscriptionService: subscriptionService,
		subPaymentMethod:    subPaymentMethod,
	}
}

func (s *Service) CreateSubscription(input domain.CreateSubscriptionInput) (*domain.CreateSubscriptionOutput, error) {
	customerParams := &stripe.CustomerParams{
		Name:  stripe.String(input.Name),
		Email: stripe.String(input.Email),
	}

	customerParams.SetIdempotencyKey(input.IdempotencyKey + "-customer")
	cust, err := customer.New(customerParams)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente no Stripe: %w", err)
	}

	attachParams := &stripe.PaymentMethodAttachParams{Customer: stripe.String(cust.ID)}
	_, err = paymentmethod.Attach(input.CardToken, attachParams)
	if err != nil {
		return nil, fmt.Errorf("erro ao vincular cartão ao cliente: %w", err)
	}

	customerUpdateParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(input.CardToken),
		},
	}

	_, err = customer.Update(cust.ID, customerUpdateParams)
	if err != nil {
		customer.Del(cust.ID, nil)
		return nil, fmt.Errorf("erro ao definir método padrão de pagamento: %w", err)
	}

	subParams := &stripe.SubscriptionParams{
		Customer: stripe.String(cust.ID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(input.PriceID),
			},
		},
		PaymentBehavior: stripe.String("default_incomplete"),
	}

	subParams.AddExpand("latest_invoice.confirmation_secret")

	subParams.SetIdempotencyKey(input.IdempotencyKey + "-subscription")
	sub, err := subscription.New(subParams)
	if err != nil {
		customer.Del(cust.ID, nil)
		return nil, fmt.Errorf("erro ao criar assinatura no Stripe: %w", err)
	}

	output := &domain.CreateSubscriptionOutput{
		CustomerID:     cust.ID,
		SubscriptionID: sub.ID,
		Status:         string(sub.Status),
	}

	if sub.LatestInvoice != nil && sub.LatestInvoice.ConfirmationSecret != nil {
		output.ClientSecret = sub.LatestInvoice.ConfirmationSecret.ClientSecret
	}

	return output, nil
}

func (s *Service) SyncSubscriptionWebhook(ctx context.Context, event stripe.Event) error {
	switch string(event.Type) {

	case "invoice.payment_succeeded":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return fmt.Errorf("erro ao deserializar invoice.payment_succeeded: %w", err)
		}

		if invoice.Parent == nil || invoice.Parent.Type != stripe.InvoiceParentTypeSubscriptionDetails ||
			invoice.Parent.SubscriptionDetails == nil || invoice.Parent.SubscriptionDetails.Subscription == nil {
			return nil
		}

		subID := invoice.Parent.SubscriptionDetails.Subscription.ID

		var periodEnd time.Time
		if len(invoice.Lines.Data) > 0 && invoice.Lines.Data[0].Period != nil {
			periodEnd = time.Unix(invoice.Lines.Data[0].Period.End, 0)
		} else {
			periodEnd = time.Unix(invoice.PeriodEnd, 0)
		}

		subscription, err := s.subscriptionService.GetSubscriptionByExternalSubscriptionId(ctx, subID)
		if err != nil {
			return err
		}

		if err := s.subscriptionService.UpdateSubscriptionStatus(ctx, subscription.ID, subscriptionDomain.UpdateSubscriptionStatusRequest{
			Status:           "active",
			CurrentPeriodEnd: periodEnd,
		}); err != nil {
			return err
		}

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return fmt.Errorf("erro ao deserializar invoice.payment_failed: %w", err)
		}

		if invoice.Parent == nil || invoice.Parent.Type != stripe.InvoiceParentTypeSubscriptionDetails ||
			invoice.Parent.SubscriptionDetails == nil || invoice.Parent.SubscriptionDetails.Subscription == nil {
			return nil
		}
		subID := invoice.Parent.SubscriptionDetails.Subscription.ID

		subscription, err := s.subscriptionService.GetSubscriptionByExternalSubscriptionId(ctx, subID)
		if err != nil {
			return err
		}

		if err := s.subscriptionService.UpdateSubscriptionStatus(ctx, subscription.ID, subscriptionDomain.UpdateSubscriptionStatusRequest{
			Status:           "paused",
			CurrentPeriodEnd: subscription.CurrentPeriodEnd,
		}); err != nil {
			return err
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("erro ao deserializar customer.subscription.deleted: %w", err)
		}
		// Atualiza o banco: Assinatura Cancelada
		subID := sub.ID

		subscription, err := s.subscriptionService.GetSubscriptionByExternalSubscriptionId(ctx, subID)
		if err != nil {
			return err
		}

		if err := s.subscriptionService.CancelSubscription(ctx, subscription.ID); err != nil {
			return err
		}

	}

	return nil
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

	attachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(subStripe.Customer.ID),
	}
	attachParams.SetIdempotencyKey(req.IdempotencyKey + "-attach")

	pm, err := paymentmethod.Attach(req.StripePaymentMethodID, attachParams)
	if err != nil {
		return fmt.Errorf("erro ao anexar payment method: %w", err)
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
		return fmt.Errorf("erro ao salvar payment method: %w", err)
	}

	if !req.SetAsDefault {
		return nil
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

	if _, err := customer.Update(subStripe.Customer.ID, custParams); err != nil {
		return fmt.Errorf("payment method salvo, mas erro ao definir como padrão do cliente: %w", err)
	}

	return nil
}
