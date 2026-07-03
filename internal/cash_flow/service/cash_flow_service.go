package service

import (
	"context"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_flow/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_flow/repository"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	GetTotalInflowByPeriod(ctx context.Context, arg db.GetTotalInflowByPeriodParams) (float64, error)
	GetTotalOutflowByPeriod(ctx context.Context, arg db.GetTotalOutflowByPeriodParams) (float64, error)
	GetCashInFlowByCategory(ctx context.Context, companyId pgtype.UUID) ([]db.GetCashInFlowByCategoryRow, error)
	GetCashOutFlowByCategory(ctx context.Context, companyId pgtype.UUID) ([]db.GetCashOutFlowByCategoryRow, error)
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

func (s *Service) CashFlowSummary(ctx context.Context, companyId uuid.UUID, startAt time.Time, endAt time.Time) (domain.CashFlowSummaryResponse, error) {
	totalInFlow, err := s.repo.GetTotalInflowByPeriod(ctx, db.GetTotalInflowByPeriodParams{
		CompanyID:   pgconv.ParseUUIDToPgType(companyId),
		CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
		CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
	})
	if err != nil {
		return domain.CashFlowSummaryResponse{}, err
	}

	totalOutFlow, err := s.repo.GetTotalOutflowByPeriod(ctx, db.GetTotalOutflowByPeriodParams{
		CompanyID:     pgconv.ParseUUIDToPgType(companyId),
		PaymentDate:   pgconv.StringToPgDate(startAt.GoString()),
		PaymentDate_2: pgconv.StringToPgDate(endAt.GoString()),
	})
	if err != nil {
		return domain.CashFlowSummaryResponse{}, err
	}

	netBalance := totalInFlow - totalOutFlow

	return domain.CashFlowSummaryResponse{
		TotalInflow:  totalInFlow,
		TotalOutflow: totalOutFlow,
		NetBalance:   netBalance,
	}, nil
}

func (s *Service) GetCashFlowHistoryProjections(ctx context.Context, companyId uuid.UUID) ([]domain.GetCashFlowHistoryProjectionsResponse, error) {
	dayBase := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)

	var response []domain.GetCashFlowHistoryProjectionsResponse

	var accumulatedBalance float64

	for i := 0; i < 12; i++ {
		startAt := dayBase.AddDate(0, -i, 0)

		endAt := startAt.AddDate(0, 1, 0).Add(-time.Nanosecond)

		totalInFlow, err := s.repo.GetTotalInflowByPeriod(ctx, db.GetTotalInflowByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
		})
		if err != nil {
			return []domain.GetCashFlowHistoryProjectionsResponse{}, err
		}

		totalOutFlow, err := s.repo.GetTotalOutflowByPeriod(ctx, db.GetTotalOutflowByPeriodParams{
			CompanyID:     pgconv.ParseUUIDToPgType(companyId),
			PaymentDate:   pgconv.StringToPgDate(startAt.GoString()),
			PaymentDate_2: pgconv.StringToPgDate(endAt.GoString()),
		})
		if err != nil {
			return []domain.GetCashFlowHistoryProjectionsResponse{}, err
		}

		accumulatedBalance += totalInFlow - totalOutFlow

		response = append(response, domain.GetCashFlowHistoryProjectionsResponse{
			Date:               startAt.Format("02/01/2006"),
			TotalInflow:        totalInFlow,
			TotalOutflow:       totalOutFlow,
			AccumulatedBalance: accumulatedBalance,
		})

	}

	return response, nil
}

func (s *Service) GetCashInFlowByCategory(ctx context.Context, companyId uuid.UUID) ([]domain.GetCashInFlowByCategoryResponse, error) {
	cashInFlow, err := s.repo.GetCashInFlowByCategory(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.GetCashInFlowByCategoryResponse{}, err
	}

	var totalInFlow float64

	for _, cash := range cashInFlow {
		totalInFlow += cash.TotalAmount
	}

	var response []domain.GetCashInFlowByCategoryResponse

	for _, cash := range cashInFlow {
		var totalPercentage float64

		if totalInFlow > 0 {
			totalPercentage = (cash.TotalAmount / totalInFlow) * 100
		}

		response = append(response, domain.GetCashInFlowByCategoryResponse{
			NameCategory:     cash.CategoryName,
			TotalInFlow:      cash.TotalAmount,
			PercentageInFlow: totalPercentage,
		})

	}

	return response, nil
}

func (s *Service) GetCashOutFlowByCategory(ctx context.Context, companyId uuid.UUID) ([]domain.GetCashOutFlowByCategoryResponse, error) {
	cashOutFlow, err := s.repo.GetCashOutFlowByCategory(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return []domain.GetCashOutFlowByCategoryResponse{}, err
	}

	var totalOutFlow float64

	for _, cash := range cashOutFlow {
		totalOutFlow += cash.TotalAmount
	}

	var response []domain.GetCashOutFlowByCategoryResponse

	var totalPercentage float64
	for _, cash := range cashOutFlow {
		totalPercentage = (cash.TotalAmount / totalOutFlow) * 100

		response = append(response, domain.GetCashOutFlowByCategoryResponse{
			NameCategory:     cash.CategoryName,
			TotalOutFlow:     cash.TotalAmount,
			PercentageInFlow: totalPercentage,
		})
	}

	return response, nil
}

func (s *Service) GetCashFlowPeriod(ctx context.Context, companyId uuid.UUID) ([]domain.GetCashFlowPeriodResponse, error) {
	dayBase := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)

	var response []domain.GetCashFlowPeriodResponse

	for i := 1; i <= 3; i++ {
		var startAt time.Time
		var endAt time.Time
		var dateString string

		if i == 1 {
			startAt = dayBase
			endAt = startAt.AddDate(0, 1, 0).Add(-time.Nanosecond)
			dateString = "current month"
		} else if i == 2 {
			startAt = dayBase.AddDate(0, -1, 0)
			endAt = startAt.AddDate(0, 1, 0).Add(-time.Nanosecond)
			dateString = "last month"
		} else {
			startAt = dayBase.AddDate(-1, 0, 0)
			endAt = startAt.AddDate(0, 0, 0).Add(-time.Nanosecond)
			dateString = "last year"
		}

		totalInFlow, err := s.repo.GetTotalInflowByPeriod(ctx, db.GetTotalInflowByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
		})
		if err != nil {
			return []domain.GetCashFlowPeriodResponse{}, err
		}

		totalOutFlow, err := s.repo.GetTotalOutflowByPeriod(ctx, db.GetTotalOutflowByPeriodParams{
			CompanyID:     pgconv.ParseUUIDToPgType(companyId),
			PaymentDate:   pgconv.StringToPgDate(startAt.GoString()),
			PaymentDate_2: pgconv.StringToPgDate(endAt.GoString()),
		})
		if err != nil {
			return []domain.GetCashFlowPeriodResponse{}, err
		}

		response = append(response, domain.GetCashFlowPeriodResponse{
			Month:        dateString,
			TotalInflow:  totalInFlow,
			TotalOutflow: totalOutFlow,
		})

	}

	return response, nil
}

func (s *Service) GetCashFlow(ctx context.Context, companyId uuid.UUID) ([]domain.GetCashFlowResponse, error) {
	dayBase := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)

	var response []domain.GetCashFlowResponse

	var accumulatedBalance float64

	for i := 6; i >= 0; i-- {
		startAt := dayBase.AddDate(0, -i, 0)

		endAt := startAt.AddDate(0, 1, 0).Add(-time.Nanosecond)

		totalInFlow, err := s.repo.GetTotalInflowByPeriod(ctx, db.GetTotalInflowByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
		})
		if err != nil {
			return []domain.GetCashFlowResponse{}, err
		}

		totalOutFlow, err := s.repo.GetTotalOutflowByPeriod(ctx, db.GetTotalOutflowByPeriodParams{
			CompanyID:     pgconv.ParseUUIDToPgType(companyId),
			PaymentDate:   pgconv.StringToPgDate(startAt.GoString()),
			PaymentDate_2: pgconv.StringToPgDate(endAt.GoString()),
		})
		if err != nil {
			return []domain.GetCashFlowResponse{}, err
		}

		accumulatedBalance += totalInFlow - totalOutFlow

		response = append(response, domain.GetCashFlowResponse{
			Date:         startAt.Format("01/01/2006"),
			TotalInflow:  totalInFlow,
			TotalOutflow: totalOutFlow,
		})

	}

	return response, nil
}
