package domain_test

import (
	"testing"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/customers/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildUpdateParams(id uuid.UUID) db.UpdateCustomerParams {
	now := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)
	return db.UpdateCustomerParams{
		ID:                  pgconv.ParseUUIDToPgType(id),
		FullName:            "Nome Original",
		BirthDate:           pgconv.ToPgDate(now),
		Cpf:                 "000.000.000-00",
		Rg:                  pgtype.Text{String: "MG1234567", Valid: true},
		MaritalStatus:       pgtype.Text{String: "solteiro", Valid: true},
		Gender:              pgtype.Text{String: "MALE", Valid: true},
		Whatsapp:            pgtype.Text{String: "+5531900000000", Valid: true},
		MobilePhone:         pgtype.Text{String: "+5531911111111", Valid: true},
		HomePhone:           pgtype.Text{String: "+5531922222222", Valid: true},
		Email:               "original@email.com",
		AddressStreet:       pgtype.Text{String: "Rua A", Valid: true},
		AddressNumber:       pgtype.Text{String: "100", Valid: true},
		AddressComplement:   pgtype.Text{String: "Apto 1", Valid: true},
		AddressNeighborhood: pgtype.Text{String: "Bairro X", Valid: true},
		AddressCity:         pgtype.Text{String: "Cidade Y", Valid: true},
		AddressState:        pgtype.Text{String: "MG", Valid: true},
		AddressZipcode:      pgtype.Text{String: "30000-000", Valid: true},
		AddressCountry:      pgtype.Text{String: "Brasil", Valid: true},
		BalanceDue:          pgconv.Float64ToPgNumeric(0),
		UpdatedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
	}
}

// ---------------------------------------------------------------------------
// TestApplyUpdateCustomerParams — atualização completa
// ---------------------------------------------------------------------------

func TestApplyUpdateCustomerParams_UpdatesAllFields(t *testing.T) {
	id := uuid.New()
	updatedBy := uuid.New()
	arg := buildUpdateParams(id)

	req := domain.UpdateCustomerRequest{
		FullName:            "Nome Novo",
		BirthDate:           "2000-01-20",
		Cpf:                 "111.111.111-11",
		Rg:                  "SP9876543",
		MaritalStatus:       "casado",
		Gender:              enums.GenderFemale,
		Whatsapp:            "+5511999999999",
		MobilePhone:         "+5511888888888",
		HomePhone:           "+5511777777777",
		Email:               "novo@email.com",
		AddressStreet:       "Rua Nova",
		AddressNumber:       "200",
		AddressComplement:   "Casa",
		AddressNeighborhood: "Bairro Novo",
		AddressCity:         "São Paulo",
		AddressState:        "SP",
		AddressZipcode:      "01000-000",
		AddressCountry:      "Brasil",
		BalanceDue:          500.00,
		UpdatedBy:           updatedBy,
	}

	domain.ApplyUpdateCustomerParams(req, &arg)

	if arg.FullName != "Nome Novo" {
		t.Errorf("FullName: esperava 'Nome Novo', obteve '%s'", arg.FullName)
	}
	if arg.Cpf != "111.111.111-11" {
		t.Errorf("Cpf: esperava '111.111.111-11', obteve '%s'", arg.Cpf)
	}
	if arg.Rg.String != "SP9876543" {
		t.Errorf("Rg: esperava 'SP9876543', obteve '%s'", arg.Rg.String)
	}
	if arg.MaritalStatus.String != "casado" {
		t.Errorf("MaritalStatus: esperava 'casado', obteve '%s'", arg.MaritalStatus.String)
	}
	if arg.Gender.(pgtype.Text).String != "FEMALE" {
		t.Errorf("Gender: esperava 'FEMALE'")
	}
	if arg.Whatsapp.String != "+5511999999999" {
		t.Errorf("Whatsapp: esperava '+5511999999999', obteve '%s'", arg.Whatsapp.String)
	}
	if arg.MobilePhone.String != "+5511888888888" {
		t.Errorf("MobilePhone incorreto: '%s'", arg.MobilePhone.String)
	}
	if arg.HomePhone.String != "+5511777777777" {
		t.Errorf("HomePhone incorreto: '%s'", arg.HomePhone.String)
	}
	if arg.Email != "novo@email.com" {
		t.Errorf("Email incorreto: '%s'", arg.Email)
	}
	if arg.AddressStreet.String != "Rua Nova" {
		t.Errorf("AddressStreet incorreto: '%s'", arg.AddressStreet.String)
	}
	if arg.AddressCity.String != "São Paulo" {
		t.Errorf("AddressCity incorreto: '%s'", arg.AddressCity.String)
	}
	if arg.AddressState.String != "SP" {
		t.Errorf("AddressState incorreto: '%s'", arg.AddressState.String)
	}
	if arg.AddressZipcode.String != "01000-000" {
		t.Errorf("AddressZipcode incorreto: '%s'", arg.AddressZipcode.String)
	}
	if pgconv.PgNumericToFloat64(arg.BalanceDue) != 500.00 {
		t.Errorf("BalanceDue: esperava 500.00, obteve %f", pgconv.PgNumericToFloat64(arg.BalanceDue))
	}
	if pgconv.PgUUIDToUUID(arg.UpdatedBy) != updatedBy {
		t.Errorf("UpdatedBy incorreto")
	}
}

// ---------------------------------------------------------------------------
// TestApplyUpdateCustomerParams — campos zerados não sobrescrevem
// ---------------------------------------------------------------------------

func TestApplyUpdateCustomerParams_DoesNotOverwriteWithZeroValues(t *testing.T) {
	id := uuid.New()
	arg := buildUpdateParams(id)

	// Req com todos os campos zerados
	req := domain.UpdateCustomerRequest{
		FullName:      "",
		BirthDate:     "",
		Cpf:           "",
		Rg:            "",
		MaritalStatus: "",
		Gender:        "",
		Whatsapp:      "",
		MobilePhone:   "",
		HomePhone:     "",
		Email:         "",
		BalanceDue:    0,
		UpdatedBy:     uuid.Nil,
	}

	domain.ApplyUpdateCustomerParams(req, &arg)

	if arg.FullName != "Nome Original" {
		t.Errorf("FullName não deve ter sido alterado; obteve '%s'", arg.FullName)
	}
	if arg.Cpf != "000.000.000-00" {
		t.Errorf("Cpf não deve ter sido alterado; obteve '%s'", arg.Cpf)
	}
	if arg.Email != "original@email.com" {
		t.Errorf("Email não deve ter sido alterado; obteve '%s'", arg.Email)
	}
	if arg.Rg.String != "MG1234567" {
		t.Errorf("Rg não deve ter sido alterado; obteve '%s'", arg.Rg.String)
	}
}

// ---------------------------------------------------------------------------
// TestApplyUpdateCustomerParams — atualização parcial
// ---------------------------------------------------------------------------

func TestApplyUpdateCustomerParams_PartialUpdate(t *testing.T) {
	id := uuid.New()
	arg := buildUpdateParams(id)

	// Atualiza apenas email e city
	req := domain.UpdateCustomerRequest{
		Email:       "parcial@email.com",
		AddressCity: "Belo Horizonte",
	}

	domain.ApplyUpdateCustomerParams(req, &arg)

	if arg.Email != "parcial@email.com" {
		t.Errorf("Email: esperava 'parcial@email.com', obteve '%s'", arg.Email)
	}
	if arg.AddressCity.String != "Belo Horizonte" {
		t.Errorf("AddressCity: esperava 'Belo Horizonte', obteve '%s'", arg.AddressCity.String)
	}
	// Campos que não foram tocados devem manter os originais
	if arg.FullName != "Nome Original" {
		t.Errorf("FullName não deve ter sido alterado; obteve '%s'", arg.FullName)
	}
	if arg.Cpf != "000.000.000-00" {
		t.Errorf("Cpf não deve ter sido alterado; obteve '%s'", arg.Cpf)
	}
}

// ---------------------------------------------------------------------------
// TestGenderEnum
// ---------------------------------------------------------------------------

func TestGender_IsValid(t *testing.T) {
	cases := []struct {
		gender enums.Gender
		valid  bool
	}{
		{enums.GenderMale, true},
		{enums.GenderFemale, true},
		{enums.GenderOther, true},
		{enums.GenderNotSay, true},
		{"INVALID", false},
		{"", false},
	}

	for _, tc := range cases {
		t.Run(string(tc.gender), func(t *testing.T) {
			if tc.gender.IsValid() != tc.valid {
				t.Errorf("Gender('%s').IsValid() = %v, esperava %v", tc.gender, tc.gender.IsValid(), tc.valid)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestCreateCustomersRequest — validação de campos
// ---------------------------------------------------------------------------

func TestCreateCustomersRequest_FieldAssignment(t *testing.T) {
	companyID := uuid.New()
	createdBy := uuid.New()

	req := domain.CreateCustomersRequest{
		CompanyID:   companyID,
		FullName:    "João Silva",
		BirthDate:   "1990-05-15",
		Cpf:         "123.456.789-00",
		Rg:          "MG1234567",
		Email:       "joao@email.com",
		Gender:      enums.GenderMale,
		BalanceDue:  100.50,
		CreatedBy:   createdBy,
	}

	if req.CompanyID != companyID {
		t.Errorf("CompanyID incorreto")
	}
	if req.FullName != "João Silva" {
		t.Errorf("FullName incorreto: '%s'", req.FullName)
	}
	if req.Gender != enums.GenderMale {
		t.Errorf("Gender incorreto: '%s'", req.Gender)
	}
	if req.BalanceDue != 100.50 {
		t.Errorf("BalanceDue incorreto: %f", req.BalanceDue)
	}
}

// ---------------------------------------------------------------------------
// TestDeleteCustomerRequest
// ---------------------------------------------------------------------------

func TestDeleteCustomerRequest_FieldAssignment(t *testing.T) {
	id := uuid.New()
	deletedBy := uuid.New()
	now := time.Now()

	req := domain.DeleteCustomerRequest{
		ID:        id,
		DeletedAt: now,
		DeletedBy: deletedBy,
	}

	if req.ID != id {
		t.Errorf("ID incorreto")
	}
	if req.DeletedBy != deletedBy {
		t.Errorf("DeletedBy incorreto")
	}
	if req.DeletedAt.IsZero() {
		t.Errorf("DeletedAt não deve ser zero")
	}
}

// ---------------------------------------------------------------------------
// TestUpdateBalanceDueCustomerRequest
// ---------------------------------------------------------------------------

func TestUpdateBalanceDueCustomerRequest_FieldAssignment(t *testing.T) {
	updatedBy := uuid.New()

	req := domain.UpdateBalanceDueCustomerRequest{
		BalanceDue: 200.00,
		Prohibited: 50.00,
		UpdatedBy:  updatedBy,
	}

	if req.BalanceDue != 200.00 {
		t.Errorf("BalanceDue incorreto: %f", req.BalanceDue)
	}
	if req.Prohibited != 50.00 {
		t.Errorf("Prohibited incorreto: %f", req.Prohibited)
	}
	if req.UpdatedBy != updatedBy {
		t.Errorf("UpdatedBy incorreto")
	}
}
