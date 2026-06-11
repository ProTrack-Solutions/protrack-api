package service

import (
	"context"
	"errors"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/validate"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/vendors/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/vendors/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateVendors(ctx context.Context, arg db.CreateVendorsParams) error
	GetVendorsById(ctx context.Context, arg db.GetVendorsByIdParams) (db.Vendor, error)
	ListVendors(ctx context.Context, companyId pgtype.UUID) ([]db.Vendor, error)
	ListVendorsIsActive(ctx context.Context, companyId pgtype.UUID) ([]db.Vendor, error)
	ToggleVendorsActive(ctx context.Context, arg db.ToggleVendorsActiveParams) error
	UpdateVendors(ctx context.Context, arg db.UpdateVendorsParams) error
}

type Service struct {
	repo RepositoryInterface
	pool *pgxpool.Pool
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool) *Service {
	return &Service{
		repo: repo,
		pool: pool,
	}
}

func (s *Service) CreateVendors(ctx context.Context, req domain.CreateVendorsRequest) error {
	_, err := validate.ValidateDocument(req.TaxID)
	if err != nil {
		return err
	}

	is := validate.IsValidEmail(req.Email)
	if !is {
		return errors.New("email invalid")
	}

	if err := s.repo.CreateVendors(ctx, db.CreateVendorsParams{
		CompanyID:    pgconv.ParseUUIDToPgType(req.CompanyID),
		Name:         req.Name,
		TaxID:        pgconv.ParseStringToPgText(req.TaxID),
		Email:        pgconv.ParseStringToPgText(req.Email),
		Phone:        pgconv.ParseStringToPgText(req.Phone),
		PostalCode:   pgconv.ParseStringToPgText(req.PostalCode),
		AddressLine1: pgconv.ParseStringToPgText(req.AddressLine1),
		AddressLine2: pgconv.ParseStringToPgText(req.AddressLine2),
		Number:       pgconv.ParseStringToPgText(req.Number),
		Neighborhood: pgconv.ParseStringToPgText(req.Neighborhood),
		City:         pgconv.ParseStringToPgText(req.City),
		State:        pgconv.ParseStringToPgText(req.State),
		Country:      pgconv.ParseStringToPgText(req.Country),
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetVendorsById(ctx context.Context, req domain.GetVendorsByIdRequest) (domain.VendorResponse, error) {
	vendor, err := s.repo.GetVendorsById(ctx, db.GetVendorsByIdParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
	})
	if err != nil {
		return domain.VendorResponse{}, err
	}

	return domain.VendorResponse{
		ID:           pgconv.PgUUIDToUUID(vendor.ID),
		CompanyID:    pgconv.PgUUIDToUUID(vendor.CompanyID),
		Name:         vendor.Name,
		TaxID:        pgconv.ParsePgTextToString(vendor.TaxID),
		Email:        pgconv.ParsePgTextToString(vendor.Email),
		Phone:        pgconv.ParsePgTextToString(vendor.Phone),
		PostalCode:   pgconv.ParsePgTextToString(vendor.PostalCode),
		AddressLine1: pgconv.ParsePgTextToString(vendor.AddressLine1),
		AddressLine2: pgconv.ParsePgTextToString(vendor.AddressLine2),
		Number:       pgconv.ParsePgTextToString(vendor.Number),
		Neighborhood: pgconv.ParsePgTextToString(vendor.Neighborhood),
		City:         pgconv.ParsePgTextToString(vendor.City),
		State:        pgconv.ParsePgTextToString(vendor.State),
		Country:      pgconv.ParsePgTextToString(vendor.Country),
		IsActive:     vendor.IsActive,
		CreatedAt:    pgconv.PgTimestamptzToTime(vendor.CreatedAt),
		UpdatedAt:    pgconv.PgTimestamptzToTime(vendor.UpdatedAt),
	}, nil
}

func (s *Service) ListVendors(ctx context.Context, companyId uuid.UUID) ([]domain.VendorResponse, error) {
	vendors, err := s.repo.ListVendors(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.VendorResponse{}, err
	}

	var response []domain.VendorResponse

	for _, vendor := range vendors {
		response = append(response, domain.VendorResponse{
			ID:           pgconv.PgUUIDToUUID(vendor.ID),
			CompanyID:    pgconv.PgUUIDToUUID(vendor.CompanyID),
			Name:         vendor.Name,
			TaxID:        pgconv.ParsePgTextToString(vendor.TaxID),
			Email:        pgconv.ParsePgTextToString(vendor.Email),
			Phone:        pgconv.ParsePgTextToString(vendor.Phone),
			PostalCode:   pgconv.ParsePgTextToString(vendor.PostalCode),
			AddressLine1: pgconv.ParsePgTextToString(vendor.AddressLine1),
			AddressLine2: pgconv.ParsePgTextToString(vendor.AddressLine2),
			Number:       pgconv.ParsePgTextToString(vendor.Number),
			Neighborhood: pgconv.ParsePgTextToString(vendor.Neighborhood),
			City:         pgconv.ParsePgTextToString(vendor.City),
			State:        pgconv.ParsePgTextToString(vendor.State),
			Country:      pgconv.ParsePgTextToString(vendor.Country),
			IsActive:     vendor.IsActive,
			CreatedAt:    pgconv.PgTimestamptzToTime(vendor.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(vendor.UpdatedAt),
		})
	}
	return response, nil
}

func (s *Service) ListVendorsIsActive(ctx context.Context, companyId uuid.UUID) ([]domain.VendorResponse, error) {
	vendors, err := s.repo.ListVendorsIsActive(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.VendorResponse{}, err
	}

	var response []domain.VendorResponse

	for _, vendor := range vendors {
		response = append(response, domain.VendorResponse{
			ID:           pgconv.PgUUIDToUUID(vendor.ID),
			CompanyID:    pgconv.PgUUIDToUUID(vendor.CompanyID),
			Name:         vendor.Name,
			TaxID:        pgconv.ParsePgTextToString(vendor.TaxID),
			Email:        pgconv.ParsePgTextToString(vendor.Email),
			Phone:        pgconv.ParsePgTextToString(vendor.Phone),
			PostalCode:   pgconv.ParsePgTextToString(vendor.PostalCode),
			AddressLine1: pgconv.ParsePgTextToString(vendor.AddressLine1),
			AddressLine2: pgconv.ParsePgTextToString(vendor.AddressLine2),
			Number:       pgconv.ParsePgTextToString(vendor.Number),
			Neighborhood: pgconv.ParsePgTextToString(vendor.Neighborhood),
			City:         pgconv.ParsePgTextToString(vendor.City),
			State:        pgconv.ParsePgTextToString(vendor.State),
			Country:      pgconv.ParsePgTextToString(vendor.Country),
			IsActive:     vendor.IsActive,
			CreatedAt:    pgconv.PgTimestamptzToTime(vendor.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(vendor.UpdatedAt),
		})
	}
	return response, nil
}

func (s *Service) ToggleVendorsActive(ctx context.Context, id uuid.UUID, req domain.ToggleVendorsActiveParams) error {
	return s.repo.ToggleVendorsActive(ctx, db.ToggleVendorsActiveParams{
		ID:       pgconv.ParseUUIDToPgType(id),
		IsActive: req.IsActive,
	})
}

func (s *Service) UpdateVendors(ctx context.Context, reqById domain.GetVendorsByIdRequest, req domain.UpdateVendorsRequest) error {
	vendor, err := s.repo.GetVendorsById(ctx, db.GetVendorsByIdParams{
		ID:        pgconv.ParseUUIDToPgType(reqById.ID),
		CompanyID: pgconv.ParseUUIDToPgType(reqById.CompanyID),
	})
	if err != nil {
		return err
	}

	arg := db.UpdateVendorsParams{
		ID:           vendor.ID,
		CompanyID:    vendor.CompanyID,
		Name:         vendor.Name,
		TaxID:        vendor.TaxID,
		Email:        vendor.Email,
		Phone:        vendor.Phone,
		PostalCode:   vendor.PostalCode,
		AddressLine1: vendor.AddressLine1,
		AddressLine2: vendor.AddressLine2,
		Number:       vendor.Number,
		Neighborhood: vendor.Neighborhood,
		City:         vendor.City,
		State:        vendor.State,
		Country:      vendor.Country,
	}

	domain.ApplyUpdateVendorsParams(req, &arg)

	if err := s.repo.UpdateVendors(ctx, arg); err != nil {
		return err
	}

	return nil
}
