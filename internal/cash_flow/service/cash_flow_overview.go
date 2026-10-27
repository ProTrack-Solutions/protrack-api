package service

import (
	"context"
	"fmt"
	"math"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/cash_flow/domain"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	overviewDefaultPeriod   = "day"
	overviewDefaultQuantity = 30
	overviewDefaultForecast = 30
	overviewRecentLimit     = 8
	dateLayout              = "2006-01-02"
)

// Limite de períodos por granularidade: ~1 ano de dias, 2 anos de semanas, 3 de meses
var overviewMaxQuantity = map[string]int{"day": 366, "week": 104, "month": 36}

// overviewWindow intervalo [start, end) em datas locais da empresa
type overviewWindow struct {
	start      time.Time
	end        time.Time
	lastBucket time.Time
	prevStart  time.Time
	step       string
}

// civilDate devolve a data do calendário local como meia-noite UTC: o pgx grava
// pgtype.Date pelo ano/mês/dia do time, então o fuso não pode deslocar o dia.
func civilDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func buildOverviewWindow(period string, quantity int, today time.Time) overviewWindow {
	var current, start, end, prevStart time.Time
	var step string

	switch period {
	case "week":
		// Semana começando na segunda, igual ao date_trunc('week') do Postgres
		current = today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
		start = current.AddDate(0, 0, -7*(quantity-1))
		end = current.AddDate(0, 0, 7)
		prevStart = start.AddDate(0, 0, -7*quantity)
		step = "1 week"
	case "month":
		current = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		start = current.AddDate(0, -(quantity - 1), 0)
		end = current.AddDate(0, 1, 0)
		prevStart = start.AddDate(0, -quantity, 0)
		step = "1 month"
	default:
		current = today
		start = current.AddDate(0, 0, -(quantity - 1))
		end = current.AddDate(0, 0, 1)
		prevStart = start.AddDate(0, 0, -quantity)
		step = "1 day"
	}

	return overviewWindow{start: start, end: end, lastBucket: current, prevStart: prevStart, step: step}
}

func share(name string, total, sum float64, count int) domain.OverviewShare {
	var percentage float64
	if sum > 0 {
		percentage = math.Round(total/sum*10000) / 100
	}
	return domain.OverviewShare{Name: name, Total: total, Percentage: percentage, Count: count}
}

// GetOverview monta toda a tela de fluxo de caixa. As consultas são agregadas
// no banco e rodam em paralelo: o custo não cresce com a quantidade de períodos.
func (s *Service) GetOverview(ctx context.Context, companyId uuid.UUID, params domain.GetOverviewParams) (domain.GetOverviewResponse, error) {
	period := params.Period
	if period == "" {
		period = overviewDefaultPeriod
	}
	quantity := params.Quantity
	if quantity == 0 {
		quantity = overviewDefaultQuantity
	}
	if limit := overviewMaxQuantity[period]; quantity > limit {
		return domain.GetOverviewResponse{}, fmt.Errorf("%w: quantity máximo para %s é %d", domain.ErrInvalidOverviewParams, period, limit)
	}
	forecastDays := params.ForecastDays
	if forecastDays == 0 {
		forecastDays = overviewDefaultForecast
	}

	companyPg := pgconv.ParseUUIDToPgType(companyId)

	tz, err := s.repo.GetCompanyTimezone(ctx, companyPg)
	if err != nil {
		return domain.GetOverviewResponse{}, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		tz = "America/Sao_Paulo"
		loc, _ = time.LoadLocation(tz)
	}

	today := civilDate(time.Now().In(loc))
	w := buildOverviewWindow(period, quantity, today)
	forecastEnd := today.AddDate(0, 0, forecastDays)

	var (
		series      []db.GetCashFlowSeriesRow
		previous    db.GetCashFlowTotalsByRangeRow
		inCategory  []db.GetCashInFlowCategoryByRangeRow
		outCategory []db.GetCashOutFlowCategoryByRangeRow
		methods     []db.GetCashInFlowPaymentMethodByRangeRow
		forecast    []db.GetCashFlowForecastRow
		overdue     db.GetCashFlowOverdueRow
		recent      []db.ListRecentCashMovementsRow
	)

	start, end := pgconv.ToPgDate(w.start), pgconv.ToPgDate(w.end)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		series, err = s.repo.GetCashFlowSeries(gctx, db.GetCashFlowSeriesParams{
			CompanyID:  companyPg,
			StartDate:  start,
			EndDate:    end,
			LastBucket: pgconv.ToPgDate(w.lastBucket),
			Step:       w.step,
			Unit:       period,
			Tz:         tz,
		})
		return err
	})
	g.Go(func() (err error) {
		previous, err = s.repo.GetCashFlowTotalsByRange(gctx, db.GetCashFlowTotalsByRangeParams{
			CompanyID: companyPg,
			StartDate: pgconv.ToPgDate(w.prevStart),
			EndDate:   start,
			Tz:        tz,
		})
		return err
	})
	g.Go(func() (err error) {
		inCategory, err = s.repo.GetCashInFlowCategoryByRange(gctx, db.GetCashInFlowCategoryByRangeParams{
			CompanyID: companyPg, StartDate: start, EndDate: end, Tz: tz,
		})
		return err
	})
	g.Go(func() (err error) {
		outCategory, err = s.repo.GetCashOutFlowCategoryByRange(gctx, db.GetCashOutFlowCategoryByRangeParams{
			CompanyID: companyPg, StartDate: start, EndDate: end,
		})
		return err
	})
	g.Go(func() (err error) {
		methods, err = s.repo.GetCashInFlowPaymentMethodByRange(gctx, db.GetCashInFlowPaymentMethodByRangeParams{
			CompanyID: companyPg, StartDate: start, EndDate: end, Tz: tz,
		})
		return err
	})
	g.Go(func() (err error) {
		forecast, err = s.repo.GetCashFlowForecast(gctx, db.GetCashFlowForecastParams{
			CompanyID: companyPg,
			StartDate: pgconv.ToPgDate(today),
			EndDate:   pgconv.ToPgDate(forecastEnd),
		})
		return err
	})
	g.Go(func() (err error) {
		overdue, err = s.repo.GetCashFlowOverdue(gctx, db.GetCashFlowOverdueParams{
			CompanyID: companyPg,
			Today:     pgconv.ToPgDate(today),
		})
		return err
	})
	g.Go(func() (err error) {
		recent, err = s.repo.ListRecentCashMovements(gctx, db.ListRecentCashMovementsParams{
			CompanyID: companyPg,
			RowLimit:  overviewRecentLimit,
			Tz:        tz,
		})
		return err
	})
	if err := g.Wait(); err != nil {
		return domain.GetOverviewResponse{}, err
	}

	response := domain.GetOverviewResponse{
		Period:                period,
		Quantity:              quantity,
		StartDate:             w.start.Format(dateLayout),
		EndDate:               w.end.AddDate(0, 0, -1).Format(dateLayout),
		Series:                make([]domain.OverviewPoint, 0, len(series)),
		InflowByCategory:      make([]domain.OverviewShare, 0, len(inCategory)),
		OutflowByCategory:     make([]domain.OverviewShare, 0, len(outCategory)),
		InflowByPaymentMethod: make([]domain.OverviewShare, 0, len(methods)),
		RecentMovements:       make([]domain.OverviewMovement, 0, len(recent)),
		Previous: domain.OverviewTotals{
			Inflow:  previous.Inflow,
			Outflow: previous.Outflow,
			Net:     previous.Inflow - previous.Outflow,
		},
	}

	nets := make([]float64, 0, len(series))
	var balance float64
	for _, row := range series {
		net := row.Inflow - row.Outflow
		balance += net
		nets = append(nets, net)

		response.Totals.Inflow += row.Inflow
		response.Totals.Outflow += row.Outflow
		response.Totals.InflowCount += int(row.InflowCount)
		response.Totals.OutflowCount += int(row.OutflowCount)

		response.Series = append(response.Series, domain.OverviewPoint{
			Date:    pgconv.PgDateToString(row.Bucket),
			Inflow:  row.Inflow,
			Outflow: row.Outflow,
			Net:     net,
			Balance: balance,
		})
	}
	response.Totals.Net = response.Totals.Inflow - response.Totals.Outflow

	// Regressão linear precisa de ao menos dois pontos; com um só repete o valor
	if len(nets) >= 2 {
		response.Projection = domain.CalcularProjecao(nets, len(nets)+1)
	} else if len(nets) == 1 {
		response.Projection = nets[0]
	}

	var sum float64
	for _, row := range inCategory {
		sum += row.TotalAmount
	}
	for _, row := range inCategory {
		response.InflowByCategory = append(response.InflowByCategory, share(row.CategoryName, row.TotalAmount, sum, 0))
	}

	sum = 0
	for _, row := range outCategory {
		sum += row.TotalAmount
	}
	for _, row := range outCategory {
		response.OutflowByCategory = append(response.OutflowByCategory, share(row.CategoryName, row.TotalAmount, sum, 0))
	}

	sum = 0
	for _, row := range methods {
		sum += row.TotalAmount
	}
	for _, row := range methods {
		response.InflowByPaymentMethod = append(response.InflowByPaymentMethod, share(row.PaymentMethod, row.TotalAmount, sum, int(row.SalesCount)))
	}

	response.Forecast = domain.OverviewForecast{
		Days:              forecastDays,
		OverdueReceivable: overdue.Receivable,
		OverduePayable:    overdue.Payable,
		Series:            make([]domain.OverviewForecastPoint, 0, len(forecast)),
	}
	var forecastBalance float64
	for _, row := range forecast {
		forecastBalance += row.Receivable - row.Payable
		response.Forecast.Receivable += row.Receivable
		response.Forecast.Payable += row.Payable
		response.Forecast.Series = append(response.Forecast.Series, domain.OverviewForecastPoint{
			Date:       pgconv.PgDateToString(row.Day),
			Receivable: row.Receivable,
			Payable:    row.Payable,
			Balance:    forecastBalance,
		})
	}
	response.Forecast.Net = response.Forecast.Receivable - response.Forecast.Payable

	for _, row := range recent {
		response.RecentMovements = append(response.RecentMovements, domain.OverviewMovement{
			ID:          pgconv.PgUUIDToUUID(row.ID).String(),
			Kind:        row.Kind,
			Description: row.Description,
			Detail:      row.Detail,
			Amount:      row.Amount,
			OccurredAt:  pgconv.PgTimestamptzToTime(row.OccurredAt),
		})
	}

	return response, nil
}
