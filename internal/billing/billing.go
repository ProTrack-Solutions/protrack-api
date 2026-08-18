package billing

import (
	"context"
	"strconv"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/stripe/stripe-go/v86"
)

type Client struct {
	stripeClient *stripe.Client
	eventName    string
}

func NewClient(stripeClient *stripe.Client, cfg config.Config) *Client {
	return &Client{
		stripeClient: stripeClient,
		eventName:    cfg.StripeWhatsappMeterEventName,
	}
}

func (c *Client) ReportUsage(ctx context.Context, customerID string, quantity int, idempotencyKey string) error {
	quantityStr := strconv.Itoa(quantity)
	billing := &stripe.BillingMeterEventCreateParams{
		EventName: stripe.String(c.eventName),
		Payload: map[string]string{
			"stripe_customer_id": customerID,
			"value":              quantityStr,
		},
		Identifier: stripe.String(idempotencyKey),
	}

	_, err := c.stripeClient.V1BillingMeterEvents.Create(ctx, billing)
	return err
}

func (c *Client) AttachOverageItem(ctx context.Context, subscriptionID, priceID string) error {
	params := &stripe.SubscriptionItemCreateParams{
		Subscription: stripe.String(subscriptionID),
		Price:        stripe.String(priceID),
	}
	params.SetIdempotencyKey("attach-overage-" + subscriptionID + "-" + priceID)

	_, err := c.stripeClient.V1SubscriptionItems.Create(ctx, params)
	return err
}
