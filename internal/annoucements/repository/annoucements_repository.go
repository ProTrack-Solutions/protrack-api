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

func (r *Repository) CreateAnnoucements(ctx context.Context, arg db.CreateAnnouncementsParams) error {
	return r.queries().CreateAnnouncements(ctx, arg)
}

func (r *Repository) ListAnnoucements(ctx context.Context, arg db.ListAnnoucementsParams) ([]db.ListAnnoucementsRow, error) {
	return r.queries().ListAnnoucements(ctx, arg)
}

func (r *Repository) DeleteAnnoucements(ctx context.Context, arg db.DeleteAnnoucementsParams) error {
	return r.queries().DeleteAnnoucements(ctx, arg)
}

func (r *Repository) CountAnnoucementsByCompany(ctx context.Context, companyId pgtype.UUID) (int64, error) {
	return r.queries().CountAnnoucementsByCompany(ctx, companyId)
}
