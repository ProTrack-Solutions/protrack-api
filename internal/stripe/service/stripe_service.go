package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/stripe/domain"

	subscriptionDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	subscriptionService "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/service"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/customer"
	"github.com/stripe/stripe-go/v86/paymentmethod"
	"github.com/stripe/stripe-go/v86/subscription"
)

type Service struct {
	secretKey           string
	subscriptionService *subscriptionService.Service
}

func NewService(cfg *config.Config, subscriptionService *subscriptionService.Service) *Service {
	return &Service{
		secretKey:           cfg.StripeSecretKey,
		subscriptionService: subscriptionService,
	}
}

func (s *Service) CreateSubscription(input domain.CreateSubscriptionInput) (*domain.CreateSubscriptionOutput, error) {
	stripe.Key = s.secretKey

	customerParams := &stripe.CustomerParams{
		Name:  stripe.String(input.Name),
		Email: stripe.String(input.Email),
	}
	cust, err := customer.New(customerParams)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente no Stripe: %w", err)
	}

	pmParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(cust.ID),
	}
	pm, err := paymentmethod.Attach(input.CardToken, pmParams)
	if err != nil {
		return nil, fmt.Errorf("erro ao vincular cartão ao cliente: %w", err)
	}

	customerUpdateParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}

	_, err = customer.Update(cust.ID, customerUpdateParams)
	if err != nil {
		return nil, fmt.Errorf("erro ao definir método padrão de pagamento: %w", err)
	}

	subParams := &stripe.SubscriptionParams{
		Customer: stripe.String(cust.ID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(input.PriceID),
			},
		},
	}

	subParams.AddExpand("latest_invoice.payment_intent")

	sub, err := subscription.New(subParams)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar assinatura no Stripe: %w", err)
	}

	return &domain.CreateSubscriptionOutput{
		CustomerID:     cust.ID,
		SubscriptionID: sub.ID,
		Status:         string(sub.Status), // "active", "incomplete", etc.
	}, nil
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

		if invoice.Customer.Subscriptions == nil {
			return nil
		}
		// Atualiza o banco: Pagamento Falhou
		subID := invoice.Customer.Subscriptions.Data[0].ID

		subscription, err := s.subscriptionService.GetSubscriptionByExternalSubscriptionId(ctx, subID)
		if err != nil {
			return err
		}

		if err := s.subscriptionService.UpdateSubscriptionStatus(ctx, subscription.ID, subscriptionDomain.UpdateSubscriptionStatusRequest{
			Status:           "pause",
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
