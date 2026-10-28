package service

import (
	"testing"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/receivable_reminders/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

func date(s string) pgtype.Date {
	t, _ := time.Parse("2006-01-02", s)
	return pgtype.Date{Time: t, Valid: true}
}

func TestReminderKind(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	// 27/10 às 22h em São Paulo (já é 28/10 em UTC)
	now := time.Date(2026, 10, 27, 22, 0, 0, 0, loc)

	tests := []struct {
		name   string
		status string
		due    string
		want   string
	}{
		{"status vencida", "overdue", "2026-11-10", domain.KindOverdue},
		{"vence hoje ainda é a vencer", "pending", "2026-10-27", domain.KindDue},
		{"venceu ontem antes do job rodar", "pending", "2026-10-26", domain.KindOverdue},
		{"parcial a vencer", "partial", "2026-11-05", domain.KindDue},
	}
	for _, tt := range tests {
		if got := reminderKind(tt.status, date(tt.due), now); got != tt.want {
			t.Errorf("%s: esperava %s, veio %s", tt.name, tt.want, got)
		}
	}
}

func TestMaskPhone(t *testing.T) {
	if got := maskPhone("5511987654321"); got != "*********4321" {
		t.Errorf("veio %q", got)
	}
	if got := maskPhone("123"); got != "123" {
		t.Errorf("veio %q", got)
	}
}
