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

func (r *Repository) WithTx(tx db.DBTX) *Repository {
	return &Repository{
		db: tx,
	}
}

func (r *Repository) queries() *db.Queries {
	return db.New(r.db)
}

func (r *Repository) CreateUsers(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	q := db.New(r.db)
	return q.CreateUser(ctx, arg)
}

func (r *Repository) DeleteUser(ctx context.Context, id pgtype.UUID) error {
	q := db.New(r.db)
	return q.DeleteUser(ctx, id)
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	q := db.New(r.db)
	return q.GetUserByEmail(ctx, email)
}

func (r *Repository) GetUserById(ctx context.Context, id pgtype.UUID) (db.User, error) {
	q := db.New(r.db)
	return q.GetUserByID(ctx, id)
}

func (r *Repository) ListUsers(ctx context.Context) ([]db.User, error) {
	q := db.New(r.db)
	return q.ListUsers(ctx)
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, arg db.UpdatePasswordHashParams) error {
	q := db.New(r.db)
	return q.UpdatePasswordHash(ctx, arg)
}

func (r *Repository) UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.User, error) {
	q := db.New(r.db)
	return q.UpdateUser(ctx, arg)
}

func (r *Repository) UpdateUserCompanyAndRole(ctx context.Context, arg db.UpdateUserCompanyAndRoleParams) error {
	q := db.New(r.db)
	return q.UpdateUserCompanyAndRole(ctx, arg)
}

func (r *Repository) UpdateLastLogin(ctx context.Context, id pgtype.UUID) error {
	return r.queries().UpdateLastLogin(ctx, id)
}
