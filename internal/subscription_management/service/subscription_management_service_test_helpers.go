package service

import metaWhatsappDomain "github.com/ProTrack-Solutions/protrack-api/internal/meta_whatsapp/domain"

// NewServiceWithDeps é um construtor auxiliar para testes que permite injetar
// diretamente as dependências do Service (como mocks), sem passar pelas
// exigências de NewService (ex.: repositório concreto, chave da Stripe).
func NewServiceWithDeps(repo RepositoryInterface, subscriptionService SubscriptionsServiceInterface, planService PlansServiceInterface, metaWhatsappService metaWhatsappDomain.ServiceInterface) *Service {
	return &Service{
		repo:                repo,
		subscriptionService: subscriptionService,
		planService:         planService,
		metaWhatsappService: metaWhatsappService,
	}
}
