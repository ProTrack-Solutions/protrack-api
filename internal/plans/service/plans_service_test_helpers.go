package service

// NewServiceWithRepo é um construtor auxiliar para testes que permite injetar
// implementações de RepositoryInterface e PlanFeatureServiceInterface (como mocks).
func NewServiceWithRepo(repo RepositoryInterface, featureSvc PlanFeatureServiceInterface) *Service {
	return &Service{
		repo:                repo,
		plansFeatureService: featureSvc,
		pool:                nil,
	}
}
