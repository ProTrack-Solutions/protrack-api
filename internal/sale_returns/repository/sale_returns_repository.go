package repository

import (
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
)

// Repository expõe as consultas geradas pelo sqlc, promovidas pelo *db.Queries
// embutido (mesmo padrão do módulo de compras).
type Repository struct {
	*db.Queries
}

func NewRepository(dbtx db.DBTX) *Repository {
	return &Repository{Queries: db.New(dbtx)}
}

func (r *Repository) WithTx(tx db.DBTX) *Repository {
	return NewRepository(tx)
}
