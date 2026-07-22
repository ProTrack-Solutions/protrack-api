package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/customers/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/customers/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/customers/service"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
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

// buildDbCustomer constrói um db.Customer completo para uso nos testes.
func buildDbCustomer(id, companyID, createdBy uuid.UUID) db.Customer {
	now := time.Now().UTC()
	birthDate := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)
	return db.Customer{
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
	}
}

// ---------------------------------------------------------------------------
// CreateCustomer
// ---------------------------------------------------------------------------

func TestCreateCustomer_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	createdBy := uuid.New()
	newID := uuid.New()

	repo.EXPECT().
		CreateCustomer(gomock.Any(), gomock.Any()).
		Return(pgconv.ParseUUIDToPgType(newID), nil)

	req := domain.CreateCustomersRequest{
		CompanyID:   companyID,
		FullName:    "João Santos",
		BirthDate:   "1985-03-10",
		Cpf:         "111.222.333-44",
		Email:       "joao@email.com",
		Gender:      enums.GenderMale,
		BalanceDue:  0,
		CreatedBy:   createdBy,
	}

	id, err := svc.CreateCustomer(context.Background(), req)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if id != newID {
		t.Errorf("ID retornado incorreto: esperava %v, obteve %v", newID, id)
	}
}

func TestCreateCustomer_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		CreateCustomer(gomock.Any(), gomock.Any()).
		Return(pgtype.UUID{}, errDatabase)

	_, err := svc.CreateCustomer(context.Background(), domain.CreateCustomersRequest{
		CompanyID: uuid.New(),
		FullName:  "Teste",
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// DeleteCustomer
// ---------------------------------------------------------------------------

func TestDeleteCustomer_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	deletedBy := uuid.New()

	repo.EXPECT().
		DeleteCustomer(gomock.Any(), db.DeleteCustomerParams{
			ID:        pgconv.ParseUUIDToPgType(id),
			DeletedBy: pgconv.ParseUUIDToPgType(deletedBy),
		}).
		Return(nil)

	err := svc.DeleteCustomer(context.Background(), domain.DeleteCustomerRequest{
		ID:        id,
		DeletedBy: deletedBy,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestDeleteCustomer_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		DeleteCustomer(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.DeleteCustomer(context.Background(), domain.DeleteCustomerRequest{
		ID:        uuid.New(),
		DeletedBy: uuid.New(),
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetCustomerByCPF
// ---------------------------------------------------------------------------

func TestGetCustomerByCPF_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	createdBy := uuid.New()
	cpf := "123.456.789-00"

	expected := buildDbCustomer(id, companyID, createdBy)

	repo.EXPECT().
		GetCustomerByCPF(gomock.Any(), cpf).
		Return(expected, nil)

	resp, err := svc.GetCustomerByCPF(context.Background(), cpf)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Cpf != cpf {
		t.Errorf("Cpf: esperava '%s', obteve '%s'", cpf, resp.Cpf)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
	if resp.Email != "maria@email.com" {
		t.Errorf("Email incorreto: '%s'", resp.Email)
	}
	if resp.Gender != enums.GenderFemale {
		t.Errorf("Gender incorreto: '%s'", resp.Gender)
	}
	if resp.BalanceDue != 250.00 {
		t.Errorf("BalanceDue: esperava 250.00, obteve %f", resp.BalanceDue)
	}
}

func TestGetCustomerByCPF_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCustomerByCPF(gomock.Any(), gomock.Any()).
		Return(db.Customer{}, errors.New("no rows in result set"))

	_, err := svc.GetCustomerByCPF(context.Background(), "000.000.000-00")

	if err == nil {
		t.Fatal("esperava erro de cliente não encontrado")
	}
}

// ---------------------------------------------------------------------------
// GetCustomerById
// ---------------------------------------------------------------------------

func TestGetCustomerById_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	createdBy := uuid.New()

	expected := buildDbCustomer(id, companyID, createdBy)

	repo.EXPECT().
		GetCustomerById(gomock.Any(), pgconv.ParseUUIDToPgType(id)).
		Return(expected, nil)

	resp, err := svc.GetCustomerById(context.Background(), id)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID incorreto")
	}
	if resp.FullName != "Maria Silva" {
		t.Errorf("FullName: esperava 'Maria Silva', obteve '%s'", resp.FullName)
	}
	if resp.CompanyID != companyID {
		t.Errorf("CompanyID incorreto")
	}
}

func TestGetCustomerById_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCustomerById(gomock.Any(), gomock.Any()).
		Return(db.Customer{}, errDatabase)

	_, err := svc.GetCustomerById(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListCustomers
// ---------------------------------------------------------------------------

func TestListCustomers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	dbCustomers := []db.Customer{
		buildDbCustomer(uuid.New(), companyID, uuid.New()),
		buildDbCustomer(uuid.New(), companyID, uuid.New()),
		buildDbCustomer(uuid.New(), companyID, uuid.New()),
	}

	repo.EXPECT().
		ListCustomers(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(dbCustomers, nil)

	resp, err := svc.ListCustomers(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 3 {
		t.Errorf("esperava 3 clientes, obteve %d", len(resp))
	}
	for _, c := range resp {
		if c.CompanyID != companyID {
			t.Errorf("CompanyID incorreto para cliente '%s'", c.FullName)
		}
	}
}

func TestListCustomers_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListCustomers(gomock.Any(), gomock.Any()).
		Return([]db.Customer{}, nil)

	resp, err := svc.ListCustomers(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("esperava lista vazia, obteve %d", len(resp))
	}
}

func TestListCustomers_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListCustomers(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListCustomers(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// UpdateBalanceDueCustomer
// ---------------------------------------------------------------------------

func TestUpdateBalanceDueCustomer_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	companyID := uuid.New()
	updatedBy := uuid.New()

	currentCustomer := buildDbCustomer(id, companyID, uuid.New())

	repo.EXPECT().
		GetCustomerById(gomock.Any(), pgconv.ParseUUIDToPgType(id)).
		Return(currentCustomer, nil)

	repo.EXPECT().
		UpdateBalanceDueCustomer(gomock.Any(), gomock.Any()).
		Return(nil)

	err := svc.UpdateBalanceDueCustomer(context.Background(), id, domain.UpdateBalanceDueCustomerRequest{
		BalanceDue: 500.00,
		UpdatedBy:  updatedBy,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestUpdateBalanceDueCustomer_CustomerNotFound_ReturnsNil(t *testing.T) {
	// Atenção: a implementação atual retorna nil quando o cliente não é encontrado
	// (comportamento do serviço original — linha: return nil)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCustomerById(gomock.Any(), gomock.Any()).
		Return(db.Customer{}, errors.New("not found"))

	// Service retorna nil quando GetCustomerById falha (comportamento atual)
	err := svc.UpdateBalanceDueCustomer(context.Background(), uuid.New(), domain.UpdateBalanceDueCustomerRequest{
		BalanceDue: 100.00,
	})

	if err != nil {
		t.Errorf("service retorna nil quando cliente não encontrado; obteve: %v", err)
	}
}

func TestUpdateBalanceDueCustomer_UpdateRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	repo.EXPECT().
		GetCustomerById(gomock.Any(), gomock.Any()).
		Return(buildDbCustomer(id, uuid.New(), uuid.New()), nil)

	repo.EXPECT().
		UpdateBalanceDueCustomer(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.UpdateBalanceDueCustomer(context.Background(), id, domain.UpdateBalanceDueCustomerRequest{
		BalanceDue: 300.00,
		UpdatedBy:  uuid.New(),
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// UpdateCustomer
// ---------------------------------------------------------------------------

func TestUpdateCustomer_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	updatedBy := uuid.New()

	repo.EXPECT().
		GetCustomerById(gomock.Any(), pgconv.ParseUUIDToPgType(id)).
		Return(buildDbCustomer(id, uuid.New(), uuid.New()), nil)

	repo.EXPECT().
		UpdateCustomer(gomock.Any(), gomock.Any()).
		Return(nil)

	err := svc.UpdateCustomer(context.Background(), id, domain.UpdateCustomerRequest{
		FullName:  "Nome Atualizado",
		Email:     "atualizado@email.com",
		UpdatedBy: updatedBy,
	})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestUpdateCustomer_CustomerNotFound_ReturnsNil(t *testing.T) {
	// Comportamento atual: retorna nil quando GetCustomerById falha
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCustomerById(gomock.Any(), gomock.Any()).
		Return(db.Customer{}, errors.New("not found"))

	err := svc.UpdateCustomer(context.Background(), uuid.New(), domain.UpdateCustomerRequest{
		FullName: "Teste",
	})

	if err != nil {
		t.Errorf("service retorna nil quando cliente não encontrado; obteve: %v", err)
	}
}

func TestUpdateCustomer_UpdateRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()
	repo.EXPECT().
		GetCustomerById(gomock.Any(), gomock.Any()).
		Return(buildDbCustomer(id, uuid.New(), uuid.New()), nil)

	repo.EXPECT().
		UpdateCustomer(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.UpdateCustomer(context.Background(), id, domain.UpdateCustomerRequest{
		FullName:  "Teste",
		UpdatedBy: uuid.New(),
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// CountCustomers
// ---------------------------------------------------------------------------

func TestCountCustomers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		CountCustomers(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(int64(15), nil)

	count, err := svc.CountCustomers(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if count != 15 {
		t.Errorf("esperava 15, obteve %d", count)
	}
}

func TestCountCustomers_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		CountCustomers(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	_, err := svc.CountCustomers(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// GetCustomersPerformanceSummary
// ---------------------------------------------------------------------------

func TestGetCustomersPerformanceSummary_PositiveGrowth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		GetCustomersPerformanceSummary(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(db.GetCustomersPerformanceSummaryRow{
			CurrentMonthCount: 120,
			LastMonthCount:    100,
		}, nil)

	// percentage = ((120 - 100) / 100) * 100 = 20%
	percentage, err := svc.GetCustomersPerformanceSummary(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 20.0 {
		t.Errorf("esperava 20%%, obteve %f%%", percentage)
	}
}

func TestGetCustomersPerformanceSummary_NegativeGrowth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCustomersPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetCustomersPerformanceSummaryRow{
			CurrentMonthCount: 60,
			LastMonthCount:    100,
		}, nil)

	// percentage = ((60 - 100) / 100) * 100 = -40%
	percentage, err := svc.GetCustomersPerformanceSummary(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != -40.0 {
		t.Errorf("esperava -40%%, obteve %f%%", percentage)
	}
}

func TestGetCustomersPerformanceSummary_NoLastMonth_HasCurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCustomersPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetCustomersPerformanceSummaryRow{
			CurrentMonthCount: 30,
			LastMonthCount:    0,
		}, nil)

	// lastMonthCount == 0 e currentMonthCount > 0 → 100%
	percentage, err := svc.GetCustomersPerformanceSummary(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 100.0 {
		t.Errorf("esperava 100%%, obteve %f%%", percentage)
	}
}

func TestGetCustomersPerformanceSummary_BothZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCustomersPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetCustomersPerformanceSummaryRow{
			CurrentMonthCount: 0,
			LastMonthCount:    0,
		}, nil)

	percentage, err := svc.GetCustomersPerformanceSummary(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if percentage != 0.0 {
		t.Errorf("esperava 0%%, obteve %f%%", percentage)
	}
}

func TestGetCustomersPerformanceSummary_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCustomersPerformanceSummary(gomock.Any(), gomock.Any()).
		Return(db.GetCustomersPerformanceSummaryRow{}, errDatabase)

	_, err := svc.GetCustomersPerformanceSummary(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// ListCustomersPaginated
// ---------------------------------------------------------------------------

func TestListCustomersPaginated_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	pagination := globalDomain.PaginationParams{Page: 1, PerPage: 10}

	dbCustomers := []db.Customer{
		buildDbCustomer(uuid.New(), companyID, uuid.New()),
		buildDbCustomer(uuid.New(), companyID, uuid.New()),
	}

	repo.EXPECT().
		CountCustomersByCompany(gomock.Any(), pgconv.ParseUUIDToPgType(companyID)).
		Return(int64(2), nil)

	repo.EXPECT().
		ListCustomersPaginate(gomock.Any(), db.ListCustomersPaginateParams{
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
			Limit:     10,
			Offset:    0,
		}).
		Return(dbCustomers, nil)

	resp, err := svc.ListCustomersPaginated(context.Background(), companyID, pagination)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("esperava 2 clientes, obteve %d", len(resp.Data))
	}
	if resp.TotalRows != 2 {
		t.Errorf("esperava TotalRows=2, obteve %d", resp.TotalRows)
	}
	if resp.Page != 1 {
		t.Errorf("esperava Page=1, obteve %d", resp.Page)
	}
}

func TestListCustomersPaginated_SecondPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	pagination := globalDomain.PaginationParams{Page: 3, PerPage: 5}

	repo.EXPECT().
		CountCustomersByCompany(gomock.Any(), gomock.Any()).
		Return(int64(20), nil)

	repo.EXPECT().
		ListCustomersPaginate(gomock.Any(), db.ListCustomersPaginateParams{
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
			Limit:     5,
			Offset:    10, // (3-1) * 5 = 10
		}).
		Return([]db.Customer{
			buildDbCustomer(uuid.New(), companyID, uuid.New()),
		}, nil)

	resp, err := svc.ListCustomersPaginated(context.Background(), companyID, pagination)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Page != 3 {
		t.Errorf("esperava Page=3, obteve %d", resp.Page)
	}
	if resp.TotalPages != 4 { // ceil(20/5) = 4
		t.Errorf("esperava TotalPages=4, obteve %d", resp.TotalPages)
	}
}

func TestListCustomersPaginated_CountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		CountCustomersByCompany(gomock.Any(), gomock.Any()).
		Return(int64(0), errDatabase)

	_, err := svc.ListCustomersPaginated(context.Background(), uuid.New(), globalDomain.PaginationParams{Page: 1, PerPage: 10})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestListCustomersPaginated_ListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		CountCustomersByCompany(gomock.Any(), gomock.Any()).
		Return(int64(5), nil)

	repo.EXPECT().
		ListCustomersPaginate(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListCustomersPaginated(context.Background(), uuid.New(), globalDomain.PaginationParams{Page: 1, PerPage: 10})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestListCustomersPaginated_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		CountCustomersByCompany(gomock.Any(), gomock.Any()).
		Return(int64(0), nil)

	repo.EXPECT().
		ListCustomersPaginate(gomock.Any(), gomock.Any()).
		Return([]db.Customer{}, nil)

	resp, err := svc.ListCustomersPaginated(context.Background(), uuid.New(), globalDomain.PaginationParams{Page: 1, PerPage: 10})

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp.Data) != 0 {
		t.Errorf("esperava lista vazia, obteve %d", len(resp.Data))
	}
	if resp.TotalRows != 0 {
		t.Errorf("esperava TotalRows=0, obteve %d", resp.TotalRows)
	}
}

// ---------------------------------------------------------------------------
// Teste de conversão dos campos opcionais (pgtype.Text)
// ---------------------------------------------------------------------------

func TestGetCustomerById_NullableFieldsHandled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	id := uuid.New()

	// Cliente com campos opcionais nulos
	dbCustomer := db.Customer{
		ID:                  pgconv.ParseUUIDToPgType(id),
		CompanyID:           pgconv.ParseUUIDToPgType(uuid.New()),
		FullName:            "Cliente Mínimo",
		BirthDate:           pgtype.Date{Valid: false},
		Cpf:                 "999.888.777-66",
		Rg:                  pgtype.Text{Valid: false},
		MaritalStatus:       pgtype.Text{Valid: false},
		Gender:              "OTHER",
		Whatsapp:            pgtype.Text{Valid: false},
		MobilePhone:         pgtype.Text{Valid: false},
		HomePhone:           pgtype.Text{Valid: false},
		Email:               "minimo@email.com",
		AddressStreet:       pgtype.Text{Valid: false},
		AddressNumber:       pgtype.Text{Valid: false},
		AddressComplement:   pgtype.Text{Valid: false},
		AddressNeighborhood: pgtype.Text{Valid: false},
		AddressCity:         pgtype.Text{Valid: false},
		AddressState:        pgtype.Text{Valid: false},
		AddressZipcode:      pgtype.Text{Valid: false},
		AddressCountry:      pgtype.Text{Valid: false},
		BalanceDue:          pgconv.Float64ToPgNumeric(0),
		CreatedBy:           pgconv.ParseUUIDToPgType(uuid.New()),
		UpdatedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		DeletedBy:           pgconv.ParseUUIDToPgType(uuid.Nil),
		CreatedAt:           pgconv.TimeToPgTimestamptz(time.Now()),
		UpdatedAt:           pgtype.Timestamptz{Valid: false},
		DeletedAt:           pgtype.Timestamptz{Valid: false},
	}

	repo.EXPECT().
		GetCustomerById(gomock.Any(), gomock.Any()).
		Return(dbCustomer, nil)

	resp, err := svc.GetCustomerById(context.Background(), id)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	// Campos nulos devem virar string vazia
	if resp.Rg != "" {
		t.Errorf("Rg nulo deve ser string vazia, obteve '%s'", resp.Rg)
	}
	if resp.Whatsapp != "" {
		t.Errorf("Whatsapp nulo deve ser string vazia, obteve '%s'", resp.Whatsapp)
	}
	if resp.AddressCity != "" {
		t.Errorf("AddressCity nulo deve ser string vazia, obteve '%s'", resp.AddressCity)
	}
	if resp.BalanceDue != 0 {
		t.Errorf("BalanceDue deve ser 0, obteve %f", resp.BalanceDue)
	}
	if resp.Gender != enums.GenderOther {
		t.Errorf("Gender: esperava OTHER, obteve '%s'", resp.Gender)
	}
}
