package generator

import (
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
	"github.com/xuri/excelize/v2"
)

const (
	excelMoneyFormat    = `"R$" #,##0.00;[Red]-"R$" #,##0.00`
	excelNumberFormat   = `#,##0.##`
	excelPercentFormat  = `0.0%`
	excelDateFormat     = `dd/mm/yyyy`
	excelDateTimeFormat = `dd/mm/yyyy hh:mm`
	excelHeaderRow      = 4
)

// GenerateReportExcel escreve o relatório em xlsx: título e período no topo,
// cabeçalho com filtro e painel congelado, valores com formato numérico real
// (o usuário consegue somar/filtrar no Excel) e linha de totais em negrito.
func GenerateReportExcel(w io.Writer, r domain.Report) (err error) {
	f := excelize.NewFile()
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()

	sheet := excelSheetName(r.Title)
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return err
	}

	titleStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	if err != nil {
		return err
	}
	subtitleStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "6B7280"}})
	if err != nil {
		return err
	}
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4338CA"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	if err != nil {
		return err
	}

	columnStyles := make([]int, len(r.Columns))
	totalStyles := make([]int, len(r.Columns))
	for i, c := range r.Columns {
		format := excelColumnFormat(c.Type)
		columnStyles[i], err = f.NewStyle(&excelize.Style{CustomNumFmt: format})
		if err != nil {
			return err
		}
		totalStyles[i], err = f.NewStyle(&excelize.Style{
			CustomNumFmt: format,
			Font:         &excelize.Font{Bold: true},
			Fill:         excelize.Fill{Type: "pattern", Color: []string{"EEF2FF"}, Pattern: 1},
			Border:       []excelize.Border{{Type: "top", Color: "4338CA", Style: 1}},
		})
		if err != nil {
			return err
		}
	}

	// setCell grava valor (nil deixa a célula vazia) e estilo numa coordenada
	setCell := func(col, row int, value any, style int) error {
		cell, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			return err
		}
		if value != nil {
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return err
			}
		}
		return f.SetCellStyle(sheet, cell, cell, style)
	}

	if err := setCell(1, 1, r.Title, titleStyle); err != nil {
		return err
	}
	if err := setCell(1, 2, reportPeriodLabel(r)+"  ·  Gerado em "+r.GeneratedAt.Format("02/01/2006 15:04"), subtitleStyle); err != nil {
		return err
	}

	widths := make([]int, len(r.Columns))
	for i, c := range r.Columns {
		if err := setCell(i+1, excelHeaderRow, c.Label, headerStyle); err != nil {
			return err
		}
		widths[i] = utf8.RuneCountInString(c.Label)
	}

	writeRow := func(rowIndex int, values []any, styles []int) error {
		for i, c := range r.Columns {
			var value any
			if i < len(values) {
				value = values[i]
				widths[i] = max(widths[i], utf8.RuneCountInString(formatReportValue(value, c.Type)))
			}
			if err := setCell(i+1, rowIndex, excelValue(value, c.Type), styles[i]); err != nil {
				return err
			}
		}
		return nil
	}

	rowIndex := excelHeaderRow + 1
	for _, row := range r.Rows {
		if err := writeRow(rowIndex, row, columnStyles); err != nil {
			return err
		}
		rowIndex++
	}
	lastDataRow := rowIndex - 1

	if len(r.Totals) > 0 {
		if err := writeRow(rowIndex, r.Totals, totalStyles); err != nil {
			return err
		}
		rowIndex++
	}

	if len(r.Notes) > 0 {
		rowIndex++
		for _, note := range r.Notes {
			if err := setCell(1, rowIndex, note, subtitleStyle); err != nil {
				return err
			}
			rowIndex++
		}
	}

	for i, width := range widths {
		colName, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, colName, colName, float64(min(max(width+2, 10), 60))); err != nil {
			return err
		}
	}

	if len(r.Columns) > 0 {
		lastCol, err := excelize.ColumnNumberToName(len(r.Columns))
		if err != nil {
			return err
		}
		if len(r.Rows) > 0 {
			filterRange := "A" + strconv.Itoa(excelHeaderRow) + ":" + lastCol + strconv.Itoa(lastDataRow)
			if err := f.AutoFilter(sheet, filterRange, nil); err != nil {
				return err
			}
		}
		if err := f.SetPanes(sheet, &excelize.Panes{
			Freeze:      true,
			YSplit:      excelHeaderRow,
			TopLeftCell: "A" + strconv.Itoa(excelHeaderRow+1),
			ActivePane:  "bottomLeft",
		}); err != nil {
			return err
		}
	}

	return f.Write(w)
}

func excelColumnFormat(t domain.ColumnType) *string {
	var format string
	switch t {
	case domain.ColumnMoney:
		format = excelMoneyFormat
	case domain.ColumnPercent:
		format = excelPercentFormat
	case domain.ColumnDate:
		format = excelDateFormat
	case domain.ColumnDateTime:
		format = excelDateTimeFormat
	case domain.ColumnNumber:
		format = excelNumberFormat
	default:
		return nil
	}
	return &format
}

// excelValue converte o valor para o que o Excel entende: percentual vira
// fração (35.2 → 0.352) e datas vão com o horário de parede, sem fuso.
func excelValue(value any, t domain.ColumnType) any {
	switch v := value.(type) {
	case nil:
		return nil
	case time.Time:
		return time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), 0, time.UTC)
	}
	if t == domain.ColumnPercent {
		if n, ok := toFloat(value); ok {
			return n / 100
		}
	}
	return value
}

// excelSheetName nome de aba válido: até 31 caracteres e sem : \ / ? * [ ]
func excelSheetName(title string) string {
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`:\/?*[]`, r) {
			return ' '
		}
		return r
	}, title)
	if runes := []rune(name); len(runes) > 31 {
		name = string(runes[:31])
	}
	if strings.TrimSpace(name) == "" {
		return "Relatório"
	}
	return name
}
