package repository

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
)

type Repository struct {
	db db.DBTX
}

func NewRepository(db db.DBTX) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) queries() *db.Queries {
	return db.New(r.db)
}

func (r *Repository) ListModules(ctx context.Context) ([]db.Module, error) {
	return r.queries().ListModules(ctx)
}

func (r *Repository) GetModule(ctx context.Context, code string) (db.Module, error) {
	return r.queries().GetModule(ctx, code)
}
