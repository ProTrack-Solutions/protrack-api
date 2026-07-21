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
	GetCashOutFlowCategoryByPeriod(ctx context.Context, arg db.GetCashOutFlowCategoryByPeriodParams) ([]db.GetCashOutFlowCategoryByPeriodRow, error)
	GetCashInFlowCategoryByPeriod(ctx context.Context, arg db.GetCashInFlowCategoryByPeriodParams) ([]db.GetCashInFlowCategoryByPeriodRow, error)
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

		endAt := startAt.AddDate(0, 1, 0)

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
			PaymentDate:   pgconv.ToPgDate(startAt),
			PaymentDate_2: pgconv.ToPgDate(endAt),
		})
		if err != nil {
			return []domain.GetCashFlowResponse{}, err
		}

		accumulatedBalance += totalInFlow - totalOutFlow

		response = append(response, domain.GetCashFlowResponse{
			Date:         startAt.Format("01/2006"),
			TotalInflow:  totalInFlow,
			TotalOutflow: totalOutFlow,
		})

	}

	return response, nil
}

func (s *Service) GetTotalSummary(ctx context.Context, req domain.GetTotalSummaryParams, companyId uuid.UUID) (domain.GetTotalSummaryResponse, error) {
	dayBase := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

	var response []domain.TotalSummaty
	var responseCategoryInFlow []domain.GetCashInFlowByCategoryResponse
	var responseCategoryOutFlow []domain.GetCashOutFlowByCategoryResponse

	var totalPeriod float64

	var totalOutFlow float64
	var totalInFlow float64

	switch req.Period {
	case "day":
		quantity := int(req.Quantity)

		for i := 0; i < quantity; i++ {
			offset := quantity - 1 - i
			startAt := dayBase.AddDate(0, 0, -offset)

			endAt := startAt.AddDate(0, 0, 1)

			totalPeriodInFlow, err := s.repo.GetTotalInflowByPeriod(ctx, db.GetTotalInflowByPeriodParams{
				CompanyID:   pgconv.ParseUUIDToPgType(companyId),
				CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
				CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
			})
			if err != nil {
				return domain.GetTotalSummaryResponse{}, err
			}

			totalPeriodOutFlow, err := s.repo.GetTotalOutflowByPeriod(ctx, db.GetTotalOutflowByPeriodParams{
				CompanyID:     pgconv.ParseUUIDToPgType(companyId),
				PaymentDate:   pgconv.ToPgDate(startAt),
				PaymentDate_2: pgconv.ToPgDate(startAt),
			})
			if err != nil {
				return domain.GetTotalSummaryResponse{}, err
			}

			totalPeriod += totalPeriodInFlow - totalPeriodOutFlow

			totalOutFlow += totalPeriodOutFlow
			totalInFlow += totalPeriodInFlow

			response = append(response, domain.TotalSummaty{
				Period:             startAt.Format("02/01/2006"),
				TotalPeriodOutFlow: totalPeriodOutFlow,
				TotalPeriodInFlow:  totalPeriodInFlow,
				TotalPeriod:        totalPeriod,
			})

		}

		initialAt := dayBase

		finishAt := dayBase.AddDate(0, 0, -int(req.Quantity))

		totalCategoriesInFlow, err := s.repo.GetCashInFlowCategoryByPeriod(ctx, db.GetCashInFlowCategoryByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(initialAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(finishAt),
		})
		if err != nil {
			return domain.GetTotalSummaryResponse{}, nil
		}

		totalCategoriesOutFlow, err := s.repo.GetCashOutFlowCategoryByPeriod(ctx, db.GetCashOutFlowCategoryByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(finishAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(initialAt),
		})
		if err != nil {
			return domain.GetTotalSummaryResponse{}, nil
		}

		var totalAmountCategoriesInFlow float64

		for _, cashIn := range totalCategoriesInFlow {
			totalAmountCategoriesInFlow += cashIn.TotalAmount
		}

		var totalAmountCategoriesOutFlow float64

		for _, cashOut := range totalCategoriesOutFlow {
			totalAmountCategoriesOutFlow += cashOut.TotalAmount
		}

		for _, inCategory := range totalCategoriesInFlow {

			var totalPercentage float64

			if totalAmountCategoriesInFlow > 0 {
				totalPercentage = (inCategory.TotalAmount / totalInFlow) * 100
			}

			responseCategoryInFlow = append(responseCategoryInFlow, domain.GetCashInFlowByCategoryResponse{
				NameCategory:     inCategory.CategoryName,
				TotalInFlow:      inCategory.TotalAmount,
				PercentageInFlow: totalPercentage,
			})
		}

		for _, outCategory := range totalCategoriesOutFlow {
			var totalPercentage float64

			if totalAmountCategoriesOutFlow > 0 {
				totalPercentage = (outCategory.TotalAmount / totalOutFlow) * 100
			}

			responseCategoryOutFlow = append(responseCategoryOutFlow, domain.GetCashOutFlowByCategoryResponse{
				NameCategory:     outCategory.CategoryName,
				TotalOutFlow:     outCategory.TotalAmount,
				PercentageInFlow: totalPercentage,
			})
		}

	case "week":
		weedDay := dayBase.Weekday()
		currentWeekStart := dayBase.AddDate(0, 0, -int(weedDay))

		quantity := int(req.Quantity)

		for i := 0; i < quantity; i++ {
			offset := quantity - 1 - i
			startAt := currentWeekStart.AddDate(0, 0, -offset*7)
			endAt := startAt.AddDate(0, 0, 7)

			totalPeriodInFlow, err := s.repo.GetTotalInflowByPeriod(ctx, db.GetTotalInflowByPeriodParams{
				CompanyID:   pgconv.ParseUUIDToPgType(companyId),
				CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
				CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
			})
			if err != nil {
				return domain.GetTotalSummaryResponse{}, err
			}

			totalPeriodOutFlow, err := s.repo.GetTotalOutflowByPeriod(ctx, db.GetTotalOutflowByPeriodParams{
				CompanyID:     pgconv.ParseUUIDToPgType(companyId),
				PaymentDate:   pgconv.ToPgDate(startAt),
				PaymentDate_2: pgconv.ToPgDate(endAt.AddDate(0, 0, -1)), // Sábado (inclusivo)
			})
			if err != nil {
				return domain.GetTotalSummaryResponse{}, err
			}

			totalPeriod += totalPeriodInFlow - totalPeriodOutFlow

			totalOutFlow += totalPeriodOutFlow
			totalInFlow += totalPeriodInFlow

			response = append(response, domain.TotalSummaty{
				Period:             startAt.Format("02/01/2006"),
				TotalPeriodOutFlow: totalPeriodOutFlow,
				TotalPeriodInFlow:  totalPeriodInFlow,
				TotalPeriod:        totalPeriod,
			})

		}

		initialAt := currentWeekStart

		finishAt := currentWeekStart.AddDate(0, 0, -int(req.Quantity))

		totalCategoriesInFlow, err := s.repo.GetCashInFlowCategoryByPeriod(ctx, db.GetCashInFlowCategoryByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(initialAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(finishAt),
		})
		if err != nil {
			return domain.GetTotalSummaryResponse{}, nil
		}

		totalCategoriesOutFlow, err := s.repo.GetCashOutFlowCategoryByPeriod(ctx, db.GetCashOutFlowCategoryByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(finishAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(initialAt),
		})
		if err != nil {
			return domain.GetTotalSummaryResponse{}, nil
		}

		var totalAmountCategoriesInFlow float64

		for _, cashIn := range totalCategoriesInFlow {
			totalAmountCategoriesInFlow += cashIn.TotalAmount
		}

		var totalAmountCategoriesOutFlow float64

		for _, cashOut := range totalCategoriesOutFlow {
			totalAmountCategoriesOutFlow += cashOut.TotalAmount
		}

		for _, inCategory := range totalCategoriesInFlow {

			var totalPercentage float64

			if totalAmountCategoriesInFlow > 0 {
				totalPercentage = (inCategory.TotalAmount / totalInFlow) * 100
			}

			responseCategoryInFlow = append(responseCategoryInFlow, domain.GetCashInFlowByCategoryResponse{
				NameCategory:     inCategory.CategoryName,
				TotalInFlow:      inCategory.TotalAmount,
				PercentageInFlow: totalPercentage,
			})
		}

		for _, outCategory := range totalCategoriesOutFlow {
			var totalPercentage float64

			if totalAmountCategoriesOutFlow > 0 {
				totalPercentage = (outCategory.TotalAmount / totalOutFlow) * 100
			}

			responseCategoryOutFlow = append(responseCategoryOutFlow, domain.GetCashOutFlowByCategoryResponse{
				NameCategory:     outCategory.CategoryName,
				TotalOutFlow:     outCategory.TotalAmount,
				PercentageInFlow: totalPercentage,
			})
		}

	case "month":
		quantity := int(req.Quantity)

		for i := 0; i < quantity; i++ {
			offset := quantity - 1 - i
			startAt := dayBase.AddDate(0, -offset, 0)
			endAt := startAt.AddDate(0, 1, 0)

			totalPeriodInFlow, err := s.repo.GetTotalInflowByPeriod(ctx, db.GetTotalInflowByPeriodParams{
				CompanyID:   pgconv.ParseUUIDToPgType(companyId),
				CreatedAt:   pgconv.TimeToPgTimestamptz(startAt),
				CreatedAt_2: pgconv.TimeToPgTimestamptz(endAt),
			})
			if err != nil {
				return domain.GetTotalSummaryResponse{}, err
			}

			totalPeriodOutFlow, err := s.repo.GetTotalOutflowByPeriod(ctx, db.GetTotalOutflowByPeriodParams{
				CompanyID:     pgconv.ParseUUIDToPgType(companyId),
				PaymentDate:   pgconv.ToPgDate(startAt),
				PaymentDate_2: pgconv.ToPgDate(endAt),
			})
			if err != nil {
				return domain.GetTotalSummaryResponse{}, err
			}

			totalPeriod += totalPeriodInFlow - totalPeriodOutFlow

			totalOutFlow += totalPeriodOutFlow
			totalInFlow += totalPeriodInFlow

			response = append(response, domain.TotalSummaty{
				Period:             startAt.Format("01/2006"),
				TotalPeriodOutFlow: totalPeriodOutFlow,
				TotalPeriodInFlow:  totalPeriodInFlow,
				TotalPeriod:        totalPeriod,
			})
		}

		initialAt := dayBase

		finishAt := dayBase.AddDate(0, -int(req.Quantity), 0)

		totalCategoriesInFlow, err := s.repo.GetCashInFlowCategoryByPeriod(ctx, db.GetCashInFlowCategoryByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(initialAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(finishAt),
		})
		if err != nil {
			return domain.GetTotalSummaryResponse{}, nil
		}

		totalCategoriesOutFlow, err := s.repo.GetCashOutFlowCategoryByPeriod(ctx, db.GetCashOutFlowCategoryByPeriodParams{
			CompanyID:   pgconv.ParseUUIDToPgType(companyId),
			CreatedAt:   pgconv.TimeToPgTimestamptz(finishAt),
			CreatedAt_2: pgconv.TimeToPgTimestamptz(initialAt),
		})
		if err != nil {
			return domain.GetTotalSummaryResponse{}, nil
		}

		var totalAmountCategoriesInFlow float64

		for _, cashIn := range totalCategoriesInFlow {
			totalAmountCategoriesInFlow += cashIn.TotalAmount
		}

		var totalAmountCategoriesOutFlow float64

		for _, cashOut := range totalCategoriesOutFlow {
			totalAmountCategoriesOutFlow += cashOut.TotalAmount
		}

		for _, inCategory := range totalCategoriesInFlow {

			var totalPercentage float64

			if totalAmountCategoriesInFlow > 0 {
				totalPercentage = (inCategory.TotalAmount / totalInFlow) * 100
			}

			responseCategoryInFlow = append(responseCategoryInFlow, domain.GetCashInFlowByCategoryResponse{
				NameCategory:     inCategory.CategoryName,
				TotalInFlow:      inCategory.TotalAmount,
				PercentageInFlow: totalPercentage,
			})
		}

		for _, outCategory := range totalCategoriesOutFlow {
			var totalPercentage float64

			if totalAmountCategoriesOutFlow > 0 {
				totalPercentage = (outCategory.TotalAmount / totalOutFlow) * 100
			}

			responseCategoryOutFlow = append(responseCategoryOutFlow, domain.GetCashOutFlowByCategoryResponse{
				NameCategory:     outCategory.CategoryName,
				TotalOutFlow:     outCategory.TotalAmount,
				PercentageInFlow: totalPercentage,
			})
		}
	}

	return domain.GetTotalSummaryResponse{
		Summary:                response,
		TotalOutFlow:           totalOutFlow,
		TotalInFlow:            totalInFlow,
		Total:                  totalPeriod,
		TotalCategoriesInFlow:  responseCategoryInFlow,
		TotalCategoriesOutFlow: responseCategoryOutFlow,
	}, nil
}
