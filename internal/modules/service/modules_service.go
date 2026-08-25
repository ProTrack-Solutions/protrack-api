package service

import (
	"context"

	"github.com/ProTrack-Solutions/protrack-api/internal/modules/domain"
)

type Service struct {
	repo domain.RepositoryInterface
}

func NewService(repo domain.RepositoryInterface) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) ListModules(ctx context.Context) ([]domain.ModuleResponse, error) {
	modules, err := s.repo.ListModules(ctx)
	if err != nil {
		return []domain.ModuleResponse{}, err
	}

	var response []domain.ModuleResponse

	for _, module := range modules {
		response = append(response, domain.ModuleResponse{
			Code: module.Code,
			Name: module.Name,
		})
	}

	return response, nil
}

func (s *Service) GetModule(ctx context.Context, code string) (domain.ModuleResponse, error) {
	module, err := s.repo.GetModule(ctx, code)
	if err != nil {
		return domain.ModuleResponse{}, err
	}

	return domain.ModuleResponse{
		Code: module.Code,
		Name: module.Name,
	}, nil
}
