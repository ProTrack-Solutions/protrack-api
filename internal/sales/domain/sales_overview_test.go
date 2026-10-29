package domain

import (
	"errors"
	"testing"
	"time"
)

func date(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestBuildOverviewWindow(t *testing.T) {
	tests := []struct {
		name                         string
		start, end                   string
		unit, first, last, prevStart string
	}{
		// 30 dias: série diária e janela anterior com os 30 dias antes
		{"30 dias", "2026-09-30", "2026-10-29", "day", "2026-09-30", "2026-10-29", "2026-08-31"},
		{"um dia", "2026-10-29", "2026-10-29", "day", "2026-10-29", "2026-10-29", "2026-10-28"},
		// 90 dias: semanas começando na segunda
		{"90 dias", "2026-08-01", "2026-10-29", "week", "2026-07-27", "2026-10-26", "2026-05-03"},
		{"1 ano", "2025-10-30", "2026-10-29", "month", "2025-10-01", "2026-10-01", "2024-10-30"},
		// "Desde o início" pode passar de um ano: continua mensal
		{"desde 2023", "2023-03-15", "2026-10-29", "month", "2023-03-01", "2026-10-01", "2019-07-29"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := BuildOverviewWindow(date(tt.start), date(tt.end))
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			got := []string{w.Unit, w.FirstBucket.Format("2006-01-02"), w.LastBucket.Format("2006-01-02"), w.PrevStart.Format("2006-01-02")}
			want := []string{tt.unit, tt.first, tt.last, tt.prevStart}
			for i := range got {
				if got[i] != want[i] {
					t.Errorf("esperava %v, obteve %v", want, got)
					break
				}
			}
			if !w.End.Equal(date(tt.end).AddDate(0, 0, 1)) {
				t.Errorf("End deveria ser o dia seguinte ao fim, obteve %s", w.End)
			}
		})
	}
}

func TestBuildOverviewWindowInvalid(t *testing.T) {
	if _, err := BuildOverviewWindow(date("2026-10-29"), date("2026-10-01")); !errors.Is(err, ErrInvalidOverviewParams) {
		t.Errorf("fim antes do início deveria falhar, obteve %v", err)
	}
}

func TestAverageTicket(t *testing.T) {
	if got := AverageTicket(100, 3); got != 33.33 {
		t.Errorf("esperava 33.33, obteve %v", got)
	}
	if got := AverageTicket(100, 0); got != 0 {
		t.Errorf("sem vendas o ticket é 0, obteve %v", got)
	}
}
