package service

import "github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"

// NewServiceWithDeps é um construtor alternativo para testes que permite injetar
// qualquer implementação das interfaces de dependência (incluindo mocks).
// Não deve ser utilizado em código de produção.
func NewServiceWithDeps(
	repo domain.RepositoryInterface,
	metaClient domain.MetaClientInterface,
	subscriptionService domain.SubscriptionsServiceInterface,
	plansService domain.PlansServiceInterface,
	billingClient domain.BillingClientInterface,
	companiesService domain.CompaniesServiceInterface,
	phoneNumberID string,
	metaAccessToken string,
	metaSecretKey string,
	stripeWhatsappOveragePriceID string,
) *Service {
	return &Service{
		repo:                         repo,
		metaClient:                   metaClient,
		subscriptionService:          subscriptionService,
		plansService:                 plansService,
		billingClient:                billingClient,
		companiesService:             companiesService,
		PhoneNumberID:                phoneNumberID,
		MetaAccessToken:              metaAccessToken,
		MetaSecretKey:                metaSecretKey,
		StripeWhatsappOveragePriceID: stripeWhatsappOveragePriceID,
	}
}
