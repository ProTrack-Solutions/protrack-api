package repository

import (
	"context"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *Repository) CreateCompanyCertificate(ctx context.Context, arg db.CreateCompanyCertificateParams) (db.CompanyCertificate, error) {
	return r.queries().CreateCompanyCertificate(ctx, arg)
}

func (r *Repository) GetCompanyCertificateByCompanyID(ctx context.Context, companyID pgtype.UUID) (db.CompanyCertificate, error) {
	return r.queries().GetCompanyCertificateByCompanyID(ctx, companyID)
}

func (r *Repository) UpdateCompanyCertificate(ctx context.Context, arg db.UpdateCompanyCertificateParams) (db.CompanyCertificate, error) {
	return r.queries().UpdateCompanyCertificate(ctx, arg)
}

func (r *Repository) DeleteCompanyCertificate(ctx context.Context, arg db.DeleteCompanyCertificateParams) error {
	return r.queries().DeleteCompanyCertificate(ctx, arg)
}
