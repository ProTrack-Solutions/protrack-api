package generator

import (
	"bytes"
	"fmt"

	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
	"github.com/go-pdf/fpdf"
)

const (
	reportPDFMargin    = 10.0
	reportPDFRowHeight = 6.0
	reportPDFFontSize  = 7.5
)

// GenerateReportPDF gera o relatório em A4 (paisagem quando tem muitas colunas),
// com indicadores no topo, cabeçalho da tabela repetido a cada página, linhas
// zebradas, totais em negrito e numeração de páginas no rodapé.
func GenerateReportPDF(r domain.Report) (*bytes.Buffer, error) {
	companyName := r.CompanyName
	orientation := "P"
	if len(r.Columns) > 5 {
		orientation = "L"
	}

	pdf := fpdf.New(orientation, "mm", "A4", "")
	pdf.SetMargins(reportPDFMargin, reportPDFMargin, reportPDFMargin)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AliasNbPages("{nb}")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pageWidth, _ := pdf.GetPageSize()
	contentWidth := pageWidth - 2*reportPDFMargin
	widths := reportPDFColumnWidths(pdf, tr, r, contentWidth)

	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Arial", "", 7)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(contentWidth/2, 4, tr(companyName+" · Gerado em "+r.GeneratedAt.Format("02/01/2006 15:04")), "", 0, "L", false, 0, "")
		pdf.CellFormat(contentWidth/2, 4, tr(fmt.Sprintf("Página %d de {nb}", pdf.PageNo())), "", 0, "R", false, 0, "")
	})

	drawHeader := func() {
		pdf.SetFont("Arial", "B", reportPDFFontSize)
		pdf.SetFillColor(67, 56, 202)
		pdf.SetTextColor(255, 255, 255)
		for i, c := range r.Columns {
			pdf.CellFormat(widths[i], reportPDFRowHeight+1, tr(fitText(pdf, tr, c.Label, widths[i])), "", 0, reportPDFAlign(c.Type), true, 0, "")
		}
		pdf.Ln(-1)
	}

	pdf.AddPage()

	// Título e período
	pdf.SetFont("Arial", "B", 15)
	pdf.SetTextColor(17, 24, 39)
	pdf.CellFormat(contentWidth, 8, tr(r.Title), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(107, 114, 128)
	pdf.CellFormat(contentWidth, 5, tr(reportPDFSubtitle(r, companyName)), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// Indicadores
	if len(r.Summary) > 0 {
		gap := 3.0
		boxWidth := (contentWidth - gap*float64(len(r.Summary)-1)) / float64(len(r.Summary))
		y := pdf.GetY()
		for i, s := range r.Summary {
			x := reportPDFMargin + float64(i)*(boxWidth+gap)
			pdf.SetFillColor(238, 242, 255)
			pdf.RoundedRect(x, y, boxWidth, 14, 2, "1234", "F")
			pdf.SetXY(x+3, y+2)
			pdf.SetFont("Arial", "", 7)
			pdf.SetTextColor(107, 114, 128)
			pdf.CellFormat(boxWidth-6, 4, tr(fitText(pdf, tr, s.Label, boxWidth-6)), "", 2, "L", false, 0, "")
			pdf.SetFont("Arial", "B", 11)
			pdf.SetTextColor(17, 24, 39)
			pdf.CellFormat(boxWidth-6, 6, tr(formatReportValue(s.Value, s.Type)), "", 0, "L", false, 0, "")
		}
		pdf.SetXY(reportPDFMargin, y+18)
	}

	drawHeader()
	pdf.SetHeaderFunc(func() {
		if pdf.PageNo() > 1 {
			drawHeader()
		}
	})

	pdf.SetFont("Arial", "", reportPDFFontSize)
	if len(r.Rows) == 0 {
		pdf.SetTextColor(107, 114, 128)
		pdf.CellFormat(contentWidth, 12, tr("Nenhum registro encontrado no período."), "", 1, "C", false, 0, "")
	}

	for i, row := range r.Rows {
		pdf.SetTextColor(31, 41, 55)
		if i%2 == 1 {
			pdf.SetFillColor(249, 250, 251)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		reportPDFRow(pdf, tr, r.Columns, widths, row)
	}

	if len(r.Totals) > 0 && len(r.Rows) > 0 {
		pdf.SetFont("Arial", "B", reportPDFFontSize)
		pdf.SetFillColor(224, 231, 255)
		pdf.SetTextColor(17, 24, 39)
		reportPDFRow(pdf, tr, r.Columns, widths, r.Totals)
	}

	if len(r.Notes) > 0 {
		pdf.Ln(4)
		pdf.SetFont("Arial", "I", 7)
		pdf.SetTextColor(107, 114, 128)
		for _, note := range r.Notes {
			pdf.MultiCell(contentWidth, 4, tr("* "+note), "", "L", false)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return &buf, nil
}

func reportPDFRow(pdf *fpdf.Fpdf, tr func(string) string, columns []domain.Column, widths []float64, values []any) {
	for i, c := range columns {
		var value any
		if i < len(values) {
			value = values[i]
		}
		text := fitText(pdf, tr, formatReportValue(value, c.Type), widths[i])
		pdf.CellFormat(widths[i], reportPDFRowHeight, tr(text), "", 0, reportPDFAlign(c.Type), true, 0, "")
	}
	pdf.Ln(-1)
}

func reportPDFSubtitle(r domain.Report, companyName string) string {
	if companyName == "" {
		return reportPeriodLabel(r)
	}
	return companyName + "  ·  " + reportPeriodLabel(r)
}

func reportPDFAlign(t domain.ColumnType) string {
	switch t {
	case domain.ColumnMoney, domain.ColumnNumber, domain.ColumnPercent:
		return "R"
	case domain.ColumnDate, domain.ColumnDateTime:
		return "C"
	}
	return "L"
}

// reportPDFColumnWidths mede o maior conteúdo de cada coluna (cabeçalho em
// negrito e uma amostra das linhas) e ajusta as larguras à área útil da página.
func reportPDFColumnWidths(pdf *fpdf.Fpdf, tr func(string) string, r domain.Report, total float64) []float64 {
	const padding, maxWidth = 3.0, 70.0
	sample := r.Rows[:min(len(r.Rows), 300)]

	widths := make([]float64, len(r.Columns))
	var sum float64
	for i, c := range r.Columns {
		pdf.SetFont("Arial", "B", reportPDFFontSize)
		needed := pdf.GetStringWidth(tr(c.Label))
		if i < len(r.Totals) {
			needed = max(needed, pdf.GetStringWidth(tr(formatReportValue(r.Totals[i], c.Type))))
		}
		pdf.SetFont("Arial", "", reportPDFFontSize)
		for _, row := range sample {
			if i < len(row) {
				needed = max(needed, pdf.GetStringWidth(tr(formatReportValue(row[i], c.Type))))
			}
		}
		widths[i] = min(needed+padding, maxWidth)
		sum += widths[i]
	}

	// Sobra ou falta espaço: distribui proporcionalmente entre as colunas
	for i := range widths {
		widths[i] = widths[i] / sum * total
	}
	return widths
}

// fitText corta o texto com "..." quando não cabe na largura da célula
func fitText(pdf *fpdf.Fpdf, tr func(string) string, text string, width float64) string {
	limit := width - 2
	if pdf.GetStringWidth(tr(text)) <= limit {
		return text
	}
	runes := []rune(text)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + "..."
		if pdf.GetStringWidth(tr(candidate)) <= limit {
			return candidate
		}
	}
	return string(runes)
}
