package repository

import (
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
)

// Repository expõe as queries do sqlc (fiscal.sql e as que a emissão usa).
type Repository struct {
	*db.Queries
}

func NewRepository(conn db.DBTX) *Repository {
	return &Repository{Queries: db.New(conn)}
}

// WithTx devolve um repositório preso à transação.
func (r *Repository) WithTx(tx db.DBTX) *Repository {
	return &Repository{Queries: db.New(tx)}
}
