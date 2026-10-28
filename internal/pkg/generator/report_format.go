package generator

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
)

// formatReportValue formata um valor do relatório no padrão brasileiro, para o PDF
func formatReportValue(value any, t domain.ColumnType) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case time.Time:
		if t == domain.ColumnDateTime {
			return v.Format("02/01/2006 15:04")
		}
		return v.Format("02/01/2006")
	case string:
		return v
	}

	n, ok := toFloat(value)
	if !ok {
		return fmt.Sprint(value)
	}

	switch t {
	case domain.ColumnMoney:
		formatted := formatNumberBR(n, 2)
		if after, negative := strings.CutPrefix(formatted, "-"); negative {
			return "-R$ " + after
		}
		return "R$ " + formatted
	case domain.ColumnPercent:
		return formatNumberBR(n, 1) + "%"
	default:
		if n == math.Trunc(n) {
			return formatNumberBR(n, 0)
		}
		return formatNumberBR(n, 2)
	}
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

// formatNumberBR 1234567.891 → "1.234.567,89"
func formatNumberBR(v float64, decimals int) string {
	s := fmt.Sprintf("%.*f", decimals, math.Abs(v))
	intPart, decPart, _ := strings.Cut(s, ".")

	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}

	out := b.String()
	if decPart != "" {
		out += "," + decPart
	}
	if v < 0 && strings.Trim(out, "0.,") != "" {
		out = "-" + out
	}
	return out
}

// reportPeriodLabel "Período: 01/10/2026 a 31/10/2026" (vazio quando o relatório não usa período)
func reportPeriodLabel(r domain.Report) string {
	if !r.UsesPeriod {
		return "Posição em " + r.GeneratedAt.Format("02/01/2006 15:04")
	}
	start, _ := time.Parse("2006-01-02", r.StartDate)
	end, _ := time.Parse("2006-01-02", r.EndDate)
	return fmt.Sprintf("Período: %s a %s", start.Format("02/01/2006"), end.Format("02/01/2006"))
}

// ReportFileName nome do arquivo para download, sem acentos nem espaços
func ReportFileName(r domain.Report, ext string) string {
	name := string(r.Type)
	if r.UsesPeriod {
		name += "_" + r.StartDate + "_" + r.EndDate
	} else {
		name += "_" + r.GeneratedAt.Format("2006-01-02")
	}
	return name + "." + ext
}
