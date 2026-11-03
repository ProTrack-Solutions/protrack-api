package service

import (
	"context"
	"errors"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_register/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_register/repository"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// lockSessions trava os dois turnos da transferência (sempre na mesma ordem) e
// devolve cada um pelo id
func lockSessions(ctx context.Context, repo *repository.Repository, ids ...pgtype.UUID) (map[pgtype.UUID]db.CashSession, error) {
	rows, err := repo.LockCashSessions(ctx, ids)
	if err != nil {
		return nil, err
	}
	sessions := make(map[pgtype.UUID]db.CashSession, len(rows))
	for _, r := range rows {
		sessions[r.ID] = r
	}
	return sessions, nil
}

// CreateTransfer leva troco do caixa deste computador para outro caixa aberto.
// O dinheiro sai da gaveta agora; o destino só recebe quando alguém de lá aceitar.
func (s *Service) CreateTransfer(ctx context.Context, companyId, userId uuid.UUID, req domain.CreateTransferRequest) (domain.CashSessionDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)
	repo := s.repo.WithTx(tx)
	company := pgconv.ParseUUIDToPgType(companyId)

	fromRegister, err := resolveRegister(ctx, q, companyId)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	if pgconv.PgUUIDToUUID(fromRegister) == req.ToRegisterID {
		return domain.CashSessionDetail{}, domain.ErrTransferSameRegister
	}

	toRegister, err := repo.GetCashRegisterById(ctx, db.GetCashRegisterByIdParams{
		ID:        pgconv.ParseUUIDToPgType(req.ToRegisterID),
		CompanyID: company,
	})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !toRegister.Active) {
		return domain.CashSessionDetail{}, domain.ErrCashRegisterNotFound
	}
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	from, err := repo.GetOpenCashSessionByRegister(ctx, fromRegister)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CashSessionDetail{}, domain.ErrNoOpenCashSession
	}
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	to, err := repo.GetOpenCashSessionByRegister(ctx, toRegister.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CashSessionDetail{}, domain.ErrTransferTargetClosed
	}
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	// Algum dos dois pode ter fechado entre a busca e a trava
	locked, err := lockSessions(ctx, repo, from.ID, to.ID)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	if locked[from.ID].Status != db.CashSessionStatusOpen {
		return domain.CashSessionDetail{}, domain.ErrNoOpenCashSession
	}
	if locked[to.ID].Status != db.CashSessionStatusOpen {
		return domain.CashSessionDetail{}, domain.ErrTransferTargetClosed
	}

	amount := round(req.Amount)
	summary, err := s.summary(ctx, repo, toSession(locked[from.ID]))
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	if amount > summary.ExpectedCash {
		return domain.CashSessionDetail{}, domain.ErrInsufficientCash
	}

	transfer, err := repo.CreateCashTransfer(ctx, db.CreateCashTransferParams{
		CompanyID:     company,
		FromSessionID: from.ID,
		ToSessionID:   to.ID,
		Amount:        pgconv.Float64ToPgNumeric(amount),
		Reason:        pgconv.ParseStringToPgText(req.Reason),
		CreatedBy:     pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.CashSessionDetail{}, err
	}

	if _, err := repo.CreateCashSessionMovement(ctx, db.CreateCashSessionMovementParams{
		SessionID:  from.ID,
		CompanyID:  company,
		Type:       db.CashMovementTypeWithdrawal,
		Amount:     transfer.Amount,
		Reason:     transfer.Reason,
		CreatedBy:  transfer.CreatedBy,
		TransferID: transfer.ID,
	}); err != nil {
		return domain.CashSessionDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.CashSessionDetail{}, err
	}
	return s.GetById(ctx, companyId, userId, true, pgconv.PgUUIDToUUID(from.ID))
}

// AcceptTransfer: alguém do caixa de destino confirma que recebeu o troco
func (s *Service) AcceptTransfer(ctx context.Context, companyId, userId, transferId uuid.UUID) (domain.CashSessionDetail, error) {
	return s.resolveTransfer(ctx, companyId, userId, transferId, db.CashTransferStatusAccepted)
}

// RejectTransfer: o caixa de destino não recebeu; o dinheiro volta para a origem
func (s *Service) RejectTransfer(ctx context.Context, companyId, userId, transferId uuid.UUID) (domain.CashSessionDetail, error) {
	return s.resolveTransfer(ctx, companyId, userId, transferId, db.CashTransferStatusRejected)
}

// CancelTransfer: quem enviou desiste antes do aceite; o dinheiro volta para a origem
func (s *Service) CancelTransfer(ctx context.Context, companyId, userId, transferId uuid.UUID) (domain.CashSessionDetail, error) {
	return s.resolveTransfer(ctx, companyId, userId, transferId, db.CashTransferStatusCanceled)
}

func (s *Service) resolveTransfer(ctx context.Context, companyId, userId, transferId uuid.UUID, status db.CashTransferStatus) (domain.CashSessionDetail, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)
	repo := s.repo.WithTx(tx)

	// A trava na transferência impede aceitar e cancelar ao mesmo tempo
	transfer, err := repo.GetCashTransferForUpdate(ctx, db.GetCashTransferForUpdateParams{
		ID:        pgconv.ParseUUIDToPgType(transferId),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CashSessionDetail{}, domain.ErrTransferNotFound
	}
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	if transfer.Status != db.CashTransferStatusPending {
		return domain.CashSessionDetail{}, domain.ErrTransferAlreadyResolved
	}

	locked, err := lockSessions(ctx, repo, transfer.FromSessionID, transfer.ToSessionID)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	from, to := locked[transfer.FromSessionID], locked[transfer.ToSessionID]

	// Aceitar e recusar é no computador do destino; cancelar, no da origem
	register, err := resolveRegister(ctx, q, companyId)
	if err != nil {
		return domain.CashSessionDetail{}, err
	}
	actingSession := to
	if status == db.CashTransferStatusCanceled {
		actingSession = from
	}
	if register != actingSession.RegisterID {
		return domain.CashSessionDetail{}, domain.ErrTransferWrongRegister
	}

	// Aceita: entra no destino. Recusada/cancelada: volta para a origem.
	// Os turnos não fecham com transferência pendente, então os dois estão abertos.
	target := from
	if status == db.CashTransferStatusAccepted {
		target = to
	}
	if _, err := repo.CreateCashSessionMovement(ctx, db.CreateCashSessionMovementParams{
		SessionID:  target.ID,
		CompanyID:  transfer.CompanyID,
		Type:       db.CashMovementTypeSupply,
		Amount:     transfer.Amount,
		Reason:     transfer.Reason,
		CreatedBy:  pgconv.ParseUUIDToPgType(userId),
		TransferID: transfer.ID,
	}); err != nil {
		return domain.CashSessionDetail{}, err
	}

	if err := repo.ResolveCashTransfer(ctx, db.ResolveCashTransferParams{
		ID:         transfer.ID,
		Status:     status,
		ResolvedBy: pgconv.ParseUUIDToPgType(userId),
	}); err != nil {
		return domain.CashSessionDetail{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.CashSessionDetail{}, err
	}
	return s.GetById(ctx, companyId, userId, true, pgconv.PgUUIDToUUID(actingSession.ID))
}

func toPendingTransfer(r db.ListPendingCashTransfersRow) domain.PendingTransfer {
	return domain.PendingTransfer{
		ID:               pgconv.PgUUIDToUUID(r.ID),
		Amount:           pgconv.PgNumericToFloat64(r.Amount),
		Reason:           pgconv.ParsePgTextToStringPtr(r.Reason),
		FromRegisterName: r.FromRegisterName,
		ToRegisterName:   r.ToRegisterName,
		CreatedByName:    r.CreatedByName,
		CreatedAt:        pgconv.PgTimestamptzToTime(r.CreatedAt),
	}
}
