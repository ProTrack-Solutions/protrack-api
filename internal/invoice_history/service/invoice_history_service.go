package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globaldomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/invoice_history/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterfave interface {
	CreateInvoiceHistory(ctx context.Context, arg db.CreateInvoiceHistoryParams) error
	UpdateInvoiceStatus(ctx context.Context, arg db.UpdateInvoiceStatusParams) error
	GetInvoiceById(ctx context.Context, id pgtype.UUID) (db.InvoiceHistory, error)
	GetInvoiceByMpPaymentId(ctx context.Context, mpPaymentId string) (db.InvoiceHistory, error)
	ListInvoiceByCompany(ctx context.Context, arg db.ListInvoicesByCompanyParams) ([]db.InvoiceHistory, error)
	CountInvouces(ctx context.Context, companyId pgtype.UUID) (int64, error)
}

type Service struct {
	repo RepositoryInterfave
	pool *pgxpool.Pool
}

func NewServie(repo RepositoryInterfave, pool *pgxpool.Pool) *Service {
	return &Service{
		repo: repo,
		pool: pool,
	}
}

func (s *Service) CreateInvoiceHistory(ctx context.Context, tx db.DBTX, companyId uuid.UUID, req domain.CreateInvoicementHistoryRequest) error {
	repoTx := db.New(tx)

	// payment_method_id é opcional (a subscription pode não ter um método de
	// pagamento vinculado); um uuid.Nil aqui violaria a FK para
	// subscription_payment_methods, então mandamos NULL nesse caso.
	var paymentMethodID uuid.UUID
	if req.PaymentMethodID != uuid.Nil {
		paymentMethodID = req.PaymentMethodID
	}

	return repoTx.CreateInvoiceHistory(ctx, db.CreateInvoiceHistoryParams{
		SubscriptionID:    pgconv.ParseUUIDToPgType(req.SubscriptionID),
		CompanyID:         pgconv.ParseUUIDToPgType(companyId),
		ExternalPaymentID: req.ExternalPaymentID,
		AmountCents:       req.AmountCents,
		Status:            req.Status,
		PaymentMethodID:   pgconv.ParseUUIDToPgType(paymentMethodID),
		PaidAt:            pgconv.TimeToPgTimestamptz(req.PaidAt),
	})
}

func (s *Service) UpdateInvoceStatus(ctx context.Context, tx db.DBTX, ExternalPaymentID string, req domain.UpdateInvoceStatusRequest) error {
	repoTx := db.New(tx)

	return repoTx.UpdateInvoiceStatus(ctx, db.UpdateInvoiceStatusParams{
		ExternalPaymentID: ExternalPaymentID,
		Status:            req.Status,
		PaidAt:            pgconv.TimeToPgTimestamptz(req.PaidAt),
	})
}

// UpsertFromWebhook grava ou atualiza o histórico de uma fatura a partir de um
// evento de webhook do gateway de pagamento (Stripe). Usa o pool diretamente
// (sem exigir uma transação do chamador), pensado para ser chamado a partir
// das rotas que efetivamente processam pagamentos.
func (s *Service) UpsertFromWebhook(ctx context.Context, companyId uuid.UUID, req domain.CreateInvoicementHistoryRequest) error {
	if _, err := s.repo.GetInvoiceByMpPaymentId(ctx, req.ExternalPaymentID); err != nil {
		// Ainda não existe um registro para essa fatura: cria.
		return s.CreateInvoiceHistory(ctx, s.pool, companyId, req)
	}

	return s.UpdateInvoceStatus(ctx, s.pool, req.ExternalPaymentID, domain.UpdateInvoceStatusRequest{
		Status: req.Status,
		PaidAt: req.PaidAt,
	})
}

func (s *Service) GetInvoceById(ctx context.Context, id uuid.UUID) (domain.InvoiceHistoryResponse, error) {
	invoice, err := s.repo.GetInvoiceById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.InvoiceHistoryResponse{}, err
	}

	return domain.InvoiceHistoryResponse{
		ID:                pgconv.PgUUIDToUUID(invoice.ID),
		SubscriptionID:    pgconv.PgUUIDToUUID(invoice.SubscriptionID),
		CompanyID:         pgconv.PgUUIDToUUID(invoice.CompanyID),
		PaymentMethodID:   pgconv.PgUUIDToUUID(invoice.PaymentMethodID),
		ExternalPaymentID: invoice.ExternalPaymentID,
		AmountCents:       invoice.AmountCents,
		Status:            invoice.Status,
		PaidAt:            pgconv.PgTimestamptzToTime(invoice.PaidAt),
		CreatedAt:         pgconv.PgTimestamptzToTime(invoice.CreatedAt),
		UpdatedAt:         pgconv.PgTimestamptzToTime(invoice.UpdatedAt),
	}, nil
}

func (s *Service) GetInvoiceByMpPaymentId(ctx context.Context, ExternalPaymentID string) (domain.InvoiceHistoryResponse, error) {
	invoice, err := s.repo.GetInvoiceByMpPaymentId(ctx, ExternalPaymentID)
	if err != nil {
		return domain.InvoiceHistoryResponse{}, err
	}

	return domain.InvoiceHistoryResponse{
		ID:                pgconv.PgUUIDToUUID(invoice.ID),
		SubscriptionID:    pgconv.PgUUIDToUUID(invoice.SubscriptionID),
		CompanyID:         pgconv.PgUUIDToUUID(invoice.CompanyID),
		PaymentMethodID:   pgconv.PgUUIDToUUID(invoice.PaymentMethodID),
		ExternalPaymentID: invoice.ExternalPaymentID,
		AmountCents:       invoice.AmountCents,
		Status:            invoice.Status,
		PaidAt:            pgconv.PgTimestamptzToTime(invoice.PaidAt),
		CreatedAt:         pgconv.PgTimestamptzToTime(invoice.CreatedAt),
		UpdatedAt:         pgconv.PgTimestamptzToTime(invoice.UpdatedAt),
	}, nil
}

func (s *Service) ListInvoiceByCompany(ctx context.Context, companyId uuid.UUID, pagination globaldomain.PaginationParams) (domain.InvoiceHistoryPaginatedResponse, error) {
	total, err := s.repo.CountInvouces(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.InvoiceHistoryPaginatedResponse{}, err
	}

	invoice, err := s.repo.ListInvoiceByCompany(ctx, db.ListInvoicesByCompanyParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Limit:     pagination.PerPage,
		Offset:    (pagination.Page - 1) * pagination.PerPage,
	})
	if err != nil {
		return domain.InvoiceHistoryPaginatedResponse{}, err
	}

	var response []domain.InvoiceHistoryResponse

	for _, in := range invoice {
		response = append(response, domain.InvoiceHistoryResponse{
			ID:                pgconv.PgUUIDToUUID(in.ID),
			SubscriptionID:    pgconv.PgUUIDToUUID(in.SubscriptionID),
			CompanyID:         pgconv.PgUUIDToUUID(in.CompanyID),
			PaymentMethodID:   pgconv.PgUUIDToUUID(in.PaymentMethodID),
			ExternalPaymentID: in.ExternalPaymentID,
			AmountCents:       in.AmountCents,
			Status:            in.Status,
			PaidAt:            pgconv.PgTimestamptzToTime(in.PaidAt),
			CreatedAt:         pgconv.PgTimestamptzToTime(in.CreatedAt),
			UpdatedAt:         pgconv.PgTimestamptzToTime(in.UpdatedAt),
		})
	}

	responsePaginate := globaldomain.NewPaginatedResponse(response, total, pagination)

	return domain.InvoiceHistoryPaginatedResponse{
		PaginatedResponse: responsePaginate,
	}, nil
}
