package repository

import (
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
)

// Repository expõe as consultas geradas pelo sqlc, promovidas pelo *db.Queries
// embutido (mesmo padrão de compras e devoluções).
type Repository struct {
	*db.Queries
}

func NewRepository(dbtx db.DBTX) *Repository {
	return &Repository{Queries: db.New(dbtx)}
}
