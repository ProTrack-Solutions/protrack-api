package service

import "time"

// NewServiceWithRepo é um construtor alternativo para testes que aceita
// qualquer RepositoryInterface e um relógio fixo.
// Não deve ser utilizado em código de produção.
func NewServiceWithRepo(repo RepositoryInterface, now func() time.Time) *Service {
	return &Service{
		repo: repo,
		now:  now,
	}
}
