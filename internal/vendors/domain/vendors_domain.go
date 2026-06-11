package domain

import (
	"time"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/google/uuid"
)

type CreateVendorsRequest struct {
	CompanyID    uuid.UUID `json:"company_id"`
	Name         string    `json:"name"`
	TaxID        string    `json:"tax_id"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	PostalCode   string    `json:"postal_code"`
	AddressLine1 string    `json:"address_line_1"`
	AddressLine2 string    `json:"address_line_2"`
	Number       string    `json:"number"`
	Neighborhood string    `json:"neighborhood"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	Country      string    `json:"country"`
}

type GetVendorsByIdRequest struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`
}

type ToggleVendorsActiveParams struct {
	IsActive bool `json:"is_active"`
}

type VendorResponse struct {
	ID           uuid.UUID `json:"id"`
	CompanyID    uuid.UUID `json:"company_id"`
	Name         string    `json:"name"`
	TaxID        string    `json:"tax_id"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	PostalCode   string    `json:"postal_code"`
	AddressLine1 string    `json:"address_line_1"`
	AddressLine2 string    `json:"address_line_2"`
	Number       string    `json:"number"`
	Neighborhood string   `json:"neighborhood"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	Country      string    `json:"country"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UpdateVendorsRequest struct {
	Name         string `json:"name"`
	TaxID        string `json:"tax_id"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	PostalCode   string `json:"postal_code"`
	AddressLine1 string `json:"address_line_1"`
	AddressLine2 string `json:"address_line_2"`
	Number       string `json:"number"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	State        string `json:"state"`
	Country      string `json:"country"`
}

func ApplyUpdateVendorsParams(req UpdateVendorsRequest, arg *db.UpdateVendorsParams) {
	if req.Name != "" {
		arg.Name = req.Name
	}

	if req.TaxID != "" {
		arg.TaxID = pgconv.ParseStringToPgText(req.TaxID)
	}

	if req.Email != "" {
		arg.Email = pgconv.ParseStringToPgText(req.Email)
	}

	if req.Phone != "" {
		arg.Phone = pgconv.ParseStringToPgText(req.Phone)
	}

	if req.PostalCode != "" {
		arg.PostalCode = pgconv.ParseStringToPgText(req.PostalCode)
	}

	if req.AddressLine1 != "" {
		arg.AddressLine1 = pgconv.ParseStringToPgText(req.AddressLine1)
	}

	if req.AddressLine2 != "" {
		arg.AddressLine2 = pgconv.ParseStringToPgText(req.AddressLine2)
	}

	if req.Number != "" {
		arg.Number = pgconv.ParseStringToPgText(req.Number)
	}

	if req.Neighborhood != "" {
		arg.Neighborhood = pgconv.ParseStringToPgText(req.Neighborhood)
	}

	if req.City != "" {
		arg.City = pgconv.ParseStringToPgText(req.City)
	}

	if req.State != "" {
		arg.State = pgconv.ParseStringToPgText(req.State)
	}

	if req.Country != "" {
		arg.Country = pgconv.ParseStringToPgText(req.Country)
	}
}
