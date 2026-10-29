package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_register/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_register/repository"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *repository.Repository
	pool *pgxpool.Pool
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool) *Service {
	return &Service{
		repo: repo,
		pool: pool,
	}
}

const (
	defaultPerPage = 20
	// Código do Postgres para violação de unique (caixa aberto em duplicidade)
	uniqueViolation = "23505"
)

func round(v float64) float64 {
	return math.Round(v*100) / 100
}

// ---------------------------------------------------------------------------
// Caixas cadastrados
// ---------------------------------------------------------------------------

// ListRegisters devolve os caixas da empresa com o turno aberto de cada um.
// Empresa sem nenhum caixa ganha o "Caixa principal" na primeira consulta.
func (s *Service) ListRegisters(ctx context.Context, companyId uuid.UUID) ([]domain.CashRegister, error) {
	company := pgconv.ParseUUIDToPgType(companyId)
	if err := s.repo.EnsureDefaultCashRegister(ctx, company); err != nil {
		return nil, err
	}

	rows, err := s.repo.ListCashRegisters(ctx, company)
	if err != nil {
		return nil, err
	}

	registers := make([]domain.CashRegister, 0, len(rows))
	for _, r := range rows {
		register := domain.CashRegister{
			ID:     pgconv.PgUUIDToUUID(r.ID),
			Name:   r.Name,
			Active: r.Active,
		}
		if r.SessionID.Valid {
			register.OpenSession = &domain.OpenRegisterSession{
				ID:            pgconv.PgUUIDToUUID(r.SessionID),
				OpenedAt:      pgconv.PgTimestamptzToTime(r.OpenedAt),
				OpeningAmount: pgconv.PgNumericToFloat64(r.OpeningAmount),
				OpenedByName:  r.OpenedByName,
			}
		}
		if r.LastClosedAt.Valid {
			register.LastClosing = &domain.LastClosing{
				ClosedAt:     pgconv.PgTimestamptzToTime(r.LastClosedAt),
				ClosedByName: r.LastClosedByName,
				CountedCash:  pgconv.PgNumericToFloat64(r.LastCountedCash),
			}
		}
		registers = append(registers, register)
	}
	return registers, nil
}

func (s *Service) CreateRegister(ctx context.Context, companyId uuid.UUID, req domain.SaveCashRegisterRequest) (domain.CashRegister, error) {
	r, err := s.repo.CreateCashRegister(ctx, db.CreateCashRegisterParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Name:      strings.TrimSpace(req.Name),
	})
	if err != nil {
		return domain.CashRegister{}, err
	}
	return domain.CashRegister{ID: pgconv.PgUUIDToUUID(r.ID), Name: r.Name, Active: r.Active}, nil
}

// UpdateRegister renomeia ou ativa/desativa um caixa. Não desativa caixa com turno aberto.
func (s *Service) UpdateRegister(ctx context.Context, companyId, id uuid.UUID, req domain.SaveCashRegisterRequest) (domain.CashRegister, error) {
	company := pgconv.ParseUUIDToPgType(companyId)
	current, err := s.repo.GetCashRegisterById(ctx, db.GetCashRegisterByIdParams{
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: company,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CashRegister{}, domain.ErrCashRegisterNotFound
	}
	if err != nil {
		return domain.CashRegister{}, err
	}

	active := current.Active
	if req.Active != nil {
		active = *req.Active
	}

	if !active && current.Active {
		_, err := s.repo.GetOpenCashSessionByRegister(ctx, current.ID)
		if err == nil {
			return domain.CashRegister{}, domain.ErrCashRegisterInUse
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return domain.CashRegister{}, err
		}
	}

	r, err := s.repo.UpdateCashRegister(ctx, db.UpdateCashRegisterParams{
		ID:        current.ID,
		CompanyID: company,
		Name:      strings.TrimSpace(req.Name),
		Active:    active,
	})
	if err != nil {
		return domain.CashRegister{}, err
	}
	return domain.CashRegister{ID: pgconv.PgUUIDToUUID(r.ID), Name: r.Name, Active: r.Active}, nil
}

// resolveRegister descobre em qual caixa o computador trabalha: o do header
// CashRegisterId ou, sem header, o único caixa ativo da empresa.
func resolveRegister(ctx context.Context, q *db.Queries, companyId uuid.UUID) (pgtype.UUID, error) {
	company := pgconv.ParseUUIDToPgType(companyId)

	if id := domain.RegisterIDFrom(ctx); id != uuid.Nil {
		r, err := q.GetCashRegisterById(ctx, db.GetCashRegisterByIdParams{
			ID:        pgconv.ParseUUIDToPgType(id),
			CompanyID: company,
		})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && !r.Active) {
			return pgtype.UUID{}, domain.ErrCashRegisterNotFound
		}
		return r.ID, err
	}

	if err := q.EnsureDefaultCashRegister(ctx, company); err != nil {
		return pgtype.UUID{}, err
	}
	ids, err := q.ListActiveCashRegisterIds(ctx, company)
	if err != nil {
		return pgtype.UUID{}, err
	}
	switch len(ids) {
	case 1:
		return ids[0], nil
	case 0:
		return pgtype.UUID{}, domain.ErrCashRegisterNotFound
	default:
		return pgtype.UUID{}, domain.ErrCashRegisterNotSelected
	}
}

// ---------------------------------------------------------------------------
// Turno do caixa
// ---------------------------------------------------------------------------

// Open abre o turno do caixa do computador. Quem abre fica registrado, mas o
// turno é do caixa: qualquer operador vende, movimenta e fecha.
func (s *Service) Open(ctx context.Context, companyId, userId uuid.UUID, req domain.OpenCashSessionRequest) (domain.CashSessionDetail, error) {
	registerId, err := resolveRegister(ctx, db.New(s.pool), companyId)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	session, err := s.repo.CreateCashSession(ctx, db.CreateCashSessionParams{
		CompanyID:     pgconv.ParseUUIDToPgType(companyId),
		RegisterID:    registerId,
		UserID:        pgconv.ParseUUIDToPgType(userId),
		OpeningAmount: pgconv.Float64ToPgNumeric(round(req.OpeningAmount)),
		OpeningNotes:  pgconv.ParseStringToPgText(req.Notes),
	})
	if err != nil {
		// O índice "um turno aberto por caixa" barra a segunda abertura
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.CashSessionDetail{}, domain.ErrCashSessionAlreadyOpen
		}
		return domain.CashSessionDetail{}, err
	}

	return s.GetById(ctx, companyId, userId, true, pgconv.PgUUIDToUUID(session.ID))
}

// Current devolve o turno aberto do caixa do computador, ou nil quando ele está fechado.
func (s *Service) Current(ctx context.Context, companyId, userId uuid.UUID) (*domain.CashSessionDetail, error) {
	registerId, err := resolveRegister(ctx, db.New(s.pool), companyId)
	if err != nil {
		return nil, err
	}

	session, err := s.repo.GetOpenCashSessionByRegister(ctx, registerId)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	detail, err := s.GetById(ctx, companyId, userId, true, pgconv.PgUUIDToUUID(session.ID))
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func (s *Service) AddMovement(ctx context.Context, companyId, userId uuid.UUID, req domain.CreateMovementRequest) (domain.CashSessionDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	defer tx.Rollback(ctx)

	repo := s.repo.WithTx(tx)

	open, err := lockOpenSession(ctx, tx, repo, companyId)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	session := toSession(open)

	amount := round(req.Amount)

	// Não dá para tirar da gaveta mais dinheiro do que deveria haver nela
	if req.Type == domain.MovementWithdrawal {
		summary, err := s.summary(ctx, repo, session)
		if err != nil {
			return domain.CashSessionDetail{}, err
		}
		if amount > summary.ExpectedCash {
			return domain.CashSessionDetail{}, domain.ErrInsufficientCash
		}
	}

	if _, err := repo.CreateCashSessionMovement(ctx, db.CreateCashSessionMovementParams{
		SessionID: open.ID,
		CompanyID: open.CompanyID,
		Type:      db.CashMovementType(req.Type),
		Amount:    pgconv.Float64ToPgNumeric(amount),
		Reason:    pgconv.ParseStringToPgText(req.Reason),
		CreatedBy: pgconv.ParseUUIDToPgType(userId),
	}); err != nil {
		return domain.CashSessionDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.CashSessionDetail{}, err
	}
	return s.GetById(ctx, companyId, userId, true, session.ID)
}

// Close fecha o turno do caixa do computador; qualquer operador pode fechar.
func (s *Service) Close(ctx context.Context, companyId, userId uuid.UUID, req domain.CloseCashSessionRequest) (domain.CashSessionDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	defer tx.Rollback(ctx)

	repo := s.repo.WithTx(tx)

	open, err := lockOpenSession(ctx, tx, repo, companyId)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	// Troco "em trânsito" deixaria a gaveta de um dos caixas sem fechar
	pending, err := repo.CountPendingCashTransfers(ctx, open.ID)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	if pending > 0 {
		return domain.CashSessionDetail{}, domain.ErrPendingTransfers
	}

	summary, err := s.summary(ctx, repo, toSession(open))
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	counted := round(req.CountedCash)

	if _, err := repo.CloseCashSession(ctx, db.CloseCashSessionParams{
		ID:               open.ID,
		ClosedBy:         pgconv.ParseUUIDToPgType(userId),
		ClosingNotes:     pgconv.ParseStringToPgText(req.Notes),
		SalesTotal:       pgconv.Float64ToPgNumeric(summary.SalesTotal),
		SalesCount:       pgconv.IntToPgInt4(int(summary.SalesCount)),
		ReceivedTotal:    pgconv.Float64ToPgNumeric(summary.ReceivedTotal),
		SuppliesTotal:    pgconv.Float64ToPgNumeric(summary.SuppliesTotal),
		WithdrawalsTotal: pgconv.Float64ToPgNumeric(summary.WithdrawalsTotal),
		ExpectedCash:     pgconv.Float64ToPgNumeric(summary.ExpectedCash),
		CountedCash:      pgconv.Float64ToPgNumeric(counted),
		Difference:       pgconv.Float64ToPgNumeric(round(counted - summary.ExpectedCash)),
	}); err != nil {
		return domain.CashSessionDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.CashSessionDetail{}, err
	}
	return s.GetById(ctx, companyId, userId, true, pgconv.PgUUIDToUUID(open.ID))
}

// List mostra o histórico de turnos. O administrador vê todos; o operador, os
// turnos de que participou (abriu, fechou, vendeu ou recebeu).
func (s *Service) List(ctx context.Context, companyId, userId uuid.UUID, isAdmin bool, params domain.ListCashSessionsParams) (domain.ListCashSessionsResponse, error) {
	page := max(params.Page, 1)
	perPage := params.PerPage
	if perPage == 0 {
		perPage = defaultPerPage
	}

	filter := pgtype.UUID{}
	if !isAdmin {
		filter = pgconv.ParseUUIDToPgType(userId)
	}
	register := pgtype.UUID{}
	if params.RegisterID != uuid.Nil {
		register = pgconv.ParseUUIDToPgType(params.RegisterID)
	}

	rows, err := s.repo.ListCashSessions(ctx, db.ListCashSessionsParams{
		CompanyID:  pgconv.ParseUUIDToPgType(companyId),
		RegisterID: register,
		UserID:     filter,
		RowLimit:   perPage,
		RowOffset:  (page - 1) * perPage,
	})
	if err != nil {
		return domain.ListCashSessionsResponse{}, err
	}

	resp := domain.ListCashSessionsResponse{
		Data:    make([]domain.CashSession, 0, len(rows)),
		Page:    page,
		PerPage: perPage,
	}
	for _, r := range rows {
		resp.TotalRows = r.TotalRows
		session := toSession(db.CashSession{
			ID:               r.ID,
			CompanyID:        r.CompanyID,
			RegisterID:       r.RegisterID,
			UserID:           r.UserID,
			Status:           r.Status,
			OpeningAmount:    r.OpeningAmount,
			OpeningNotes:     r.OpeningNotes,
			OpenedAt:         r.OpenedAt,
			ClosedAt:         r.ClosedAt,
			ClosedBy:         r.ClosedBy,
			ClosingNotes:     r.ClosingNotes,
			SalesTotal:       r.SalesTotal,
			SalesCount:       r.SalesCount,
			ReceivedTotal:    r.ReceivedTotal,
			SuppliesTotal:    r.SuppliesTotal,
			WithdrawalsTotal: r.WithdrawalsTotal,
			ExpectedCash:     r.ExpectedCash,
			CountedCash:      r.CountedCash,
			Difference:       r.Difference,
		})
		session.UserName = r.UserName
		session.RegisterName = r.RegisterName
		session.ClosedByName = r.ClosedByName
		resp.Data = append(resp.Data, session)
	}
	resp.TotalPages = (resp.TotalRows + int64(perPage) - 1) / int64(perPage)

	return resp, nil
}

func (s *Service) GetById(ctx context.Context, companyId, userId uuid.UUID, isAdmin bool, id uuid.UUID) (domain.CashSessionDetail, error) {
	r, err := s.repo.GetCashSessionById(ctx, db.GetCashSessionByIdParams{
		ID:        pgconv.ParseUUIDToPgType(id),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CashSessionDetail{}, domain.ErrCashSessionNotFound
	}
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	// Operador comum só enxerga os turnos de que participou
	if !isAdmin {
		tookPart, err := s.repo.UserTookPartInCashSession(ctx, db.UserTookPartInCashSessionParams{
			SessionID: r.ID,
			UserID:    pgconv.ParseUUIDToPgType(userId),
		})
		if err != nil {
			return domain.CashSessionDetail{}, err
		}
		if !tookPart {
			return domain.CashSessionDetail{}, domain.ErrCashSessionNotFound
		}
	}

	session := toSession(db.CashSession{
		ID:               r.ID,
		CompanyID:        r.CompanyID,
		RegisterID:       r.RegisterID,
		UserID:           r.UserID,
		Status:           r.Status,
		OpeningAmount:    r.OpeningAmount,
		OpeningNotes:     r.OpeningNotes,
		OpenedAt:         r.OpenedAt,
		ClosedAt:         r.ClosedAt,
		ClosedBy:         r.ClosedBy,
		ClosingNotes:     r.ClosingNotes,
		SalesTotal:       r.SalesTotal,
		SalesCount:       r.SalesCount,
		ReceivedTotal:    r.ReceivedTotal,
		SuppliesTotal:    r.SuppliesTotal,
		WithdrawalsTotal: r.WithdrawalsTotal,
		ExpectedCash:     r.ExpectedCash,
		CountedCash:      r.CountedCash,
		Difference:       r.Difference,
	})
	session.UserName = r.UserName
	session.RegisterName = r.RegisterName
	session.ClosedByName = r.ClosedByName

	return s.detail(ctx, s.repo, session)
}

// BindOpenSession liga a operação (venda, recebimento de parcela, devolução) ao
// turno aberto do caixa do computador. Grava o turno na variável da transação,
// que é o valor padrão de cash_session_id nas tabelas (migração 000066), e trava
// o turno para leitura até o commit, para o fechamento não passar no meio.
//
// Sem turno aberto a operação segue sem caixa, a menos que a empresa ligue
// require_open_cash_register. Sem header e com mais de um caixa aberto não dá
// para adivinhar a gaveta: aí o computador precisa escolher o caixa.
func BindOpenSession(ctx context.Context, tx db.DBTX, companyId uuid.UUID) error {
	q := db.New(tx)

	required, err := openSessionRequired(ctx, q, companyId)
	if err != nil {
		return err
	}

	var sessionId pgtype.UUID
	if id := domain.RegisterIDFrom(ctx); id != uuid.Nil {
		sessionId, err = q.GetOpenCashSessionByRegisterForShare(ctx, pgconv.ParseUUIDToPgType(id))
		if errors.Is(err, pgx.ErrNoRows) {
			err = nil
		}
		if err != nil {
			return err
		}
	} else {
		open, err := q.ListOpenCashSessionIdsForShare(ctx, pgconv.ParseUUIDToPgType(companyId))
		if err != nil {
			return err
		}
		if len(open) > 1 {
			return domain.ErrCashRegisterNotSelected
		}
		if len(open) == 1 {
			sessionId = open[0]
		}
	}

	if !sessionId.Valid {
		if required {
			return domain.ErrCashRegisterRequired
		}
		return nil
	}

	return q.BindCashSession(ctx, pgconv.PgUUIDToUUID(sessionId).String())
}

// openSessionRequired lê a configuração require_open_cash_register; empresas sem
// a configuração gravada seguem sem a exigência.
func openSessionRequired(ctx context.Context, q *db.Queries, companyId uuid.UUID) (bool, error) {
	setting, err := q.GetCompanySetting(ctx, db.GetCompanySettingParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Key:       string(enums.RequireOpenCashRegister),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var required bool
	if err := json.Unmarshal([]byte(setting.Value), &required); err != nil {
		return false, err
	}
	return required, nil
}

func lockOpenSession(ctx context.Context, tx db.DBTX, repo *repository.Repository, companyId uuid.UUID) (db.CashSession, error) {
	registerId, err := resolveRegister(ctx, db.New(tx), companyId)
	if err != nil {
		return db.CashSession{}, err
	}

	open, err := repo.GetOpenCashSessionByRegisterForUpdate(ctx, registerId)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.CashSession{}, domain.ErrNoOpenCashSession
	}
	return open, err
}

func (s *Service) detail(ctx context.Context, repo *repository.Repository, session domain.CashSession) (domain.CashSessionDetail, error) {
	summary, err := s.summary(ctx, repo, session)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	rows, err := repo.ListCashSessionMovements(ctx, pgconv.ParseUUIDToPgType(session.ID))
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	movements := make([]domain.CashMovement, 0, len(rows))
	for _, m := range rows {
		movements = append(movements, domain.CashMovement{
			ID:            pgconv.PgUUIDToUUID(m.ID),
			Type:          string(m.Type),
			Amount:        pgconv.PgNumericToFloat64(m.Amount),
			Reason:        pgconv.ParsePgTextToStringPtr(m.Reason),
			CreatedByName: m.CreatedByName,
			TransferLabel: m.TransferLabel,
			CreatedAt:     pgconv.PgTimestamptzToTime(m.CreatedAt),
		})
	}

	transfers, err := repo.ListPendingCashTransfers(ctx, pgconv.ParseUUIDToPgType(session.ID))
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	incoming := make([]domain.PendingTransfer, 0)
	outgoing := make([]domain.PendingTransfer, 0)
	for _, t := range transfers {
		if t.Incoming {
			incoming = append(incoming, toPendingTransfer(t))
		} else {
			outgoing = append(outgoing, toPendingTransfer(t))
		}
	}

	return domain.CashSessionDetail{
		Session:           session,
		Summary:           summary,
		Movements:         movements,
		IncomingTransfers: incoming,
		OutgoingTransfers: outgoing,
	}, nil
}

// summary soma o turno: vendas, parcelas recebidas e reembolsos gravados com o
// cash_session_id dele, mais sangrias e suprimentos.
func (s *Service) summary(ctx context.Context, repo *repository.Repository, session domain.CashSession) (domain.CashSummary, error) {
	sessionId := pgconv.ParseUUIDToPgType(session.ID)

	sales, err := repo.GetCashSessionSalesByMethod(ctx, sessionId)
	if err != nil {
		return domain.CashSummary{}, err
	}

	receipts, err := repo.GetCashSessionReceiptsByMethod(ctx, sessionId)
	if err != nil {
		return domain.CashSummary{}, err
	}

	refunds, err := repo.GetCashSessionRefundsByMethod(ctx, sessionId)
	if err != nil {
		return domain.CashSummary{}, err
	}

	operators, err := repo.GetCashSessionByOperator(ctx, sessionId)
	if err != nil {
		return domain.CashSummary{}, err
	}

	movements, err := repo.ListCashSessionMovements(ctx, sessionId)
	if err != nil {
		return domain.CashSummary{}, err
	}

	methods := map[string]*domain.MethodSummary{}
	get := func(method string) *domain.MethodSummary {
		if m, ok := methods[method]; ok {
			return m
		}
		m := &domain.MethodSummary{PaymentMethod: method}
		methods[method] = m
		return m
	}

	var sum domain.CashSummary

	for _, r := range sales {
		m := get(r.PaymentMethod)
		m.SalesCount += r.SalesCount
		m.SalesTotal += r.SalesTotal
		m.Received += r.Received

		sum.SalesCount += r.SalesCount
		sum.SalesTotal += r.SalesTotal
		sum.ReceivedTotal += r.Received
		sum.ChangeCount += r.ChangeCount
		sum.ChangeTotal += r.ChangeTotal

		// A entrada da venda a prazo é paga no balcão e conta como dinheiro na gaveta
		if r.PaymentMethod == "cash" || r.PaymentMethod == "installments" {
			sum.CashReceived += r.Received
		}
	}

	for _, r := range receipts {
		m := get(r.PaymentMethod)
		m.ReceiptsCount += r.ReceiptsCount
		m.ReceiptsTotal += r.Received
		m.Received += r.Received

		sum.ReceiptsCount += r.ReceiptsCount
		sum.ReceiptsTotal += r.Received
		sum.ReceivedTotal += r.Received

		if r.PaymentMethod == "cash" {
			sum.CashReceived += r.Received
		}
	}

	// Reembolso de troca/devolução sai do que entrou naquela forma de pagamento
	for _, r := range refunds {
		m := get(r.PaymentMethod)
		m.RefundsCount += r.RefundsCount
		m.RefundsTotal += r.Refunded
		m.Received -= r.Refunded

		sum.RefundsCount += r.RefundsCount
		sum.RefundsTotal += r.Refunded
		sum.ReceivedTotal -= r.Refunded

		if r.PaymentMethod == "cash" {
			sum.CashRefunds += r.Refunded
		}
	}

	for _, m := range movements {
		amount := pgconv.PgNumericToFloat64(m.Amount)
		if m.Type == db.CashMovementTypeSupply {
			sum.SuppliesTotal += amount
		} else {
			sum.WithdrawalsTotal += amount
		}
	}

	sum.ByMethod = make([]domain.MethodSummary, 0, len(methods))
	for _, m := range methods {
		m.SalesTotal = round(m.SalesTotal)
		m.ReceiptsTotal = round(m.ReceiptsTotal)
		m.Received = round(m.Received)
		m.RefundsTotal = round(m.RefundsTotal)
		sum.ByMethod = append(sum.ByMethod, *m)
	}
	sort.Slice(sum.ByMethod, func(i, j int) bool {
		return sum.ByMethod[i].Received > sum.ByMethod[j].Received
	})

	sum.ByOperator = make([]domain.OperatorSummary, 0, len(operators))
	for _, o := range operators {
		sum.ByOperator = append(sum.ByOperator, domain.OperatorSummary{
			UserID:        pgconv.PgUUIDToUUID(o.UserID),
			UserName:      o.UserName,
			SalesCount:    o.SalesCount,
			SalesTotal:    round(o.SalesTotal),
			ReceiptsTotal: round(o.ReceiptsTotal),
		})
	}

	sum.SalesTotal = round(sum.SalesTotal)
	sum.ReceiptsTotal = round(sum.ReceiptsTotal)
	sum.ReceivedTotal = round(sum.ReceivedTotal)
	sum.CashReceived = round(sum.CashReceived)
	sum.ChangeTotal = round(sum.ChangeTotal)
	sum.SuppliesTotal = round(sum.SuppliesTotal)
	sum.WithdrawalsTotal = round(sum.WithdrawalsTotal)
	sum.RefundsTotal = round(sum.RefundsTotal)
	sum.CashRefunds = round(sum.CashRefunds)
	sum.ExpectedCash = round(session.OpeningAmount + sum.CashReceived + sum.SuppliesTotal - sum.WithdrawalsTotal - sum.CashRefunds)

	return sum, nil
}

func toSession(r db.CashSession) domain.CashSession {
	return domain.CashSession{
		ID:            pgconv.PgUUIDToUUID(r.ID),
		CompanyID:     pgconv.PgUUIDToUUID(r.CompanyID),
		RegisterID:    pgconv.PgUUIDToUUID(r.RegisterID),
		UserID:        pgconv.PgUUIDToUUID(r.UserID),
		Status:        string(r.Status),
		OpeningAmount: pgconv.PgNumericToFloat64(r.OpeningAmount),
		OpeningNotes:  pgconv.ParsePgTextToStringPtr(r.OpeningNotes),
		OpenedAt:      pgconv.PgTimestamptzToTime(r.OpenedAt),
		ClosedAt:      pgconv.PgTimestamptzToTimePtr(r.ClosedAt),
		ClosingNotes:  pgconv.ParsePgTextToStringPtr(r.ClosingNotes),
		SalesTotal:    numericPtr(r.SalesTotal),
		SalesCount:    pgconv.PgInt4ToIntPtr(r.SalesCount),
		ReceivedTotal: numericPtr(r.ReceivedTotal),
		ExpectedCash:  numericPtr(r.ExpectedCash),
		CountedCash:   numericPtr(r.CountedCash),
		Difference:    numericPtr(r.Difference),
	}
}

func numericPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	v := pgconv.PgNumericToFloat64(n)
	return &v
}
