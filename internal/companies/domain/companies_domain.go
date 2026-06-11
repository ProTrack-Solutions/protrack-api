package domain

import (
	"time"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/google/uuid"
)

type Company struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	TradeName           string    `json:"trade_name"`
	Document            string    `json:"document"`
	DocumentType        string    `json:"document_type"`
	Email               string    `json:"email"`
	Phone               string    `json:"phone"`
	Website             string    `json:"website"`
	AddressStreet       string    `json:"address_street"`
	AddressNumber       string    `json:"address_number"`
	AddressComplement   string    `json:"address_complement"`
	AddressNeighborhood string    `json:"address_neighborhood"`
	AddressCity         string    `json:"address_city"`
	AddressState        string    `json:"address_state"`
	AddressZipcode      string    `json:"address_zipcode"`
	AddressCountry      string    `json:"address_country"`
	Status              any       `json:"status"`
	Timezone            string    `json:"timezone"`
	CreatedBy           uuid.UUID `json:"created_by"`
	UpdatedBy           uuid.UUID `json:"updated_by"`
	DeletedBy           uuid.UUID `json:"deleted_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	DeletedAt           time.Time `json:"deleted_at"`
}

type CreateCompanyParams struct {
	Name                string    `json:"name"`
	TradeName           string    `json:"trade_name"`
	Document            string    `json:"document"`
	DocumentType        string    `json:"document_type"`
	Email               string    `json:"email"`
	Phone               string    `json:"phone"`
	Website             string    `json:"website"`
	AddressStreet       string    `json:"address_street"`
	AddressNumber       string    `json:"address_number"`
	AddressComplement   string    `json:"address_complement"`
	AddressNeighborhood string    `json:"address_neighborhood"`
	AddressCity         string    `json:"address_city"`
	AddressState        string    `json:"address_state"`
	AddressZipcode      string    `json:"address_zipcode"`
	AddressCountry      string    `json:"address_country"`
	Status              any       `json:"status"`
	Timezone            string    `json:"timezone"`
	CreatedBy           uuid.UUID `json:"created_by"`
	UpdatedBy           uuid.UUID `json:"updated_by"`
	DeletedBy           uuid.UUID `json:"deleted_by"`
}

type DeleteCompanyParams struct {
	ID        uuid.UUID `json:"id"`
	DeletedBy uuid.UUID `json:"deleted_by"`
}

type UpdateCompanyRequest struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	TradeName           string    `json:"trade_name"`
	Document            string    `json:"document"`
	DocumentType        string    `json:"document_type"`
	Email               string    `json:"email"`
	Phone               string    `json:"phone"`
	Website             string    `json:"website"`
	AddressStreet       string    `json:"address_street"`
	AddressNumber       string    `json:"address_number"`
	AddressComplement   string    `json:"address_complement"`
	AddressNeighborhood string    `json:"address_neighborhood"`
	AddressCity         string    `json:"address_city"`
	AddressState        string    `json:"address_state"`
	AddressZipcode      string    `json:"address_zipcode"`
	AddressCountry      string    `json:"address_country"`
	Status              any       `json:"status"`
	Timezone            string    `json:"timezone"`
	UpdatedBy           uuid.UUID `json:"updated_by"`
}

type CompanyResponse struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	TradeName           string    `json:"trade_name"`
	Document            string    `json:"document"`
	DocumentType        string    `json:"document_type"`
	Email               string    `json:"email"`
	Phone               string    `json:"phone"`
	Website             string    `json:"website"`
	AddressStreet       string    `json:"address_street"`
	AddressNumber       string    `json:"address_number"`
	AddressComplement   string    `json:"address_complement"`
	AddressNeighborhood string    `json:"address_neighborhood"`
	AddressCity         string    `json:"address_city"`
	AddressState        string    `json:"address_state"`
	AddressZipcode      string    `json:"address_zipcode"`
	AddressCountry      string    `json:"address_country"`
	Status              any       `json:"status"`
	CreatedBy           uuid.UUID `json:"created_by"`
	UpdatedBy           uuid.UUID `json:"updated_by"`
	DeletedBy           uuid.UUID `json:"deleted_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	DeletedAt           time.Time `json:"deleted_at"`
}

type SetCompanyStatusParams struct {
	ID     uuid.UUID `json:"id"`
	Status any       `json:"status"`
}

func ApplyUpdateCompanyParams(req UpdateCompanyRequest, arg *db.UpdateCompanyParams) {
	if req.Name != "" {
		arg.Name = req.Name
	}

	if req.TradeName != "" {
		arg.TradeName = pgconv.ParseStringToPgText(req.TradeName)
	}

	if req.Document != "" {
		arg.Document = pgconv.ParseStringToPgText(req.Document)
	}

	if req.DocumentType != "" {
		arg.DocumentType = pgconv.ParseStringToPgText(req.DocumentType)
	}

	if req.Email != "" {
		arg.Email = pgconv.ParseStringToPgText(req.Email)
	}

	if req.Phone != "" {
		arg.Phone = pgconv.ParseStringToPgText(req.Phone)
	}

	if req.Website != "" {
		arg.Website = pgconv.ParseStringToPgText(req.Website)
	}

	if req.AddressStreet != "" {
		arg.AddressStreet = pgconv.ParseStringToPgText(req.AddressStreet)
	}

	if req.AddressNumber != "" {
		arg.AddressNumber = pgconv.ParseStringToPgText(req.AddressNumber)
	}

	if req.AddressComplement != "" {
		arg.AddressComplement = pgconv.ParseStringToPgText(req.AddressComplement)
	}

	if req.AddressNeighborhood != "" {
		arg.AddressNeighborhood = pgconv.ParseStringToPgText(req.AddressNeighborhood)
	}

	if req.AddressCity != "" {
		arg.AddressCity = pgconv.ParseStringToPgText(req.AddressCity)
	}

	if req.AddressState != "" {
		arg.AddressState = pgconv.ParseStringToPgText(req.AddressState)
	}

	if req.AddressZipcode != "" {
		arg.AddressZipcode = pgconv.ParseStringToPgText(req.AddressZipcode)
	}

	if req.AddressCountry != "" {
		arg.AddressCountry = pgconv.ParseStringToPgText(req.AddressCountry)
	}

	if req.Timezone != "" {
		arg.Timezone = pgconv.ParseStringToPgText(req.Timezone)
	}

	if req.UpdatedBy != (uuid.UUID{}) {
		arg.UpdatedBy = pgconv.ParseUUIDToPgType(req.UpdatedBy)
	}
}
