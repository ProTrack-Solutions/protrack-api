package domain

import (
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/google/uuid"
)

type Customer struct {
	ID                  uuid.UUID    `json:"id"`
	CompanyID           uuid.UUID    `json:"company_id"`
	FullName            string       `json:"full_name"`
	BirthDate           string       `json:"birth_date"`
	Cpf                 string       `json:"cpf"`
	Rg                  string       `json:"rg"`
	MaritalStatus       string       `json:"marital_status"`
	Gender              enums.Gender `json:"gender"`
	Whatsapp            string       `json:"whatsapp"`
	MobilePhone         string       `json:"mobile_phone"`
	HomePhone           string       `json:"home_phone"`
	Email               string       `json:"email"`
	AddressStreet       string       `json:"address_street"`
	AddressNumber       string       `json:"address_number"`
	AddressComplement   string       `json:"address_complement"`
	AddressNeighborhood string       `json:"address_neighborhood"`
	AddressCity         string       `json:"address_city"`
	AddressState        string       `json:"address_state"`
	AddressZipcode      string       `json:"address_zipcode"`
	AddressCountry      string       `json:"address_country"`
	BalanceDue          float64      `json:"balance_due"`
	CreatedBy           uuid.UUID    `json:"created_by"`
	UpdatedBy           uuid.UUID    `json:"updated_by"`
	DeletedBy           uuid.UUID    `json:"deleted_by"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
	DeletedAt           time.Time    `json:"deleted_at"`
}

type CreateCustomersRequest struct {
	CompanyID           uuid.UUID    `json:"company_id"`
	FullName            string       `json:"full_name"`
	BirthDate           string       `json:"birth_date"`
	Cpf                 string       `json:"cpf"`
	Rg                  string       `json:"rg"`
	MaritalStatus       string       `json:"marital_status"`
	Gender              enums.Gender `json:"gender"`
	Whatsapp            string       `json:"whatsapp"`
	MobilePhone         string       `json:"mobile_phone"`
	HomePhone           string       `json:"home_phone"`
	Email               string       `json:"email"`
	AddressStreet       string       `json:"address_street"`
	AddressNumber       string       `json:"address_number"`
	AddressComplement   string       `json:"address_complement"`
	AddressNeighborhood string       `json:"address_neighborhood"`
	AddressCity         string       `json:"address_city"`
	AddressState        string       `json:"address_state"`
	AddressZipcode      string       `json:"address_zipcode"`
	AddressCountry      string       `json:"address_country"`
	BalanceDue          float64      `json:"balance_due"`
	CreatedBy           uuid.UUID    `json:"created_by"`
}

type DeleteCustomerRequest struct {
	ID        uuid.UUID `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
	DeletedBy uuid.UUID `json:"deleted_by"`
}

type UpdateBalanceDueCustomerRequest struct {
	ID         uuid.UUID `json:"id"`
	BalanceDue float64   `json:"balance_due"`
	Prohibited float64   `json:"prohibited"`
	UpdatedBy  uuid.UUID `json:"updated_by"`
}

type UpdateCustomerRequest struct {
	ID                  uuid.UUID    `json:"id"`
	FullName            string       `json:"full_name"`
	BirthDate           string       `json:"birth_date"`
	Cpf                 string       `json:"cpf"`
	Rg                  string       `json:"rg"`
	MaritalStatus       string       `json:"marital_status"`
	Gender              enums.Gender `json:"gender"`
	Whatsapp            string       `json:"whatsapp"`
	MobilePhone         string       `json:"mobile_phone"`
	HomePhone           string       `json:"home_phone"`
	Email               string       `json:"email"`
	AddressStreet       string       `json:"address_street"`
	AddressNumber       string       `json:"address_number"`
	AddressComplement   string       `json:"address_complement"`
	AddressNeighborhood string       `json:"address_neighborhood"`
	AddressCity         string       `json:"address_city"`
	AddressState        string       `json:"address_state"`
	AddressZipcode      string       `json:"address_zipcode"`
	AddressCountry      string       `json:"address_country"`
	BalanceDue          float64      `json:"balance_due"`
	UpdatedBy           uuid.UUID    `json:"updated_by"`
}

type CustomerResponse struct {
	ID                  uuid.UUID    `json:"id"`
	CompanyID           uuid.UUID    `json:"company_id"`
	FullName            string       `json:"full_name"`
	BirthDate           string       `json:"birth_date"`
	Cpf                 string       `json:"cpf"`
	Rg                  string       `json:"rg"`
	MaritalStatus       string       `json:"marital_status"`
	Gender              enums.Gender `json:"gender"`
	Whatsapp            string       `json:"whatsapp"`
	MobilePhone         string       `json:"mobile_phone"`
	HomePhone           string       `json:"home_phone"`
	Email               string       `json:"email"`
	AddressStreet       string       `json:"address_street"`
	AddressNumber       string       `json:"address_number"`
	AddressComplement   string       `json:"address_complement"`
	AddressNeighborhood string       `json:"address_neighborhood"`
	AddressCity         string       `json:"address_city"`
	AddressState        string       `json:"address_state"`
	AddressZipcode      string       `json:"address_zipcode"`
	AddressCountry      string       `json:"address_country"`
	BalanceDue          float64      `json:"balance_due"`
	CreatedBy           uuid.UUID    `json:"created_by"`
	UpdatedBy           uuid.UUID    `json:"updated_by"`
	DeletedBy           uuid.UUID    `json:"deleted_by"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
	DeletedAt           time.Time    `json:"deleted_at"`
}

type CustomerPaginatedResponse struct {
	globalDomain.PaginatedResponse[CustomerResponse]
}

func ApplyUpdateCustomerParams(req UpdateCustomerRequest, arg *db.UpdateCustomerParams) {
	if req.FullName != "" {
		arg.FullName = req.FullName
	}

	if req.BirthDate != "" {
		arg.BirthDate = pgconv.StringToPgDate(req.BirthDate)
	}

	if req.Cpf != "" {
		arg.Cpf = req.Cpf
	}

	if req.Rg != "" {
		arg.Rg = pgconv.ParseStringToPgText(req.Rg)
	}

	if req.MaritalStatus != "" {
		arg.MaritalStatus = pgconv.ParseStringToPgText(req.MaritalStatus)
	}

	if req.Whatsapp != "" {
		arg.Whatsapp = pgconv.ParseStringToPgText(req.Whatsapp)
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

	if req.BalanceDue != 0 {
		arg.BalanceDue = pgconv.Float64ToPgNumeric(req.BalanceDue)
	}

	if req.UpdatedBy != uuid.Nil {
		arg.UpdatedBy = pgconv.ParseUUIDToPgType(req.UpdatedBy)
	}
}
