package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"time"

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

// Enquanto o caixa está aberto, o turno vai até "agora": qualquer venda nova do
// operador entra no resumo. Um limite bem distante evita depender do relógio da API.
var openEnd = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

func round(v float64) float64 {
	return math.Round(v*100) / 100
}

func (s *Service) Open(ctx context.Context, companyId, userId uuid.UUID, req domain.OpenCashSessionRequest) (domain.CashSessionDetail, error) {
	_, err := s.repo.GetOpenCashSessionByUser(ctx, db.GetOpenCashSessionByUserParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		UserID:    pgconv.ParseUUIDToPgType(userId),
	})
	if err == nil {
		return domain.CashSessionDetail{}, domain.ErrCashSessionAlreadyOpen
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.CashSessionDetail{}, err
	}

	session, err := s.repo.CreateCashSession(ctx, db.CreateCashSessionParams{
		CompanyID:     pgconv.ParseUUIDToPgType(companyId),
		UserID:        pgconv.ParseUUIDToPgType(userId),
		OpeningAmount: pgconv.Float64ToPgNumeric(round(req.OpeningAmount)),
		OpeningNotes:  pgconv.ParseStringToPgText(req.Notes),
	})
	if err != nil {
		// Duas aberturas ao mesmo tempo: o índice único segura a segunda
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.CashSessionDetail{}, domain.ErrCashSessionAlreadyOpen
		}
		return domain.CashSessionDetail{}, err
	}

	return s.detail(ctx, s.repo, toSession(session, ""))
}

// Current devolve o caixa aberto do operador, ou nil quando não há nenhum.
func (s *Service) Current(ctx context.Context, companyId, userId uuid.UUID) (*domain.CashSessionDetail, error) {
	session, err := s.repo.GetOpenCashSessionByUser(ctx, db.GetOpenCashSessionByUserParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		UserID:    pgconv.ParseUUIDToPgType(userId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	detail, err := s.detail(ctx, s.repo, toSession(session, ""))
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

	open, err := lockOpenSession(ctx, repo, companyId, userId)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	session := toSession(open, "")

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

	detail, err := s.detail(ctx, repo, session)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	return detail, tx.Commit(ctx)
}

func (s *Service) Close(ctx context.Context, companyId, userId uuid.UUID, req domain.CloseCashSessionRequest) (domain.CashSessionDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	defer tx.Rollback(ctx)

	repo := s.repo.WithTx(tx)

	open, err := lockOpenSession(ctx, repo, companyId, userId)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	summary, err := s.summary(ctx, repo, toSession(open, ""))
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	counted := round(req.CountedCash)

	closed, err := repo.CloseCashSession(ctx, db.CloseCashSessionParams{
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
	})
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	detail, err := s.detail(ctx, repo, toSession(closed, ""))
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	return detail, tx.Commit(ctx)
}

// List mostra o histórico de caixas. O administrador vê os de todos os operadores.
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

	rows, err := s.repo.ListCashSessions(ctx, db.ListCashSessionsParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		UserID:    filter,
		RowLimit:  perPage,
		RowOffset: (page - 1) * perPage,
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
		resp.Data = append(resp.Data, toSession(db.CashSession{
			ID:               r.ID,
			CompanyID:        r.CompanyID,
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
		}, r.UserName))
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

	// Operador comum só enxerga os próprios caixas
	if !isAdmin && pgconv.PgUUIDToUUID(r.UserID) != userId {
		return domain.CashSessionDetail{}, domain.ErrCashSessionNotFound
	}

	return s.detail(ctx, s.repo, toSession(db.CashSession{
		ID:               r.ID,
		CompanyID:        r.CompanyID,
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
	}, r.UserName))
}

// RequireOpenSession barra vendas e recebimentos do operador sem caixa aberto
// quando a empresa liga a configuração require_open_cash_register. Empresas sem
// a configuração gravada seguem sem a exigência.
func RequireOpenSession(ctx context.Context, tx db.DBTX, companyId, userId uuid.UUID) error {
	q := db.New(tx)

	setting, err := q.GetCompanySetting(ctx, db.GetCompanySettingParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Key:       string(enums.RequireOpenCashRegister),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	var required bool
	if err := json.Unmarshal([]byte(setting.Value), &required); err != nil {
		return err
	}
	if !required {
		return nil
	}

	_, err = q.GetOpenCashSessionByUser(ctx, db.GetOpenCashSessionByUserParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		UserID:    pgconv.ParseUUIDToPgType(userId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrCashRegisterRequired
	}
	return err
}

func lockOpenSession(ctx context.Context, repo *repository.Repository, companyId, userId uuid.UUID) (db.CashSession, error) {
	open, err := repo.GetOpenCashSessionByUserForUpdate(ctx, db.GetOpenCashSessionByUserForUpdateParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		UserID:    pgconv.ParseUUIDToPgType(userId),
	})
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
			ID:        pgconv.PgUUIDToUUID(m.ID),
			Type:      string(m.Type),
			Amount:    pgconv.PgNumericToFloat64(m.Amount),
			Reason:    pgconv.ParsePgTextToStringPtr(m.Reason),
			CreatedAt: pgconv.PgTimestamptzToTime(m.CreatedAt),
		})
	}

	return domain.CashSessionDetail{
		Session:   session,
		Summary:   summary,
		Movements: movements,
	}, nil
}

// summary soma o turno do operador: vendas e parcelas recebidas entre a abertura
// e o fechamento, mais sangrias e suprimentos.
func (s *Service) summary(ctx context.Context, repo *repository.Repository, session domain.CashSession) (domain.CashSummary, error) {
	end := openEnd
	if session.ClosedAt != nil {
		end = *session.ClosedAt
	}

	companyId := pgconv.ParseUUIDToPgType(session.CompanyID)
	userId := pgconv.ParseUUIDToPgType(session.UserID)
	startAt := pgconv.TimeToPgTimestamptz(session.OpenedAt)
	endAt := pgconv.TimeToPgTimestamptz(end)

	sales, err := repo.GetCashSessionSalesByMethod(ctx, db.GetCashSessionSalesByMethodParams{
		CompanyID: companyId, UserID: userId, StartAt: startAt, EndAt: endAt,
	})
	if err != nil {
		return domain.CashSummary{}, err
	}

	receipts, err := repo.GetCashSessionReceiptsByMethod(ctx, db.GetCashSessionReceiptsByMethodParams{
		CompanyID: companyId, UserID: userId, StartAt: startAt, EndAt: endAt,
	})
	if err != nil {
		return domain.CashSummary{}, err
	}

	refunds, err := repo.GetCashSessionRefundsByMethod(ctx, db.GetCashSessionRefundsByMethodParams{
		CompanyID: companyId, UserID: userId, StartAt: startAt, EndAt: endAt,
	})
	if err != nil {
		return domain.CashSummary{}, err
	}

	movements, err := repo.ListCashSessionMovements(ctx, pgconv.ParseUUIDToPgType(session.ID))
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

func toSession(r db.CashSession, userName string) domain.CashSession {
	return domain.CashSession{
		ID:            pgconv.PgUUIDToUUID(r.ID),
		CompanyID:     pgconv.PgUUIDToUUID(r.CompanyID),
		UserID:        pgconv.PgUUIDToUUID(r.UserID),
		UserName:      userName,
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
