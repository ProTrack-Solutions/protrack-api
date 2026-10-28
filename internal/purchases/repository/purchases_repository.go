package repository

import (
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
)

// Repository expõe as consultas geradas pelo sqlc. O módulo tem muitas
// consultas usadas só aqui, então em vez de um repasse por método as consultas
// ficam promovidas pelo *db.Queries embutido.
type Repository struct {
	*db.Queries
}

func NewRepository(dbtx db.DBTX) *Repository {
	return &Repository{Queries: db.New(dbtx)}
}

func (r *Repository) WithTx(tx db.DBTX) *Repository {
	return NewRepository(tx)
}
