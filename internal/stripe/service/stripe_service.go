package service

import (
	"fmt"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/stripe/domain"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/customer"
	"github.com/stripe/stripe-go/v78/paymentmethod"
	"github.com/stripe/stripe-go/v78/subscription"
)

type Service struct {
	secretKey string
}

func NewService(cfg *config.Config) *Service {
	return &Service{secretKey: cfg.StripeSecretKey}
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
