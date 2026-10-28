package service

import (
	"context"
	"fmt"
	"math"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	dateLayout      = "2006-01-02"
	defaultTimezone = "America/Sao_Paulo"
	// Período máximo aceito: ~2 anos, o suficiente para comparar exercícios
	maxPeriodDays = 731
)

type RepositoryInterface interface {
	ReportCompanyHeader(ctx context.Context, companyId pgtype.UUID) (db.ReportCompanyHeaderRow, error)
	ReportSales(ctx context.Context, arg db.ReportSalesParams) ([]db.ReportSalesRow, error)
	ReportSalesByDay(ctx context.Context, arg db.ReportSalesByDayParams) ([]db.ReportSalesByDayRow, error)
	ReportSalesByPaymentMethod(ctx context.Context, arg db.ReportSalesByPaymentMethodParams) ([]db.ReportSalesByPaymentMethodRow, error)
	ReportSalesBySeller(ctx context.Context, arg db.ReportSalesBySellerParams) ([]db.ReportSalesBySellerRow, error)
	ReportSalesByCustomer(ctx context.Context, arg db.ReportSalesByCustomerParams) ([]db.ReportSalesByCustomerRow, error)
	ReportSalesByProduct(ctx context.Context, arg db.ReportSalesByProductParams) ([]db.ReportSalesByProductRow, error)
	ReportSalesByCategory(ctx context.Context, arg db.ReportSalesByCategoryParams) ([]db.ReportSalesByCategoryRow, error)
	ReportInventory(ctx context.Context, companyId pgtype.UUID) ([]db.ReportInventoryRow, error)
	ReportStockMovements(ctx context.Context, arg db.ReportStockMovementsParams) ([]db.ReportStockMovementsRow, error)
	ReportPaymentsReceived(ctx context.Context, arg db.ReportPaymentsReceivedParams) ([]db.ReportPaymentsReceivedRow, error)
	ReportOpenInstallments(ctx context.Context, arg db.ReportOpenInstallmentsParams) ([]db.ReportOpenInstallmentsRow, error)
	ReportBillsPayable(ctx context.Context, arg db.ReportBillsPayableParams) ([]db.ReportBillsPayableRow, error)
	ReportCashSessions(ctx context.Context, arg db.ReportCashSessionsParams) ([]db.ReportCashSessionsRow, error)
	ReportDreRevenue(ctx context.Context, arg db.ReportDreRevenueParams) (db.ReportDreRevenueRow, error)
	ReportDreExpenses(ctx context.Context, arg db.ReportDreExpensesParams) ([]db.ReportDreExpensesRow, error)
}

type Service struct {
	repo RepositoryInterface
	now  func() time.Time
}

func NewService(repo *repository.Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

// period intervalo do relatório em datas locais da empresa: [start, end)
type period struct {
	companyID pgtype.UUID
	tz        string
	loc       *time.Location
	start     time.Time
	end       time.Time
	today     time.Time
}

func (p period) startDate() pgtype.Date { return pgconv.ToPgDate(p.start) }
func (p period) endDate() pgtype.Date   { return pgconv.ToPgDate(p.end) }
func (p period) todayDate() pgtype.Date { return pgconv.ToPgDate(p.today) }

// builder monta um relatório a partir do período
type builder func(s *Service, ctx context.Context, p period) (domain.Report, error)

var builders = map[domain.ReportType]builder{
	domain.ReportSales:                (*Service).salesReport,
	domain.ReportSalesByDay:           (*Service).salesByDayReport,
	domain.ReportSalesByPaymentMethod: (*Service).salesByPaymentMethodReport,
	domain.ReportSalesBySeller:        (*Service).salesBySellerReport,
	domain.ReportSalesByCustomer:      (*Service).salesByCustomerReport,
	domain.ReportProductsABC:          (*Service).productsABCReport,
	domain.ReportProfitCategory:       (*Service).profitCategoryReport,
	domain.ReportInventory:            (*Service).inventoryReport,
	domain.ReportStockMovements:       (*Service).stockMovementsReport,
	domain.ReportDRE:                  (*Service).dreReport,
	domain.ReportPaymentsReceived:     (*Service).paymentsReceivedReport,
	domain.ReportOpenInstallments:     (*Service).openInstallmentsReport,
	domain.ReportBillsPayable:         (*Service).billsPayableReport,
	domain.ReportCashSessions:         (*Service).cashSessionsReport,
}

// GenerateReport monta o relatório pedido. As datas chegam como YYYY-MM-DD no
// calendário da empresa, ambas inclusivas; sem datas, usa o mês corrente.
func (s *Service) GenerateReport(ctx context.Context, companyId uuid.UUID, req domain.ReportRequest) (domain.Report, error) {
	build, ok := builders[req.Type]
	if !ok {
		return domain.Report{}, fmt.Errorf("%w: %s", domain.ErrReportNotFound, req.Type)
	}

	companyPg := pgconv.ParseUUIDToPgType(companyId)

	company, err := s.repo.ReportCompanyHeader(ctx, companyPg)
	if err != nil {
		return domain.Report{}, err
	}
	tz := company.Timezone
	loc, err := time.LoadLocation(tz)
	if err != nil {
		tz = defaultTimezone
		loc, _ = time.LoadLocation(tz)
	}

	today := civilDate(s.now().In(loc))
	start, end, err := parsePeriod(req.StartDate, req.EndDate, today)
	if err != nil {
		return domain.Report{}, err
	}

	p := period{companyID: companyPg, tz: tz, loc: loc, start: start, end: end, today: today}

	report, err := build(s, ctx, p)
	if err != nil {
		return domain.Report{}, err
	}

	report.Type = req.Type
	report.CompanyName = company.CompanyName
	report.GeneratedAt = s.now().In(loc)
	if report.UsesPeriod {
		report.StartDate = start.Format(dateLayout)
		report.EndDate = end.AddDate(0, 0, -1).Format(dateLayout)
	}
	if report.Rows == nil {
		report.Rows = [][]any{}
	}
	if report.Summary == nil {
		report.Summary = []domain.SummaryItem{}
	}

	return report, nil
}

// parsePeriod valida o período e devolve [start, end) — end já é o dia
// seguinte à data final informada.
func parsePeriod(startStr, endStr string, today time.Time) (time.Time, time.Time, error) {
	start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := today

	if startStr != "" {
		parsed, err := time.Parse(dateLayout, startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: start_date deve estar no formato AAAA-MM-DD", domain.ErrInvalidReportParams)
		}
		start = parsed
	}
	if endStr != "" {
		parsed, err := time.Parse(dateLayout, endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: end_date deve estar no formato AAAA-MM-DD", domain.ErrInvalidReportParams)
		}
		end = parsed
	}

	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: a data final não pode ser anterior à inicial", domain.ErrInvalidReportParams)
	}
	if days := int(end.Sub(start).Hours()/24) + 1; days > maxPeriodDays {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: o período máximo é de %d dias", domain.ErrInvalidReportParams, maxPeriodDays)
	}

	return start, end.AddDate(0, 0, 1), nil
}

// civilDate devolve a data do calendário local como meia-noite UTC: o pgx grava
// pgtype.Date pelo ano/mês/dia do time, então o fuso não pode deslocar o dia.
func civilDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// localTime converte um timestamptz para o fuso da empresa; nulo vira nil
func (p period) localTime(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time.In(p.loc)
}

// localDate devolve a data como meia-noite no fuso da empresa; nula vira nil
func (p period) localDate(d pgtype.Date) any {
	if !d.Valid {
		return nil
	}
	return time.Date(d.Time.Year(), d.Time.Month(), d.Time.Day(), 0, 0, 0, 0, p.loc)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// percent devolve part/total em pontos percentuais (35.2 = 35,2%), 0 se total for 0
func percent(part, total float64) float64 {
	if total == 0 {
		return 0
	}
	return round2(part / total * 100)
}

var paymentMethodLabels = map[string]string{
	"cash":          "Dinheiro",
	"credit_card":   "Cartão de crédito",
	"debit_card":    "Cartão de débito",
	"pix":           "Pix",
	"bank_transfer": "Transferência",
	"installments":  "A prazo",
	"other":         "Outros",
}

var statusLabels = map[string]string{
	"pending":   "Pendente",
	"paid":      "Pago",
	"overdue":   "Vencido",
	"scheduled": "Agendado",
	"canceled":  "Cancelado",
	"partial":   "Parcial",
	"open":      "Aberto",
	"closed":    "Fechado",
}

var stockMovementLabels = map[string]string{
	"ENTRY":             "Entrada",
	"SALE":              "Venda",
	"SALE_CANCEL":       "Cancelamento de venda",
	"ADJUSTMENT":        "Ajuste",
	"PURCHASE":          "Compra",
	"PURCHASE_REVERSAL": "Estorno de compra",
}

func label(labels map[string]string, value string) string {
	if l, ok := labels[value]; ok {
		return l
	}
	return value
}
