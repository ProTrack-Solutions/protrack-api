package service

// NewServiceWithRepo é um construtor alternativo para testes que permite injetar
// qualquer implementação de RepositoryInterface (incluindo mocks).
// Não deve ser utilizado em código de produção.
func NewServiceWithRepo(repo RepositoryInterface) *Service {
	return &Service{
		repo: repo,
		pool: nil,
	}
}

// NewServiceWithDeps é um construtor alternativo para testes que permite injetar
// implementações de RepositoryInterface, PlansServiceInterface e SubscriptionsServiceInterface
// (incluindo mocks). Não deve ser utilizado em código de produção.
func NewServiceWithDeps(repo RepositoryInterface, plansSvc PlansServiceInterface, subscriptionsSvc SubscriptionsServiceInterface) *Service {
	return &Service{
		repo:                 repo,
		plansService:         plansSvc,
		subscriptionsService: subscriptionsSvc,
		pool:                 nil,
	}
}
