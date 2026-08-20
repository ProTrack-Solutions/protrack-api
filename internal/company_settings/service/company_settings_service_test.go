package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/mocks"
	"github.com/ProTrack-Solutions/protrack-api/internal/company_settings/service"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/mock/gomock"
)

// ---------------------------------------------------------------------------
// Helpers e Fixtures
// ---------------------------------------------------------------------------

var errDatabase = errors.New("database error")

// newSvc cria um Service com o mock de repositório injetado.
func newSvc(t *testing.T, repo *mocks.MockRepositoryInterface) *service.Service {
	t.Helper()
	return service.NewService(repo)
}

// buildDbCompanySetting constrói um db.CompanySetting completo para uso nos testes.
func buildDbCompanySetting(id, companyID uuid.UUID, key string, value any) db.CompanySetting {
	raw, _ := json.Marshal(value)
	now := time.Now().UTC()

	return db.CompanySetting{
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: pgconv.ParseUUIDToPgType(companyID),
		Key:       key,
		Value:     raw,
		CreatedAt: pgconv.TimeToPgTimestamptz(now),
		UpdatedAt: pgconv.TimeToPgTimestamptz(now),
	}
}

// ---------------------------------------------------------------------------
// GetCompanySetting
// ---------------------------------------------------------------------------

func TestGetCompanySetting_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	setting := buildDbCompanySetting(uuid.New(), companyID, string(enums.IsWhatsappActive), true)

	repo.EXPECT().
		GetCompanySetting(gomock.Any(), db.GetCompanySettingParams{
			CompanyID: pgconv.OptionalUUIDToPgType(companyID),
			Key:       string(enums.IsWhatsappActive),
		}).
		Return(setting, nil)

	resp, err := svc.GetCompanySetting(context.Background(), companyID, enums.IsWhatsappActive)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.Key != enums.IsWhatsappActive {
		t.Errorf("Key incorreta: '%s'", resp.Key)
	}
	if resp.Value != true {
		t.Errorf("Value incorreto: %v", resp.Value)
	}
	if resp.CompanyID != companyID {
		t.Errorf("CompanyID incorreto")
	}
}

func TestGetCompanySetting_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		GetCompanySetting(gomock.Any(), gomock.Any()).
		Return(db.CompanySetting{}, errDatabase)

	_, err := svc.GetCompanySetting(context.Background(), uuid.New(), enums.LowStock)

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestGetCompanySetting_InvalidJSONValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	setting := buildDbCompanySetting(uuid.New(), companyID, string(enums.LowStock), 2)
	setting.Value = []byte(`{invalido`)

	repo.EXPECT().
		GetCompanySetting(gomock.Any(), gomock.Any()).
		Return(setting, nil)

	_, err := svc.GetCompanySetting(context.Background(), companyID, enums.LowStock)

	if err == nil {
		t.Fatal("esperava erro de decodificação do value")
	}
}

// ---------------------------------------------------------------------------
// ListCompanySettings
// ---------------------------------------------------------------------------

func TestListCompanySettings_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	settings := []db.CompanySetting{
		buildDbCompanySetting(uuid.New(), companyID, string(enums.IsWhatsappActive), true),
		buildDbCompanySetting(uuid.New(), companyID, string(enums.LowStock), 2),
		buildDbCompanySetting(uuid.New(), companyID, string(enums.NormalStock), 5),
	}

	repo.EXPECT().
		ListCompanySettings(gomock.Any(), pgconv.OptionalUUIDToPgType(companyID)).
		Return(settings, nil)

	resp, err := svc.ListCompanySettings(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 3 {
		t.Errorf("esperava 3 configs, obteve %d", len(resp))
	}
}

func TestListCompanySettings_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListCompanySettings(gomock.Any(), gomock.Any()).
		Return([]db.CompanySetting{}, nil)

	resp, err := svc.ListCompanySettings(context.Background(), uuid.New())

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("esperava lista vazia, obteve %d", len(resp))
	}
}

func TestListCompanySettings_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		ListCompanySettings(gomock.Any(), gomock.Any()).
		Return(nil, errDatabase)

	_, err := svc.ListCompanySettings(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

func TestListCompanySettings_InvalidJSONValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	setting := buildDbCompanySetting(uuid.New(), companyID, string(enums.LowStock), 2)
	setting.Value = []byte(`{invalido`)

	repo.EXPECT().
		ListCompanySettings(gomock.Any(), gomock.Any()).
		Return([]db.CompanySetting{setting}, nil)

	_, err := svc.ListCompanySettings(context.Background(), companyID)

	if err == nil {
		t.Fatal("esperava erro de decodificação do value")
	}
}

// ---------------------------------------------------------------------------
// UpsertCompanySetting
// ---------------------------------------------------------------------------

func TestUpsertCompanySetting_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	settingID := uuid.New()
	now := time.Now().UTC()

	req := domain.UpsertCompanySettingRequest{
		Key:   enums.LowStock,
		Value: 3,
	}

	rawValue, _ := json.Marshal(req.Value)

	repo.EXPECT().
		UpsertCompanySetting(gomock.Any(), db.UpsertCompanySettingParams{
			CompanyID: pgconv.OptionalUUIDToPgType(companyID),
			Key:       string(enums.LowStock),
			Value:     rawValue,
		}).
		Return(db.CompanySetting{
			ID:        pgconv.ParseUUIDToPgType(settingID),
			CompanyID: pgconv.ParseUUIDToPgType(companyID),
			Key:       string(enums.LowStock),
			Value:     rawValue,
			CreatedAt: pgconv.TimeToPgTimestamptz(now),
			UpdatedAt: pgconv.TimeToPgTimestamptz(now),
		}, nil)

	resp, err := svc.UpsertCompanySetting(context.Background(), companyID, req)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if resp.ID != settingID {
		t.Errorf("ID incorreto")
	}
	if !resp.CreatedAt.Equal(resp.UpdatedAt) {
		t.Errorf("esperava CreatedAt == UpdatedAt na criação")
	}
}

func TestUpsertCompanySetting_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		UpsertCompanySetting(gomock.Any(), gomock.Any()).
		Return(db.CompanySetting{}, errDatabase)

	_, err := svc.UpsertCompanySetting(context.Background(), uuid.New(), domain.UpsertCompanySettingRequest{
		Key:   enums.LowStock,
		Value: 3,
	})

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// DeleteCompanySetting
// ---------------------------------------------------------------------------

func TestDeleteCompanySetting_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		DeleteCompanySetting(gomock.Any(), db.DeleteCompanySettingParams{
			CompanyID: pgconv.OptionalUUIDToPgType(companyID),
			Key:       string(enums.LowStock),
		}).
		Return(nil)

	err := svc.DeleteCompanySetting(context.Background(), companyID, string(enums.LowStock))

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestDeleteCompanySetting_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		DeleteCompanySetting(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.DeleteCompanySetting(context.Background(), uuid.New(), string(enums.LowStock))

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// DeleteAllCompanySettings
// ---------------------------------------------------------------------------

func TestDeleteAllCompanySettings_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	repo.EXPECT().
		DeleteAllCompanySettings(gomock.Any(), pgconv.OptionalUUIDToPgType(companyID)).
		Return(nil)

	err := svc.DeleteAllCompanySettings(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestDeleteAllCompanySettings_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	repo.EXPECT().
		DeleteAllCompanySettings(gomock.Any(), gomock.Any()).
		Return(errDatabase)

	err := svc.DeleteAllCompanySettings(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// RestorDefaultSettings
// ---------------------------------------------------------------------------

func TestRestorDefaultSettings_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()

	// domain.DefaultSettings tem 5 chaves — cada uma deve gerar um upsert.
	repo.EXPECT().
		UpsertCompanySetting(gomock.Any(), gomock.Any()).
		Return(db.CompanySetting{}, nil).
		Times(len(domain.DefaultSettings))

	err := svc.RestorDefaultSettings(context.Background(), companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
}

func TestRestorDefaultSettings_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	// Basta uma das chamadas falhar para o método retornar erro.
	repo.EXPECT().
		UpsertCompanySetting(gomock.Any(), gomock.Any()).
		Return(db.CompanySetting{}, errDatabase).
		AnyTimes()

	err := svc.RestorDefaultSettings(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("esperava erro do repositório")
	}
}

// ---------------------------------------------------------------------------
// SetDefaultSettings
// ---------------------------------------------------------------------------

// TestSetDefaultSettings_Success garante que SetDefaultSettings usa a conexão/tx
// recebida (via db.New(tx)) para persistir todas as configs default, sem passar
// pelo repositório injetado no Service.
func TestSetDefaultSettings_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	companyID := uuid.New()
	tx := &fakeDBTX{}

	err := svc.SetDefaultSettings(context.Background(), tx, companyID)

	if err != nil {
		t.Fatalf("esperava nil, obteve: %v", err)
	}
	if tx.execCalls != len(domain.DefaultSettings) {
		t.Errorf("esperava %d chamadas de upsert via tx, obteve %d", len(domain.DefaultSettings), tx.execCalls)
	}
}

func TestSetDefaultSettings_TxError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepositoryInterface(ctrl)
	svc := newSvc(t, repo)

	tx := &fakeDBTX{err: errDatabase}

	err := svc.SetDefaultSettings(context.Background(), tx, uuid.New())

	if err == nil {
		t.Fatal("esperava erro da transação")
	}
}

// ---------------------------------------------------------------------------
// fakeDBTX — implementação mínima de db.DBTX para testar SetDefaultSettings,
// que constrói suas próprias Queries a partir da tx recebida (db.New(tx)) em
// vez de usar o repositório injetado.
// ---------------------------------------------------------------------------

type fakeDBTX struct {
	execCalls int
	err       error
}

func (f *fakeDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, f.err
}

func (f *fakeDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, f.err
}

func (f *fakeDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.execCalls++
	return &fakeRow{err: f.err}
}

type fakeRow struct {
	err error
}

func (r *fakeRow) Scan(dest ...any) error {
	return r.err
}
