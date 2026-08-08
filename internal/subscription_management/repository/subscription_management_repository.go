package repository

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *Repository) WithTx(tx db.DBTX) *Repository {
	return &Repository{
		db: tx,
	}
}

func (r *Repository) GetSubscriptionDetailsByCompanyID(ctx context.Context, companyId pgtype.UUID) (db.GetSubscriptionDetailsByCompanyIDRow, error) {
	return r.queries().GetSubscriptionDetailsByCompanyID(ctx, companyId)
}
