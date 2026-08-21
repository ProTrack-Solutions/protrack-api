package domain

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
)

type RepositoryInterface interface {
	ListModules(ctx context.Context) ([]db.Module, error)
	GetModule(ctx context.Context, code string) (db.Module, error)
}

type ServiceInterface interface {
	ListModules(ctx context.Context) ([]ModuleResponse, error)
	GetModule(ctx context.Context, code string) (ModuleResponse, error)
}

type ModuleResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
