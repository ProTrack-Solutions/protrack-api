package client

import (
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/stripe/stripe-go/v86"
)

type StripeClient struct {
	client *stripe.Client
	cfg    *config.Config
}

func NewStripeClient(cfg *config.Config) *stripe.Client {
	client := stripe.NewClient(cfg.StripeSecretKey)
	return client
}
