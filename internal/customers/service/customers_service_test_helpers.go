package service

import (
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// NewServiceWithRepo é um construtor alternativo para testes que aceita
// qualquer RepositoryInterface (incluindo mocks).
// Não deve ser utilizado em código de produção.
func NewServiceWithRepo(repo RepositoryInterface) *Service {
	return &Service{
		repo: repo,
		pool: nil,
	}
}

// buildListCustomersPaginateRow constrói um db.ListCustomersPaginateRow completo para uso nos testes de paginação.
func BuildListCustomersPaginateRow(id, companyID, createdBy uuid.UUID, totalCount int64) db.ListCustomersPaginateRow {
	now := time.Now().UTC()
	birthDate := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)
	return db.ListCustomersPaginateRow{
		ID:                  pgconv.ParseUUIDToPgType(id),
		CompanyID:           pgconv.ParseUUIDToPgType(companyID),
		FullName:            "Maria Silva",
		BirthDate:           pgconv.ToPgDate(birthDate),
		Cpf:                 "123.456.789-00",
		Rg:                  pgtype.Text{String: "MG1234567", Valid: true},
		MaritalStatus:       pgtype.Text{String: "solteira", Valid: true},
		Gender:              "FEMALE",
		Whatsapp:            pgtype.Text{String: "+5531999999999", Valid: true},
		MobilePhone:         pgtype.Text{String: "+5531988888888", Valid: true},
		HomePhone:           pgtype.Text{String: "", Valid: false},
		Email:               "maria@email.com",
		AddressStreet:       pgtype.Text{String: "Rua das Flores", Valid: true},
		AddressNumber:       pgtype.Text{String: "123", Valid: true},
		AddressComplement:   pgtype.Text{String: "Apto 5", Valid: true},
		AddressNeighborhood: pgtype.Text{String: "Centro", Valid: true},
		AddressCity:         pgtype.Text{String: "Belo Horizonte", Valid: true},
		AddressState:        pgtype.Text{String: "MG", Valid: true},
		AddressZipcode:      pgtype.Text{String: "30000-000", Valid: true},
		AddressCountry:      pgtype.Text{String: "Brasil", Valid: true},
		BalanceDue:          pgconv.Float64ToPgNumeric(250.00),
		CreatedBy:           pgconv.ParseUUIDToPgType(createdBy),
		UpdatedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		DeletedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		CreatedAt:           pgconv.TimeToPgTimestamptz(now),
		UpdatedAt:           pgconv.TimeToPgTimestamptz(now),
		DeletedAt:           pgtype.Timestamptz{Valid: false},
		TotalCount:          totalCount,
	}
}
