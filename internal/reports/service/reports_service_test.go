package service

import (
	"context"
	"errors"
	"testing"
	"time"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// fakeRepo devolve dados fixos e guarda os parâmetros recebidos
type fakeRepo struct {
	RepositoryInterface
	timezone  string
	salesArg  db.ReportSalesParams
	sales     []db.ReportSalesRow
	products  []db.ReportSalesByProductRow
	inventory []db.ReportInventoryRow
	headerErr error
}

func (f *fakeRepo) ReportCompanyHeader(ctx context.Context, companyId pgtype.UUID) (db.ReportCompanyHeaderRow, error) {
	return db.ReportCompanyHeaderRow{CompanyName: "Loja", Timezone: f.timezone}, f.headerErr
}

func (f *fakeRepo) ReportSales(ctx context.Context, arg db.ReportSalesParams) ([]db.ReportSalesRow, error) {
	f.salesArg = arg
	return f.sales, nil
}

func (f *fakeRepo) ReportSalesByProduct(ctx context.Context, arg db.ReportSalesByProductParams) ([]db.ReportSalesByProductRow, error) {
	return f.products, nil
}

func (f *fakeRepo) ReportInventory(ctx context.Context, companyId pgtype.UUID) ([]db.ReportInventoryRow, error) {
	return f.inventory, nil
}

func newTestService(repo RepositoryInterface) *Service {
	return &Service{
		repo: repo,
		// 28/10/2026 01:30 UTC = 27/10/2026 22:30 em São Paulo
		now: func() time.Time { return time.Date(2026, 10, 28, 1, 30, 0, 0, time.UTC) },
	}
}

func day(s string) time.Time {
	t, _ := time.Parse(dateLayout, s)
	return t
}

func TestParsePeriod(t *testing.T) {
	today := day("2026-10-27")

	tests := []struct {
		name      string
		start     string
		end       string
		wantStart string
		wantEnd   string
		wantErr   bool
	}{
		{"sem datas usa o mês corrente até hoje", "", "", "2026-10-01", "2026-10-28", false},
		{"datas inclusivas", "2026-09-01", "2026-09-30", "2026-09-01", "2026-10-01", false},
		{"um único dia", "2026-10-05", "2026-10-05", "2026-10-05", "2026-10-06", false},
		{"final antes do início", "2026-10-10", "2026-10-01", "", "", true},
		{"formato inválido", "01/10/2026", "", "", "", true},
		{"período acima do limite", "2024-01-01", "2026-10-01", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parsePeriod(tt.start, tt.end, today)
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidReportParams) {
					t.Fatalf("esperava ErrInvalidReportParams, veio %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got := start.Format(dateLayout); got != tt.wantStart {
				t.Errorf("start: esperava %s, veio %s", tt.wantStart, got)
			}
			if got := end.Format(dateLayout); got != tt.wantEnd {
				t.Errorf("end: esperava %s, veio %s", tt.wantEnd, got)
			}
		})
	}
}

func TestAbcClass(t *testing.T) {
	tests := []struct {
		before float64
		want   string
	}{
		{0, "A"},
		{79.99, "A"}, // o item que cruza os 80% ainda é A
		{80, "B"},
		{94.9, "B"},
		{95, "C"},
	}
	for _, tt := range tests {
		if got := abcClass(tt.before); got != tt.want {
			t.Errorf("abcClass(%v): esperava %s, veio %s", tt.before, tt.want, got)
		}
	}
}

func TestBuildDRE(t *testing.T) {
	report := buildDRE(
		db.ReportDreRevenueRow{SalesCount: 4, Gross: 438, Discount: 10, Net: 428, Cost: 281.5},
		[]db.ReportDreExpensesRow{{CategoryName: "Aluguel", Total: 50}, {CategoryName: "Energia", Total: 20}},
	)

	last := report.Rows[len(report.Rows)-1]
	if last[0] != "(=) Resultado do período" || last[1] != 76.5 {
		t.Fatalf("resultado: esperava 76.5, veio %v", last)
	}
	if len(report.Rows) != 10 {
		t.Fatalf("esperava 10 linhas (8 fixas + 2 categorias), veio %d", len(report.Rows))
	}

	summary := map[string]float64{}
	for _, s := range report.Summary {
		summary[s.Label] = s.Value
	}
	if summary["Lucro bruto"] != 146.5 || summary["Margem bruta"] != 34.23 || summary["Margem líquida"] != 17.87 {
		t.Errorf("indicadores inesperados: %v", summary)
	}
}

func TestBuildDREComDevolucoes(t *testing.T) {
	// Receita 428 - devoluções 28 = 400; CMV 281.5 - 18.5 (peças que voltaram) = 263
	report := buildDRE(
		db.ReportDreRevenueRow{SalesCount: 4, Gross: 438, Discount: 10, Net: 428, Cost: 281.5, ReturnsTotal: 28, ReturnsCost: 18.5},
		nil,
	)

	summary := map[string]float64{}
	for _, s := range report.Summary {
		summary[s.Label] = s.Value
	}
	if summary["Receita líquida"] != 400 || summary["Lucro bruto"] != 137 {
		t.Errorf("indicadores inesperados: %v", summary)
	}
	if report.Rows[2][0] != "(-) Devoluções" || report.Rows[2][1] != -28.0 {
		t.Errorf("linha de devoluções: %v", report.Rows[2])
	}
}

func TestBuildDRESemVendas(t *testing.T) {
	report := buildDRE(db.ReportDreRevenueRow{}, nil)
	for _, s := range report.Summary {
		if s.Value != 0 {
			t.Errorf("%s: esperava 0 sem vendas, veio %v", s.Label, s.Value)
		}
	}
}

func TestGenerateReportTipoInexistente(t *testing.T) {
	s := newTestService(&fakeRepo{timezone: "America/Sao_Paulo"})
	_, err := s.GenerateReport(context.Background(), uuid.New(), domain.ReportRequest{Type: "nao_existe"})
	if !errors.Is(err, domain.ErrReportNotFound) {
		t.Fatalf("esperava ErrReportNotFound, veio %v", err)
	}
}

func TestGenerateReportUsaDataLocalDaEmpresa(t *testing.T) {
	repo := &fakeRepo{timezone: "America/Sao_Paulo"}
	s := newTestService(repo)

	report, err := s.GenerateReport(context.Background(), uuid.New(), domain.ReportRequest{Type: domain.ReportSales})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Em UTC já é dia 28, mas em São Paulo ainda é 27: o período padrão vai até o dia 27
	if report.StartDate != "2026-10-01" || report.EndDate != "2026-10-27" {
		t.Errorf("período: esperava 2026-10-01 a 2026-10-27, veio %s a %s", report.StartDate, report.EndDate)
	}
	if repo.salesArg.Tz != "America/Sao_Paulo" {
		t.Errorf("tz: esperava America/Sao_Paulo, veio %s", repo.salesArg.Tz)
	}
	if got := repo.salesArg.EndDate.Time.Format(dateLayout); got != "2026-10-28" {
		t.Errorf("end_date exclusivo: esperava 2026-10-28, veio %s", got)
	}
	if report.Rows == nil || report.Summary == nil {
		t.Error("rows e summary devem ser listas vazias, não nulas")
	}
	if report.CompanyName != "Loja" {
		t.Errorf("company_name: esperava Loja, veio %s", report.CompanyName)
	}
}

func TestGenerateReportFusoInvalidoCaiNoPadrao(t *testing.T) {
	repo := &fakeRepo{timezone: "Fuso/Inexistente"}
	s := newTestService(repo)

	if _, err := s.GenerateReport(context.Background(), uuid.New(), domain.ReportRequest{Type: domain.ReportSales}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if repo.salesArg.Tz != defaultTimezone {
		t.Errorf("tz: esperava %s, veio %s", defaultTimezone, repo.salesArg.Tz)
	}
}

func TestSalesReportCanceladaForaDosTotais(t *testing.T) {
	repo := &fakeRepo{
		timezone: "America/Sao_Paulo",
		sales: []db.ReportSalesRow{
			{Status: "paid", PaymentMethod: "pix", Subtotal: 100, DiscountAmount: 10, TotalAmount: 90, TotalCost: 50},
			{Status: "canceled", Canceled: true, PaymentMethod: "cash", Subtotal: 40, TotalAmount: 40, TotalCost: 20},
		},
	}
	report, err := newTestService(repo).GenerateReport(context.Background(), uuid.New(), domain.ReportRequest{Type: domain.ReportSales})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(report.Rows) != 2 {
		t.Fatalf("a cancelada deve aparecer na lista: esperava 2 linhas, veio %d", len(report.Rows))
	}
	if report.Rows[0][3] != "Pix" || report.Rows[1][4] != "Cancelado" {
		t.Errorf("rótulos inesperados: %v / %v", report.Rows[0][3], report.Rows[1][4])
	}
	if total := report.Totals[8]; total != 90.0 {
		t.Errorf("total: esperava 90 (sem a cancelada), veio %v", total)
	}
	if profit := report.Totals[9]; profit != 40.0 {
		t.Errorf("lucro: esperava 40, veio %v", profit)
	}
}

func TestProductsABC(t *testing.T) {
	repo := &fakeRepo{
		timezone: "America/Sao_Paulo",
		products: []db.ReportSalesByProductRow{
			{ProductName: "P1", Net: 700, Cost: 400},
			{ProductName: "P2", Net: 200, Cost: 100},
			{ProductName: "P3", Net: 60, Cost: 30},
			{ProductName: "P4", Net: 40, Cost: 40},
		},
	}
	report, err := newTestService(repo).GenerateReport(context.Background(), uuid.New(), domain.ReportRequest{Type: domain.ReportProductsABC})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	want := []string{"A", "A", "B", "C"} // acumulado antes: 0, 70, 90, 96
	for i, row := range report.Rows {
		if row[0] != want[i] {
			t.Errorf("%s: esperava classe %s, veio %v", row[1], want[i], row[0])
		}
	}
	if margin := report.Rows[3][7]; margin != 0.0 {
		t.Errorf("P4 sem lucro: esperava margem 0, veio %v", margin)
	}
}

func TestInventoryNaoUsaPeriodoEIgnoraGranel(t *testing.T) {
	repo := &fakeRepo{
		timezone: "America/Sao_Paulo",
		inventory: []db.ReportInventoryRow{
			{ProductName: "Unitário", Quantity: 10, CostPrice: 2, SalePrice: 5},
			{ProductName: "Granel", SellInBulk: true, CostPrice: 30, SalePrice: 50},
		},
	}
	report, err := newTestService(repo).GenerateReport(context.Background(), uuid.New(), domain.ReportRequest{Type: domain.ReportInventory})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if report.UsesPeriod || report.StartDate != "" {
		t.Error("posição de estoque não deve informar período")
	}
	if report.Totals[8] != 20.0 || report.Totals[9] != 50.0 {
		t.Errorf("valorização: esperava 20/50, veio %v/%v", report.Totals[8], report.Totals[9])
	}
	if report.Rows[1][4] != "Granel" {
		t.Errorf("saldo do granel: esperava \"Granel\", veio %v", report.Rows[1][4])
	}
}
