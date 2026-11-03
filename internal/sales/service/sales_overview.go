package service

import (
	"context"
	"fmt"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/sales/domain"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const overviewDateLayout = "2006-01-02"

// civilDate devolve a data do calendário local como meia-noite UTC: o pgx grava
// pgtype.Date pelo ano/mês/dia do time, então o fuso não pode deslocar o dia.
func civilDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// companyLocation fuso da empresa; cai em São Paulo se o cadastro tiver um inválido
func (s *Service) companyLocation(ctx context.Context, companyId uuid.UUID) (string, *time.Location, error) {
	tz, err := s.repo.GetCompanyTimezone(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return "", nil, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		tz = "America/Sao_Paulo"
		loc, _ = time.LoadLocation(tz)
	}
	return tz, loc, nil
}

func overviewTotals(row db.GetSalesOverviewTotalsRow) domain.SalesOverviewTotals {
	return domain.SalesOverviewTotals{
		SalesCount:        int(row.SalesCount),
		Revenue:           row.Revenue,
		AverageTicket:     domain.AverageTicket(row.Revenue, int(row.SalesCount)),
		Discounts:         row.Discounts,
		ItemsSold:         int(row.ItemsSold),
		CustomersCount:    int(row.CustomersCount),
		CanceledCount:     int(row.CanceledCount),
		CanceledAmount:    row.CanceledAmount,
		ReturnedAmount:    row.ReturnedAmount,
		OpenReceivable:    row.OpenReceivable,
		OverdueReceivable: row.OverdueReceivable,
	}
}

// GetSalesOverview monta os cards da tela de vendas para o período, com a janela
// anterior de mesmo tamanho para comparação. Sem start_date o período começa na
// primeira venda da empresa e não há comparação. As consultas rodam em paralelo.
func (s *Service) GetSalesOverview(ctx context.Context, companyId uuid.UUID, params domain.GetSalesOverviewParams) (domain.GetSalesOverviewResponse, error) {
	tz, loc, err := s.companyLocation(ctx, companyId)
	if err != nil {
		return domain.GetSalesOverviewResponse{}, err
	}

	companyPg := pgconv.ParseUUIDToPgType(companyId)

	end := civilDate(time.Now().In(loc))
	if params.EndDate != "" {
		if end, err = time.Parse(overviewDateLayout, params.EndDate); err != nil {
			return domain.GetSalesOverviewResponse{}, fmt.Errorf("%w: end_date inválido", domain.ErrInvalidOverviewParams)
		}
	}

	allTime := params.StartDate == ""
	start := end
	if allTime {
		first, err := s.repo.GetFirstSaleDate(ctx, db.GetFirstSaleDateParams{CompanyID: companyPg, Tz: tz})
		if err != nil {
			return domain.GetSalesOverviewResponse{}, err
		}
		if first.Valid && first.Time.Before(end) {
			start = first.Time
		}
	} else {
		if start, err = time.Parse(overviewDateLayout, params.StartDate); err != nil {
			return domain.GetSalesOverviewResponse{}, fmt.Errorf("%w: start_date inválido", domain.ErrInvalidOverviewParams)
		}
		if days := int(end.Sub(start).Hours()/24) + 1; days > domain.OverviewMaxDays {
			return domain.GetSalesOverviewResponse{}, fmt.Errorf("%w: período máximo é de %d dias", domain.ErrInvalidOverviewParams, domain.OverviewMaxDays)
		}
	}

	w, err := domain.BuildOverviewWindow(start, end)
	if err != nil {
		return domain.GetSalesOverviewResponse{}, err
	}

	startPg, endPg := pgconv.ToPgDate(w.Start), pgconv.ToPgDate(w.End)

	var (
		totals   db.GetSalesOverviewTotalsRow
		previous db.GetSalesOverviewTotalsRow
		series   []db.GetSalesOverviewSeriesRow
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		totals, err = s.repo.GetSalesOverviewTotals(gctx, db.GetSalesOverviewTotalsParams{
			CompanyID: companyPg, StartDate: startPg, EndDate: endPg, Tz: tz,
		})
		return err
	})
	// Antes da primeira venda não há nada para comparar
	if !allTime {
		g.Go(func() (err error) {
			previous, err = s.repo.GetSalesOverviewTotals(gctx, db.GetSalesOverviewTotalsParams{
				CompanyID: companyPg, StartDate: pgconv.ToPgDate(w.PrevStart), EndDate: startPg, Tz: tz,
			})
			return err
		})
	}
	g.Go(func() (err error) {
		series, err = s.repo.GetSalesOverviewSeries(gctx, db.GetSalesOverviewSeriesParams{
			CompanyID:   companyPg,
			StartDate:   startPg,
			EndDate:     endPg,
			FirstBucket: pgconv.ToPgDate(w.FirstBucket),
			LastBucket:  pgconv.ToPgDate(w.LastBucket),
			Step:        w.Step,
			Unit:        w.Unit,
			Tz:          tz,
		})
		return err
	})
	if err := g.Wait(); err != nil {
		return domain.GetSalesOverviewResponse{}, err
	}

	response := domain.GetSalesOverviewResponse{
		StartDate: w.Start.Format(overviewDateLayout),
		EndDate:   end.Format(overviewDateLayout),
		Period:    w.Unit,
		Totals:    overviewTotals(totals),
		Previous:  overviewTotals(previous),
		Series:    make([]domain.SalesOverviewPoint, 0, len(series)),
	}

	for _, row := range series {
		response.Series = append(response.Series, domain.SalesOverviewPoint{
			Date:       pgconv.PgDateToString(row.Bucket),
			SalesCount: int(row.SalesCount),
			Revenue:    row.Revenue,
		})
	}

	return response, nil
}
