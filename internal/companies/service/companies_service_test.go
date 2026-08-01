package service_test

import (
	"context"
	"errors"
	"testing"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/companies/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers e Fixtures
// ---------------------------------------------------------------------------

var errDatabase = errors.New("database error")

// newSvc cria um Service com o mock de repositório injetado.
func newSvc(t *testing.T, repo *mocks.MockRepositoryInterface) *service.Service {
	t.Helper()
	return service.NewServiceWithRepo(repo)
}

// buildDbCompany constrói um db.Company completo para uso nos testes.
func buildDbCompany(id uuid.UUID, name string) db.Company {
	return db.Company{
		ID:                  pgconv.ParseUUIDToPgType(id),
		Name:                name,
		TradeName:           pgtype.Text{String: "Trade " + name, Valid: true},
		Document:            pgtype.Text{String: "00000000000191", Valid: true},
		DocumentType:        pgtype.Text{String: "CNPJ", Valid: true},
		Email:               pgtype.Text{String: "empresa@email.com", Valid: true},
		Phone:               pgtype.Text{String: "+5511999999999", Valid: true},
		Website:             pgtype.Text{String: "https://empresa.com.br", Valid: true},
		AddressStreet:       pgtype.Text{String: "Rua das Palmeiras", Valid: true},
		AddressNumber:       pgtype.Text{String: "100", Valid: true},
		AddressComplement:   pgtype.Text{Valid: false},
		AddressNeighborhood: pgtype.Text{String: "Centro", Valid: true},
		AddressCity:         pgtype.Text{String: "São Paulo", Valid: true},
		AddressState:        pgtype.Text{String: "SP", Valid: true},
		AddressZipcode:      pgtype.Text{String: "01000-000", Valid: true},
		AddressCountry:      pgtype.Text{String: "Brasil", Valid: true},
		Status:              "active",
		Timezone:            pgtype.Text{String: "America/Sao_Paulo", Valid: true},
		CreatedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		UpdatedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		DeletedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		CreatedAt:           pgconv.TimeToPgTimestamptz(pgconv.PgTimestamptzToTime(pgtype.Timestamptz{Valid: false})),
		UpdatedAt:           pgtype.Timestamptz{Valid: false},
		DeletedAt:           pgtype.Timestamptz{Valid: false},
	}
}

// ---------------------------------------------------------------------------
// DeleteCompany
// ---------------------------------------------------------------------------

func TestDeleteCompany_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	deletedBy := uuid.New()

	repo.EXPECT().
		DeleteCompany(gomock.Any(), db.DeleteCompanyParams{
			ID:        pgconv.ParseUUIDToPgType(id),
			DeletedBy: pgconv.ParseUUIDToPgType(deletedBy),
		}).
		Return(nil)

	err := svc.DeleteCompany(context.Background(), domain.DeleteCompanyParams{
		ID:        id,
		DeletedBy: deletedBy,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestDeleteCompany_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		DeleteCompany(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.DeleteCompany(context.Background(), domain.DeleteCompanyParams{
		ID:        uuid.New(),
		DeletedBy: uuid.New(),
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetCompanyByID
// ---------------------------------------------------------------------------

func TestGetCompanyByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	expected := buildDbCompany(id, "Empresa Alpha")

	repo.EXPECT().
		GetCompanyByID(gomock.Any(), pgconv.ParseUUIDToPgType(id)).
		Return(expected, nil)

	resp, err := svc.GetCompanyByID(context.Background(), id)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
	if resp.Name != "Empresa Alpha" {
		t.Errorf("Name incorreto: '%s'", resp.Name)
	}
	if resp.Document != "00000000000191" {
		t.Errorf("Document incorreto: '%s'", resp.Document)
	}
}

func TestGetCompanyByID_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCompanyByID(gomock.Any(), gomock.Any()).
		Return(db.Company{}, errDatabase)

	_, err := svc.GetCompanyByID(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetCompanyByDocument
// ---------------------------------------------------------------------------

func TestGetCompanyByDocument_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	document := "00000000000191"
	expected := buildDbCompany(id, "Empresa Beta")

	repo.EXPECT().
		GetCompanyByDocument(gomock.Any(), pgconv.ParseStringToPgText(document)).
		Return(expected, nil)

	resp, err := svc.GetCompanyByDocument(context.Background(), document)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
	if resp.Document != document {
		t.Errorf("Document incorreto: '%s'", resp.Document)
	}
}

func TestGetCompanyByDocument_InvalidDocument(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Não chamamos o repo pois a validação deve falhar antes
	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	_, err := svc.GetCompanyByDocument(context.Background(), "documento-invalido")

	if err == nil {
		t.Fatal("esperava erro de documento inválido")
	}
}

func TestGetCompanyByDocument_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	document := "00000000000191"

	repo.EXPECT().
		GetCompanyByDocument(gomock.Any(), gomock.Any()).
		Return(db.Company{}, errDatabase)

	_, err := svc.GetCompanyByDocument(context.Background(), document)

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListCompanies
// ---------------------------------------------------------------------------

func TestListCompanies_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companies := []db.Company{
		buildDbCompany(uuid.New(), "Empresa A"),
		buildDbCompany(uuid.New(), "Empresa B"),
		buildDbCompany(uuid.New(), "Empresa C"),
	}

	repo.EXPECT().
		ListCompanies(gomock.Any()).
		Return(companies, nil)

	resp, err := svc.ListCompanies(context.Background())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 3 {
		t.Errorf("esperava 3 empresas, obteve %d", len(resp))
	}
}

func TestListCompanies_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListCompanies(gomock.Any()).
		Return([]db.Company{}, nil)

	resp, err := svc.ListCompanies(context.Background())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("esperava lista vazia, obteve %d", len(resp))
	}
}

func TestListCompanies_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListCompanies(gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListCompanies(context.Background())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// SetCompanyStatus
// ---------------------------------------------------------------------------

func TestSetCompanyStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()

	repo.EXPECT().
		SetCompanyStatus(gomock.Any(), db.SetCompanyStatusParams{
			ID:      pgconv.ParseUUIDToPgType(id),
			Column2: "inactive",
		}).
		Return(int64(1), nil)

	count, err := svc.SetCompanyStatus(context.Background(), domain.SetCompanyStatusParams{
		ID:     id,
		Status: "inactive",
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 1 {
		t.Errorf("esperava count=1, obteve %d", count)
	}
}

func TestSetCompanyStatus_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		SetCompanyStatus(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	_, err := svc.SetCompanyStatus(context.Background(), domain.SetCompanyStatusParams{
		ID:     uuid.New(),
		Status: "active",
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// UpdateCompany
// ---------------------------------------------------------------------------

func TestUpdateCompany_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	currentCompany := buildDbCompany(id, "Empresa Original")
	updatedCompany := buildDbCompany(id, "Empresa Atualizada")

	repo.EXPECT().
		GetCompanyByID(gomock.Any(), pgconv.ParseUUIDToPgType(id)).
		Return(currentCompany, nil)

	repo.EXPECT().
		UpdateCompany(gomock.Any(), gomock.Any()).
		Return(updatedCompany, nil)

	resp, err := svc.UpdateCompany(context.Background(), id, domain.UpdateCompanyRequest{
		Name: "Empresa Atualizada",
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Name != "Empresa Atualizada" {
		t.Errorf("Name incorreto após update: '%s'", resp.Name)
	}
}

func TestUpdateCompany_CompanyNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCompanyByID(gomock.Any(), gomock.Any()).
		Return(db.Company{}, errors.New("not found"))

	_, err := svc.UpdateCompany(context.Background(), uuid.New(), domain.UpdateCompanyRequest{
		Name: "Teste",
	})

	if err == nil {
		t.Fatal("esperava erro ao buscar empresa")
	}
}

func TestUpdateCompany_UpdateRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	currentCompany := buildDbCompany(id, "Empresa Atual")

	repo.EXPECT().
		GetCompanyByID(gomock.Any(), gomock.Any()).
		Return(currentCompany, nil)

	repo.EXPECT().
		UpdateCompany(gomock.Any(), gomock.Any()).
		Return(db.Company{}, errDatabase)

	_, err := svc.UpdateCompany(context.Background(), id, domain.UpdateCompanyRequest{
		Name: "Empresa Nova",
	})

	if err == nil {
		t.Fatal("esperava erro do repositório ao atualizar")
	}
}

// ---------------------------------------------------------------------------
// Teste de conversão dos campos opcionais (pgtype.Text)
// ---------------------------------------------------------------------------

func TestGetCompanyByID_NullableFieldsHandled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()

	// Empresa com campos opcionais nulos
	dbCompany := db.Company{
		ID:                  pgconv.ParseUUIDToPgType(id),
		Name:                "Empresa Mínima",
		TradeName:           pgtype.Text{Valid: false},
		Document:            pgtype.Text{Valid: false},
		DocumentType:        pgtype.Text{Valid: false},
		Email:               pgtype.Text{Valid: false},
		Phone:               pgtype.Text{Valid: false},
		Website:             pgtype.Text{Valid: false},
		AddressStreet:       pgtype.Text{Valid: false},
		AddressNumber:       pgtype.Text{Valid: false},
		AddressComplement:   pgtype.Text{Valid: false},
		AddressNeighborhood: pgtype.Text{Valid: false},
		AddressCity:         pgtype.Text{Valid: false},
		AddressState:        pgtype.Text{Valid: false},
		AddressZipcode:      pgtype.Text{Valid: false},
		AddressCountry:      pgtype.Text{Valid: false},
		Status:              "active",
		CreatedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		UpdatedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		DeletedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		CreatedAt:           pgtype.Timestamptz{Valid: false},
		UpdatedAt:           pgtype.Timestamptz{Valid: false},
		DeletedAt:           pgtype.Timestamptz{Valid: false},
	}

	repo.EXPECT().
		GetCompanyByID(gomock.Any(), gomock.Any()).
		Return(dbCompany, nil)

	resp, err := svc.GetCompanyByID(context.Background(), id)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	// Campos nulos devem virar string vazia
	if resp.TradeName != "" {
		t.Errorf("TradeName nulo deve ser string vazia, obteve '%s'", resp.TradeName)
	}
	if resp.Email != "" {
		t.Errorf("Email nulo deve ser string vazia, obteve '%s'", resp.Email)
	}
	if resp.AddressCity != "" {
		t.Errorf("AddressCity nulo deve ser string vazia, obteve '%s'", resp.AddressCity)
	}
	if resp.Name != "Empresa Mínima" {
		t.Errorf("Name incorreto: '%s'", resp.Name)
	}
}
