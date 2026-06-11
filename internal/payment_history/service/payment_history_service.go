package service

import (
	"context"
	"time"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/payment_history/domain"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/payment_history/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreatePaymentHistory(ctx context.Context, arg db.CreatePaymentHistoryParams) error
	GetPaymentsByCustomer(ctx context.Context, arg db.GetPaymentsByCustomerParams) ([]db.PaymentHistory, error)
	GetPaymentsBySale(ctx context.Context, arg db.GetPaymentsBySaleParams) ([]db.PaymentHistory, error)
	GetTotalReceivedByPeriod(ctx context.Context, arg db.GetTotalReceivedByPeriodParams) (pgtype.Numeric, error)
	ListPaymentHistory(ctx context.Context, companyId pgtype.UUID) ([]db.ListPaymentHistoryRow, error)
	GetPaymentsHistoryReport(ctx context.Context, arg db.GetPaymentsHistoryReportParams) ([]db.GetPaymentsHistoryReportRow, error)
}

type Service struct {
	repo RepositoryInterface
	pool *pgxpool.Pool
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool) *Service {
	return &Service{
		repo: repo,
		pool: pool,
	}
}

func (s *Service) CreatePaymentHistory(ctx context.Context, req domain.CreatePaymentHistoryRequest) error {
	return s.repo.CreatePaymentHistory(ctx, db.CreatePaymentHistoryParams{
		CompanyID:       pgconv.ParseUUIDToPgType(req.CompanyID),
		CustomerID:      pgconv.ParseUUIDToPgType(req.CustomerID),
		SaleID:          pgconv.ParseUUIDToPgType(req.SaleID),
		PaymentMethodID: pgconv.ParseUUIDToPgType(req.PaymentMethodID),
		UserID:          pgconv.ParseUUIDToPgType(req.UserID),
		AmountPaid:      pgconv.Float64ToPgNumeric(req.AmountPaid),
		Notes:           pgconv.ParseStringToPgText(req.Notes),
	})
}

func (s *Service) CreatePaymentHistoryTx(ctx context.Context, tx db.DBTX, req domain.CreatePaymentHistoryRequest) error {
	repoTx := db.New(tx)

	return repoTx.CreatePaymentHistory(ctx, db.CreatePaymentHistoryParams{
		CompanyID:       pgconv.ParseUUIDToPgType(req.CompanyID),
		CustomerID:      pgconv.ParseUUIDToPgType(req.CustomerID),
		SaleID:          pgconv.ParseUUIDToPgType(req.SaleID),
		PaymentMethodID: pgconv.ParseUUIDToPgType(req.PaymentMethodID),
		UserID:          pgconv.ParseUUIDToPgType(req.UserID),
		AmountPaid:      pgconv.Float64ToPgNumeric(req.AmountPaid),
		Notes:           pgconv.ParseStringToPgText(req.Notes),
	})
}

func (s *Service) GetPaymentsByCustomer(ctx context.Context, req domain.GetPaymentsByCustomerRequest) ([]domain.PaymentHistoryResponse, error) {
	paymentsHistory, err := s.repo.GetPaymentsByCustomer(ctx, db.GetPaymentsByCustomerParams{
		CompanyID:  pgconv.ParseUUIDToPgType(req.CompanyID),
		CustomerID: pgconv.ParseUUIDToPgType(req.CustomerID),
	})
	if err != nil {
		return []domain.PaymentHistoryResponse{}, err
	}

	var response []domain.PaymentHistoryResponse

	for _, paymentHistory := range paymentsHistory {
		response = append(response, domain.PaymentHistoryResponse{
			ID:              pgconv.PgUUIDToUUID(paymentHistory.ID),
			CompanyID:       pgconv.PgUUIDToUUID(paymentHistory.CompanyID),
			CustomerID:      pgconv.PgUUIDToUUID(paymentHistory.CustomerID),
			SaleID:          pgconv.PgUUIDToUUID(paymentHistory.SaleID),
			PaymentMethodID: pgconv.PgUUIDToUUID(paymentHistory.PaymentMethodID),
			UserID:          pgconv.PgUUIDToUUID(paymentHistory.UserID),
			AmountPaid:      pgconv.PgNumericToFloat64(paymentHistory.AmountPaid),
			PaymentDate:     pgconv.PgTimestamptzToTime(paymentHistory.PaymentDate),
			Notes:           pgconv.ParsePgTextToString(paymentHistory.Notes),
		})
	}

	return response, nil
}

func (s *Service) GetPaymentsBySale(ctx context.Context, req domain.GetPaymentsBySaleRequest) ([]domain.PaymentHistoryResponse, error) {
	paymentsHistory, err := s.repo.GetPaymentsBySale(ctx, db.GetPaymentsBySaleParams{
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		SaleID:    pgconv.ParseUUIDToPgType(req.SaleID),
	})
	if err != nil {
		return []domain.PaymentHistoryResponse{}, err
	}

	var response []domain.PaymentHistoryResponse

	for _, paymentHistory := range paymentsHistory {
		response = append(response, domain.PaymentHistoryResponse{
			ID:              pgconv.PgUUIDToUUID(paymentHistory.ID),
			CompanyID:       pgconv.PgUUIDToUUID(paymentHistory.CompanyID),
			CustomerID:      pgconv.PgUUIDToUUID(paymentHistory.CustomerID),
			SaleID:          pgconv.PgUUIDToUUID(paymentHistory.SaleID),
			PaymentMethodID: pgconv.PgUUIDToUUID(paymentHistory.PaymentMethodID),
			UserID:          pgconv.PgUUIDToUUID(paymentHistory.UserID),
			AmountPaid:      pgconv.PgNumericToFloat64(paymentHistory.AmountPaid),
			PaymentDate:     pgconv.PgTimestamptzToTime(paymentHistory.PaymentDate),
			Notes:           pgconv.ParsePgTextToString(paymentHistory.Notes),
		})
	}

	return response, nil
}

func (s *Service) GetTotalReceivedByPeriod(ctx context.Context, req domain.GetTotalReceivedByPeriodRequest) (float64, error) {
	total, err := s.repo.GetTotalReceivedByPeriod(ctx, db.GetTotalReceivedByPeriodParams{
		CompanyID:     pgconv.ParseUUIDToPgType(req.CompanyID),
		PaymentDate:   pgconv.TimeToPgTimestamptz(req.PaymentDate),
		PaymentDate_2: pgconv.TimeToPgTimestamptz(req.PaymentDate_2),
	})
	if err != nil {
		return 0, err
	}

	return pgconv.PgNumericToFloat64(total), nil
}

func (s *Service) ListPaymentHistory(ctx context.Context, companyId uuid.UUID) ([]domain.ListPaymentHistoryRow, error) {
	paymentsHistory, err := s.repo.ListPaymentHistory(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.ListPaymentHistoryRow{}, err
	}

	var response []domain.ListPaymentHistoryRow

	for _, paymentHistory := range paymentsHistory {
		response = append(response, domain.ListPaymentHistoryRow{
			ID:                pgconv.PgUUIDToUUID(paymentHistory.ID),
			AmountPaid:        pgconv.PgNumericToFloat64(paymentHistory.AmountPaid),
			PaymentDate:       pgconv.PgTimestamptzToTime(paymentHistory.PaymentDate),
			Notes:             pgconv.ParsePgTextToString(paymentHistory.Notes),
			CustomerName:      paymentHistory.CustomerName,
			UserName:          paymentHistory.UserName,
			PaymentMethodName: pgconv.ParsePgTextToString(paymentHistory.PaymentMethodName),
			SaleID:            pgconv.PgUUIDToUUID(paymentHistory.SaleID),
		})
	}

	return response, nil
}

func (s *Service) GetPaymentsHistoryReport(ctx context.Context, companyId uuid.UUID, startDate time.Time, endDate time.Time) ([]domain.GetPaymentsHistoryReportResponse, error) {
	paymentsHistory, err := s.repo.GetPaymentsHistoryReport(ctx, db.GetPaymentsHistoryReportParams{
		CompanyID:     pgconv.ParseUUIDToPgType(companyId),
		PaymentDate:   pgconv.TimeToPgTimestamptz(startDate),
		PaymentDate_2: pgconv.TimeToPgTimestamptz(endDate),
	})
	if err != nil {
		return []domain.GetPaymentsHistoryReportResponse{}, err
	}

	var response []domain.GetPaymentsHistoryReportResponse

	for _, paymentHistory := range paymentsHistory {
		response = append(response, domain.GetPaymentsHistoryReportResponse{
			AmountPaid:        pgconv.PgNumericToFloat64(paymentHistory.AmountPaid),
			PaymentDate:       pgconv.PgTimestamptzToTime(paymentHistory.PaymentDate),
			Notes:             pgconv.ParsePgTextToString(paymentHistory.Notes),
			CustomerName:      paymentHistory.CustomerName,
			UserName:          paymentHistory.UserName,
			PaymentMethodName: pgconv.ParsePgTextToString(paymentHistory.PaymentMethodName),
		})
	}

	return response, nil
}
