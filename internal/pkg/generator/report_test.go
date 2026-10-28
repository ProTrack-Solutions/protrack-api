package generator

import (
	"bytes"
	"testing"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
	"github.com/xuri/excelize/v2"
)

func TestFormatReportValue(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	moment := time.Date(2026, 10, 5, 14, 30, 0, 0, loc)

	tests := []struct {
		value any
		t     domain.ColumnType
		want  string
	}{
		{1234567.891, domain.ColumnMoney, "R$ 1.234.567,89"},
		{-10.0, domain.ColumnMoney, "-R$ 10,00"},
		{-0.001, domain.ColumnMoney, "R$ 0,00"},
		{34.23, domain.ColumnPercent, "34,2%"},
		{int32(1500), domain.ColumnNumber, "1.500"},
		{2.5, domain.ColumnNumber, "2,50"},
		{moment, domain.ColumnDate, "05/10/2026"},
		{moment, domain.ColumnDateTime, "05/10/2026 14:30"},
		{"Granel", domain.ColumnNumber, "Granel"},
		{nil, domain.ColumnMoney, ""},
	}
	for _, tt := range tests {
		if got := formatReportValue(tt.value, tt.t); got != tt.want {
			t.Errorf("formatReportValue(%v, %s): esperava %q, veio %q", tt.value, tt.t, tt.want, got)
		}
	}
}

func TestExcelSheetName(t *testing.T) {
	if got := excelSheetName("Vendas: 01/10 [teste]"); got != "Vendas  01 10  teste " {
		t.Errorf("caracteres inválidos não removidos: %q", got)
	}
	if got := excelSheetName("Parcelas em aberto e inadimplência total"); len([]rune(got)) != 31 {
		t.Errorf("esperava 31 caracteres, veio %d", len([]rune(got)))
	}
}

func sampleReport() domain.Report {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	return domain.Report{
		Type:        domain.ReportSales,
		Title:       "Vendas no período",
		CompanyName: "Loja",
		UsesPeriod:  true,
		StartDate:   "2026-10-01",
		EndDate:     "2026-10-31",
		GeneratedAt: time.Date(2026, 10, 28, 12, 0, 0, 0, loc),
		Columns: []domain.Column{
			{Key: "sale_at", Label: "Data", Type: domain.ColumnDateTime},
			{Key: "customer", Label: "Cliente", Type: domain.ColumnText},
			{Key: "total", Label: "Total", Type: domain.ColumnMoney},
			{Key: "margin", Label: "Margem", Type: domain.ColumnPercent},
		},
		Rows: [][]any{
			{time.Date(2026, 10, 31, 23, 30, 0, 0, loc), "Cliente com acentuação", 90.0, 35.5},
		},
		Totals:  []any{"Total", nil, 90.0, nil},
		Summary: []domain.SummaryItem{{Label: "Faturamento", Value: 90, Type: domain.ColumnMoney}},
		Notes:   []string{"Observação"},
	}
}

func TestGenerateReportExcel(t *testing.T) {
	var buf bytes.Buffer
	if err := GenerateReportExcel(&buf, sampleReport()); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatalf("xlsx inválido: %v", err)
	}
	sheet := f.GetSheetList()[0]

	raw := func(cell string) string {
		v, _ := f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
		return v
	}
	if got := raw("C5"); got != "90" {
		t.Errorf("valor monetário deve ser número: esperava 90, veio %q", got)
	}
	if got := raw("D5"); got != "0.355" {
		t.Errorf("percentual deve virar fração: esperava 0.355, veio %q", got)
	}
	// A data mantém o horário de parede local (23:30 do dia 31), sem virar o dia em UTC
	if got, _ := f.GetCellValue(sheet, "A5"); got != "31/10/2026 23:30" {
		t.Errorf("data: esperava 31/10/2026 23:30, veio %q", got)
	}
	if got := raw("A6"); got != "Total" {
		t.Errorf("linha de totais: esperava Total, veio %q", got)
	}
}

func TestGenerateReportPDF(t *testing.T) {
	buf, err := GenerateReportPDF(sampleReport())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Error("saída não é um PDF")
	}

	empty := sampleReport()
	empty.Rows = [][]any{}
	if _, err := GenerateReportPDF(empty); err != nil {
		t.Fatalf("relatório vazio: %v", err)
	}
}

func TestReportFileName(t *testing.T) {
	r := sampleReport()
	if got := ReportFileName(r, "xlsx"); got != "sales_2026-10-01_2026-10-31.xlsx" {
		t.Errorf("veio %q", got)
	}
	r.UsesPeriod = false
	r.Type = domain.ReportInventory
	if got := ReportFileName(r, "pdf"); got != "inventory_2026-10-28.pdf" {
		t.Errorf("veio %q", got)
	}
}
