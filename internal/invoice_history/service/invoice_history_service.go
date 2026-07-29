package service

import (
	"context"
	"time"

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

	return repoTx.CreateInvoiceHistory(ctx, db.CreateInvoiceHistoryParams{
		SubscriptionID:  pgconv.ParseUUIDToPgType(req.SubscriptionID),
		CompanyID:       pgconv.ParseUUIDToPgType(companyId),
		MpPaymentID:     req.MpPaymentID,
		AmountCents:     req.AmountCents,
		Status:          req.Status,
		PaymentMethodID: pgconv.ParseUUIDToPgType(req.PaymentMethodID),
		PaidAt:          pgconv.TimeToPgTimestamptz(req.PaidAt),
	})
}

func (s *Service) UpdateInvoceStatus(ctx context.Context, tx db.DBTX, mpPaymentId string, req domain.UpdateInvoceStatusRequest) error {
	repoTx := db.New(tx)

	return repoTx.UpdateInvoiceStatus(ctx, db.UpdateInvoiceStatusParams{
		MpPaymentID: mpPaymentId,
		Status:      req.Status,
		PaidAt:      pgconv.TimeToPgTimestamptz(time.Now()),
	})
}

func (s *Service) GetInvoceById(ctx context.Context, id uuid.UUID) (domain.InvoiceHistoryResponse, error) {
	invoice, err := s.repo.GetInvoiceById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.InvoiceHistoryResponse{}, nil
	}

	return domain.InvoiceHistoryResponse{
		ID:              pgconv.PgUUIDToUUID(invoice.CompanyID),
		SubscriptionID:  pgconv.PgUUIDToUUID(invoice.SubscriptionID),
		CompanyID:       pgconv.PgUUIDToUUID(invoice.CompanyID),
		PaymentMethodID: pgconv.PgUUIDToUUID(invoice.PaymentMethodID),
		MpPaymentID:     invoice.MpPaymentID,
		AmountCents:     invoice.AmountCents,
		Status:          invoice.Status,
		PaidAt:          pgconv.PgTimestamptzToTime(invoice.PaidAt),
		CreatedAt:       pgconv.PgTimestamptzToTime(invoice.CreatedAt),
		UpdatedAt:       pgconv.PgTimestamptzToTime(invoice.UpdatedAt),
	}, nil
}

func (s *Service) GetInvoiceByMpPaymentId(ctx context.Context, mpPaymentId string) (domain.InvoiceHistoryResponse, error) {
	invoice, err := s.repo.GetInvoiceByMpPaymentId(ctx, mpPaymentId)
	if err != nil {
		return domain.InvoiceHistoryResponse{}, nil
	}

	return domain.InvoiceHistoryResponse{
		ID:              pgconv.PgUUIDToUUID(invoice.CompanyID),
		SubscriptionID:  pgconv.PgUUIDToUUID(invoice.SubscriptionID),
		CompanyID:       pgconv.PgUUIDToUUID(invoice.CompanyID),
		PaymentMethodID: pgconv.PgUUIDToUUID(invoice.PaymentMethodID),
		MpPaymentID:     invoice.MpPaymentID,
		AmountCents:     invoice.AmountCents,
		Status:          invoice.Status,
		PaidAt:          pgconv.PgTimestamptzToTime(invoice.PaidAt),
		CreatedAt:       pgconv.PgTimestamptzToTime(invoice.CreatedAt),
		UpdatedAt:       pgconv.PgTimestamptzToTime(invoice.UpdatedAt),
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
			ID:              pgconv.PgUUIDToUUID(in.CompanyID),
			SubscriptionID:  pgconv.PgUUIDToUUID(in.SubscriptionID),
			CompanyID:       pgconv.PgUUIDToUUID(in.CompanyID),
			PaymentMethodID: pgconv.PgUUIDToUUID(in.PaymentMethodID),
			MpPaymentID:     in.MpPaymentID,
			AmountCents:     in.AmountCents,
			Status:          in.Status,
			PaidAt:          pgconv.PgTimestamptzToTime(in.PaidAt),
			CreatedAt:       pgconv.PgTimestamptzToTime(in.CreatedAt),
			UpdatedAt:       pgconv.PgTimestamptzToTime(in.UpdatedAt),
		})
	}

	responsePaginate := globaldomain.NewPaginatedResponse(response, total, pagination)

	return domain.InvoiceHistoryPaginatedResponse{
		PaginatedResponse: responsePaginate,
	}, nil
}
