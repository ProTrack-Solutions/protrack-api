// Package service: configuração fiscal da empresa (emitente na protrack-fiscal-api,
// A1, perfil fiscal, CSC), perfis tributários e tributação de produtos. Toda
// conversa com a API fiscal sai daqui (API a API); o front só fala com o ERP.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/client"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FiscalAPI é o que o serviço usa do cliente da API fiscal (fake nos testes).
type FiscalAPI interface {
	Configured(env client.Environment) bool
	CreateIssuer(ctx context.Context, env client.Environment, in client.CreateIssuerRequest) (client.IssuerPair, error)
	ListIssuers(ctx context.Context, env client.Environment) ([]client.Issuer, error)
	UploadCertificate(ctx context.Context, env client.Environment, issuerID, filename string, pfx []byte, password string) (client.CertificateUploaded, error)
	GetCertificate(ctx context.Context, env client.Environment, issuerID string) (client.CertificateStatus, error)
	PutFiscalProfile(ctx context.Context, env client.Environment, issuerID string, in client.FiscalProfileRequest) (client.FiscalProfile, error)
	GetFiscalProfile(ctx context.Context, env client.Environment, issuerID string) (client.FiscalProfile, error)
	SefazStatus(ctx context.Context, env client.Environment, issuerID string) (client.SefazStatus, error)
	CopyFromHomologation(ctx context.Context, liveIssuerID string, csc client.CSC) (client.CopyFromHomologationResult, error)
}

type Service struct {
	repo *repository.Repository
	pool *pgxpool.Pool
	api  FiscalAPI
	now  func() time.Time
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool, api FiscalAPI) *Service {
	return &Service{repo: repo, pool: pool, api: api, now: time.Now}
}

func (s *Service) inTx(ctx context.Context, fn func(repo *repository.Repository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(s.repo.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---------- Erros da API fiscal ----------

// APIError é uma recusa da API fiscal com texto para o lojista (o handler responde 422/502).
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return e.Message }

// translate converte erros do cliente em erros do domínio.
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, client.ErrNotConfigured) {
		return domain.ErrFiscalNotConfigured
	}
	if apiErr, ok := client.AsAPIError(err); ok {
		msg := apiErr.UserMessage()
		if apiErr.Hint != "" {
			msg += " (" + apiErr.Hint + ")"
		}
		if msg == "" {
			msg = fmt.Sprintf("a API fiscal recusou o pedido (%d)", apiErr.Status)
		}
		return &APIError{Status: apiErr.Status, Message: msg}
	}
	return &APIError{Status: 502, Message: "não foi possível falar com o serviço fiscal; tente de novo em instantes"}
}

// ---------- Configuração ----------

type settingsRow struct {
	db.CompanyFiscalSetting
	exists bool
}

func (s *Service) loadSettings(ctx context.Context, repo *repository.Repository, companyId uuid.UUID) (settingsRow, error) {
	row, err := repo.GetCompanyFiscalSettings(ctx, pgconv.ParseUUIDToPgType(companyId))
	if errors.Is(err, pgx.ErrNoRows) {
		return settingsRow{CompanyFiscalSetting: db.CompanyFiscalSetting{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			Environment: domain.EnvTest,
		}}, nil
	}
	if err != nil {
		return settingsRow{}, err
	}
	return settingsRow{CompanyFiscalSetting: row, exists: true}, nil
}

func saveSettings(ctx context.Context, repo *repository.Repository, row db.CompanyFiscalSetting, userId uuid.UUID) (db.CompanyFiscalSetting, error) {
	return repo.UpsertCompanyFiscalSettings(ctx, db.UpsertCompanyFiscalSettingsParams{
		CompanyID:            row.CompanyID,
		Enabled:              row.Enabled,
		Environment:          row.Environment,
		IssuerIDTest:         row.IssuerIDTest,
		IssuerIDLive:         row.IssuerIDLive,
		TaxRegime:            row.TaxRegime,
		DefaultTaxProfileID:  row.DefaultTaxProfileID,
		CertificateExpiresAt: row.CertificateExpiresAt,
		UpdatedBy:            pgconv.ParseUUIDToPgType(userId),
	})
}

// IssuerFor devolve o id do emitente no ambiente atual da empresa.
func IssuerFor(row db.CompanyFiscalSetting) (client.Environment, string, bool) {
	env := client.Environment(row.Environment)
	id := row.IssuerIDTest
	if env == client.EnvLive {
		id = row.IssuerIDLive
	}
	if !id.Valid {
		return env, "", false
	}
	return env, pgconv.PgUUIDToUUID(id).String(), true
}

func (s *Service) GetSettings(ctx context.Context, companyId uuid.UUID) (domain.FiscalSettings, error) {
	row, err := s.loadSettings(ctx, s.repo, companyId)
	if err != nil {
		return domain.FiscalSettings{}, err
	}
	company, err := s.repo.GetCompanyForFiscal(ctx, pgconv.ParseUUIDToPgType(companyId))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.FiscalSettings{}, domain.ErrCompanyNotFound
	}
	if err != nil {
		return domain.FiscalSettings{}, err
	}
	missing, err := s.repo.CountProductsMissingNcm(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.FiscalSettings{}, err
	}
	_, _, registered := IssuerFor(row.CompanyFiscalSetting)
	return domain.FiscalSettings{
		Enabled:             row.Enabled,
		Environment:         row.Environment,
		TaxRegime:           pgconv.ParsePgTextToStringPtr(row.TaxRegime),
		DefaultTaxProfileID: pgconv.PgUUIDToUUIDPtr(row.DefaultTaxProfileID),
		IssuerRegistered:    registered,
		CertificateExpires:  pgconv.PgTimestamptzToTimePtr(row.CertificateExpiresAt),
		ServerConfigured:    s.api.Configured(client.Environment(row.Environment)),
		ProductsMissingNCM:  missing,
		Company: domain.CompanyFiscalData{
			Name:              company.Name,
			TradeName:         pgconv.ParsePgTextToString(company.TradeName),
			CNPJ:              domain.OnlyDigits(pgconv.ParsePgTextToString(company.Document)),
			StateRegistration: domain.OnlyDigits(pgconv.ParsePgTextToString(company.InscricaoEstadual)),
			Phone:             domain.OnlyDigits(pgconv.ParsePgTextToString(company.Phone)),
			Street:            pgconv.ParsePgTextToString(company.AddressStreet),
			Number:            pgconv.ParsePgTextToString(company.AddressNumber),
			Complement:        pgconv.ParsePgTextToString(company.AddressComplement),
			District:          pgconv.ParsePgTextToString(company.AddressNeighborhood),
			City:              pgconv.ParsePgTextToString(company.AddressCity),
			State:             strings.ToUpper(pgconv.ParsePgTextToString(company.AddressState)),
			ZipCode:           domain.OnlyDigits(pgconv.ParsePgTextToString(company.AddressZipcode)),
		},
	}, nil
}

// UpdateSettings liga/desliga a emissão e escolhe o perfil tributário padrão.
// Ligar exige emitente, A1 válido, regime e perfil padrão.
func (s *Service) UpdateSettings(ctx context.Context, companyId, userId uuid.UUID, in domain.UpdateSettingsInput) (domain.FiscalSettings, error) {
	err := s.inTx(ctx, func(repo *repository.Repository) error {
		if err := repo.LockCompanyFiscalSettings(ctx, companyId.String()); err != nil {
			return err
		}
		row, err := s.loadSettings(ctx, repo, companyId)
		if err != nil {
			return err
		}
		if in.DefaultTaxProfileID != nil {
			if _, err := repo.GetTaxProfile(ctx, db.GetTaxProfileParams{
				ID: pgconv.ParseUUIDToPgType(*in.DefaultTaxProfileID), CompanyID: pgconv.ParseUUIDToPgType(companyId),
			}); errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrTaxProfileNotFound
			} else if err != nil {
				return err
			}
		}
		row.DefaultTaxProfileID = pgconv.OptionalPtrUUIDToPgType(in.DefaultTaxProfileID)
		if in.Enabled {
			_, _, registered := IssuerFor(row.CompanyFiscalSetting)
			var missing []string
			if !registered {
				missing = append(missing, "cadastrar a empresa como emitente")
			}
			if !row.CertificateExpiresAt.Valid {
				missing = append(missing, "enviar o certificado A1")
			} else if row.CertificateExpiresAt.Time.Before(s.now()) {
				missing = append(missing, "trocar o certificado A1 vencido")
			}
			if !row.TaxRegime.Valid {
				missing = append(missing, "salvar o perfil fiscal (IE, regime e endereço)")
			}
			if !row.DefaultTaxProfileID.Valid {
				missing = append(missing, "escolher o perfil tributário padrão")
			}
			if len(missing) > 0 {
				return fmt.Errorf("%w: falta %s", domain.ErrNotReadyToEnable, strings.Join(missing, ", "))
			}
		}
		row.Enabled = in.Enabled
		_, err = saveSettings(ctx, repo, row.CompanyFiscalSetting, userId)
		return err
	})
	if err != nil {
		return domain.FiscalSettings{}, err
	}
	return s.GetSettings(ctx, companyId)
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:max]))
}

// RegisterIssuer cadastra o CNPJ da empresa na API fiscal (homologação e produção).
func (s *Service) RegisterIssuer(ctx context.Context, companyId, userId uuid.UUID) (domain.FiscalSettings, error) {
	err := s.inTx(ctx, func(repo *repository.Repository) error {
		if err := repo.LockCompanyFiscalSettings(ctx, companyId.String()); err != nil {
			return err
		}
		row, err := s.loadSettings(ctx, repo, companyId)
		if err != nil {
			return err
		}
		if row.IssuerIDTest.Valid && row.IssuerIDLive.Valid {
			return domain.ErrIssuerAlreadyExists
		}
		company, err := repo.GetCompanyForFiscal(ctx, pgconv.ParseUUIDToPgType(companyId))
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrCompanyNotFound
		}
		if err != nil {
			return err
		}
		cnpj := domain.OnlyDigits(pgconv.ParsePgTextToString(company.Document))
		if len(cnpj) != 14 {
			return fmt.Errorf("%w: a NFC-e exige CNPJ; atualize o documento da empresa em Configurações", domain.ErrInvalidFiscalData)
		}
		if state := strings.ToUpper(pgconv.ParsePgTextToString(company.AddressState)); state != "" && state != "MG" {
			return fmt.Errorf("%w: por enquanto a emissão de NFC-e atende só empresas de MG", domain.ErrInvalidFiscalData)
		}
		req := client.CreateIssuerRequest{CNPJ: cnpj, LegalName: truncate(company.Name, 60), State: "MG"}
		if trade := truncate(pgconv.ParsePgTextToString(company.TradeName), 60); utf8.RuneCountInString(trade) >= 2 {
			req.TradeName = trade
		}
		pair, err := s.api.CreateIssuer(ctx, client.EnvTest, req)
		if apiErr, ok := client.AsAPIError(err); ok && apiErr.Status == 409 {
			// Já cadastrado numa tentativa anterior (resposta perdida): recupera os ids
			pair, err = s.findIssuerPair(ctx, cnpj)
		}
		if err != nil {
			return translate(err)
		}
		row.IssuerIDTest = pgconv.ParseUUIDToPgType(uuid.MustParse(pair.Test.ID))
		row.IssuerIDLive = pgconv.ParseUUIDToPgType(uuid.MustParse(pair.Live.ID))
		_, err = saveSettings(ctx, repo, row.CompanyFiscalSetting, userId)
		return err
	})
	if err != nil {
		return domain.FiscalSettings{}, err
	}
	return s.GetSettings(ctx, companyId)
}

func (s *Service) findIssuerPair(ctx context.Context, cnpj string) (client.IssuerPair, error) {
	var pair client.IssuerPair
	for _, env := range []client.Environment{client.EnvTest, client.EnvLive} {
		issuers, err := s.api.ListIssuers(ctx, env)
		if err != nil {
			return pair, err
		}
		for _, is := range issuers {
			if is.CNPJ != cnpj || is.DeactivatedAt != nil {
				continue
			}
			if is.Environment == 2 {
				pair.Test = is
			} else {
				pair.Live = is
			}
		}
	}
	if pair.Test.ID == "" || pair.Live.ID == "" {
		return pair, &APIError{Status: 409, Message: "esse CNPJ já está cadastrado no serviço fiscal por outra conta; fale com o suporte"}
	}
	if _, err := uuid.Parse(pair.Test.ID); err != nil {
		return pair, err
	}
	if _, err := uuid.Parse(pair.Live.ID); err != nil {
		return pair, err
	}
	return pair, nil
}

// currentIssuer lê a configuração e devolve ambiente e emitente atuais.
func (s *Service) currentIssuer(ctx context.Context, companyId uuid.UUID) (settingsRow, client.Environment, string, error) {
	row, err := s.loadSettings(ctx, s.repo, companyId)
	if err != nil {
		return row, "", "", err
	}
	env, issuer, ok := IssuerFor(row.CompanyFiscalSetting)
	if !ok {
		return row, env, "", domain.ErrIssuerNotRegistered
	}
	return row, env, issuer, nil
}

// UploadCertificate repassa o A1 para o emitente do ambiente atual (o ERP não guarda o arquivo).
func (s *Service) UploadCertificate(ctx context.Context, companyId, userId uuid.UUID, filename string, pfx []byte, password string) (domain.CertificateInfo, error) {
	_, env, issuer, err := s.currentIssuer(ctx, companyId)
	if err != nil {
		return domain.CertificateInfo{}, err
	}
	uploaded, err := s.api.UploadCertificate(ctx, env, issuer, filename, pfx, password)
	if err != nil {
		return domain.CertificateInfo{}, translate(err)
	}
	if err := s.storeCertificateExpiry(ctx, companyId, userId, uploaded.NotAfter); err != nil {
		return domain.CertificateInfo{}, err
	}
	return s.certificateInfo(uploaded.NotAfter), nil
}

func (s *Service) storeCertificateExpiry(ctx context.Context, companyId, userId uuid.UUID, expires time.Time) error {
	return s.inTx(ctx, func(repo *repository.Repository) error {
		if err := repo.LockCompanyFiscalSettings(ctx, companyId.String()); err != nil {
			return err
		}
		row, err := s.loadSettings(ctx, repo, companyId)
		if err != nil {
			return err
		}
		row.CertificateExpiresAt = pgconv.TimeToPgTimestamptz(expires)
		_, err = saveSettings(ctx, repo, row.CompanyFiscalSetting, userId)
		return err
	})
}

func (s *Service) certificateInfo(expires time.Time) domain.CertificateInfo {
	left := expires.Sub(s.now())
	days := int(left.Hours() / 24)
	return domain.CertificateInfo{ExpiresAt: expires, Expired: left <= 0, DaysToExpire: days}
}

// GetCertificate consulta a situação do A1 na API fiscal.
func (s *Service) GetCertificate(ctx context.Context, companyId uuid.UUID) (domain.CertificateInfo, error) {
	_, env, issuer, err := s.currentIssuer(ctx, companyId)
	if err != nil {
		return domain.CertificateInfo{}, err
	}
	status, err := s.api.GetCertificate(ctx, env, issuer)
	if err != nil {
		return domain.CertificateInfo{}, translate(err)
	}
	return s.certificateInfo(status.ExpiresAt), nil
}

func fromAPIProfile(p client.FiscalProfile) domain.FiscalProfile {
	out := domain.FiscalProfile{
		FiscalProfileInput: domain.FiscalProfileInput{
			StateRegistration: p.StateRegistration,
			TaxRegime:         p.TaxRegime,
			Street:            p.Address.Street,
			Number:            p.Address.Number,
			District:          p.Address.District,
			CityCode:          p.Address.CityCode,
			CityName:          p.Address.CityName,
			ZipCode:           p.Address.ZipCode,
		},
		CSCConfigured: p.CSC.Configured,
	}
	if p.Address.Complement != nil {
		out.Complement = *p.Address.Complement
	}
	if p.Address.Phone != nil {
		out.Phone = *p.Address.Phone
	}
	if p.CSC.ID != nil {
		out.CSCID = *p.CSC.ID
	}
	return out
}

// GetFiscalProfile lê o perfil fiscal na API fiscal; nil quando ainda não foi salvo.
func (s *Service) GetFiscalProfile(ctx context.Context, companyId uuid.UUID) (*domain.FiscalProfile, error) {
	_, env, issuer, err := s.currentIssuer(ctx, companyId)
	if err != nil {
		return nil, err
	}
	profile, err := s.api.GetFiscalProfile(ctx, env, issuer)
	if apiErr, ok := client.AsAPIError(err); ok && apiErr.Status == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, translate(err)
	}
	out := fromAPIProfile(profile)
	return &out, nil
}

// SaveFiscalProfile grava IE, regime, endereço e CSC na API fiscal e guarda o regime aqui.
func (s *Service) SaveFiscalProfile(ctx context.Context, companyId, userId uuid.UUID, in domain.FiscalProfileInput) (domain.FiscalProfile, error) {
	in, err := in.Normalize()
	if err != nil {
		return domain.FiscalProfile{}, err
	}
	_, env, issuer, err := s.currentIssuer(ctx, companyId)
	if err != nil {
		return domain.FiscalProfile{}, err
	}
	req := client.FiscalProfileRequest{
		StateRegistration: in.StateRegistration,
		TaxRegime:         in.TaxRegime,
		Address: client.Address{
			Street: in.Street, Number: in.Number, District: in.District,
			CityCode: in.CityCode, CityName: in.CityName, ZipCode: in.ZipCode,
		},
	}
	if in.Complement != "" {
		req.Address.Complement = &in.Complement
	}
	if in.Phone != "" {
		req.Address.Phone = &in.Phone
	}
	if in.CSCID != "" {
		req.CSC = &client.CSC{ID: in.CSCID, Secret: in.CSCSecret}
	}
	saved, err := s.api.PutFiscalProfile(ctx, env, issuer, req)
	if err != nil {
		return domain.FiscalProfile{}, translate(err)
	}
	err = s.inTx(ctx, func(repo *repository.Repository) error {
		if err := repo.LockCompanyFiscalSettings(ctx, companyId.String()); err != nil {
			return err
		}
		row, err := s.loadSettings(ctx, repo, companyId)
		if err != nil {
			return err
		}
		row.TaxRegime = pgconv.ParseStringToPgText(in.TaxRegime)
		_, err = saveSettings(ctx, repo, row.CompanyFiscalSetting, userId)
		return err
	})
	if err != nil {
		return domain.FiscalProfile{}, err
	}
	return fromAPIProfile(saved), nil
}

// SefazStatus testa a conexão com a SEFAZ usando o A1 do emitente (não emite nada).
func (s *Service) SefazStatus(ctx context.Context, companyId uuid.UUID) (domain.SefazStatus, error) {
	_, env, issuer, err := s.currentIssuer(ctx, companyId)
	if err != nil {
		return domain.SefazStatus{}, err
	}
	st, err := s.api.SefazStatus(ctx, env, issuer)
	if err != nil {
		return domain.SefazStatus{}, translate(err)
	}
	return domain.SefazStatus{
		Online: st.Online, StatusCode: st.StatusCode, Reason: st.Reason,
		LatencyMs: st.LatencyMs, CheckedAt: st.CheckedAt,
	}, nil
}

// GoLive copia A1 e perfil fiscal de homologação para produção (com o CSC de
// produção) e passa a empresa a emitir com valor fiscal.
func (s *Service) GoLive(ctx context.Context, companyId, userId uuid.UUID, in domain.GoLiveInput) (domain.FiscalSettings, error) {
	cscID := domain.OnlyDigits(in.CSCID)
	secret := strings.TrimSpace(in.CSCSecret)
	if cscID == "" || len(cscID) > 6 || secret == "" || len(secret) > 64 {
		return domain.FiscalSettings{}, fmt.Errorf("%w: informe o ID e o código do CSC de produção (gerados no portal da SEFAZ-MG)", domain.ErrInvalidFiscalData)
	}
	row, err := s.loadSettings(ctx, s.repo, companyId)
	if err != nil {
		return domain.FiscalSettings{}, err
	}
	if row.Environment == domain.EnvLive {
		return domain.FiscalSettings{}, domain.ErrAlreadyLive
	}
	if !row.IssuerIDLive.Valid {
		return domain.FiscalSettings{}, domain.ErrIssuerNotRegistered
	}
	copied, err := s.api.CopyFromHomologation(ctx, pgconv.PgUUIDToUUID(row.IssuerIDLive).String(), client.CSC{ID: cscID, Secret: secret})
	if err != nil {
		return domain.FiscalSettings{}, translate(err)
	}
	err = s.inTx(ctx, func(repo *repository.Repository) error {
		if err := repo.LockCompanyFiscalSettings(ctx, companyId.String()); err != nil {
			return err
		}
		row, err := s.loadSettings(ctx, repo, companyId)
		if err != nil {
			return err
		}
		row.Environment = domain.EnvLive
		if copied.Certificate != nil {
			row.CertificateExpiresAt = pgconv.TimeToPgTimestamptz(copied.Certificate.ExpiresAt)
		}
		_, err = saveSettings(ctx, repo, row.CompanyFiscalSetting, userId)
		return err
	})
	if err != nil {
		return domain.FiscalSettings{}, err
	}
	return s.GetSettings(ctx, companyId)
}

// ---------- Conversões ----------

// rateToNumeric guarda alíquotas com 4 casas (o pgconv padrão arredonda para 2).
func rateToNumeric(v *float64) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(fmt.Sprintf("%.4f", *v)); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

func numericToRate(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	v := pgconv.PgNumericToFloat64(n)
	return &v
}

func optText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
