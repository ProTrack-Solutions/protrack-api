package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/handler"
	"github.com/ProTrack-Solutions/protrack-api/internal/reports/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeRepo struct {
	service.RepositoryInterface
}

func (fakeRepo) ReportCompanyHeader(ctx context.Context, companyId pgtype.UUID) (db.ReportCompanyHeaderRow, error) {
	return db.ReportCompanyHeaderRow{CompanyName: "Loja", Timezone: "America/Sao_Paulo"}, nil
}

func (fakeRepo) ReportSalesByPaymentMethod(ctx context.Context, arg db.ReportSalesByPaymentMethodParams) ([]db.ReportSalesByPaymentMethodRow, error) {
	return []db.ReportSalesByPaymentMethodRow{
		{PaymentMethod: "pix", SalesCount: 2, Total: 150},
		{PaymentMethod: "cash", SalesCount: 1, Total: 50},
	}, nil
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	now := func() time.Time { return time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC) }
	h := handler.NewHandler(service.NewServiceWithRepo(fakeRepo{}, now), nil, nil, nil)

	r := gin.New()
	r.GET("/reports", func(c *gin.Context) {
		c.Set("company_id", uuid.New())
		h.GenerateReport(c)
	})
	return r
}

func get(t *testing.T, query string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reports?"+query, nil)
	newRouter().ServeHTTP(w, req)
	return w
}

func TestGenerateReport_Erros(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		status int
	}{
		{"sem tipo", "", http.StatusBadRequest},
		{"tipo inexistente", "type=nao_existe", http.StatusNotFound},
		{"data inválida", "type=sales_by_payment_method&start_date=01-10-2026", http.StatusBadRequest},
		{"período invertido", "type=sales_by_payment_method&start_date=2026-10-10&end_date=2026-10-01", http.StatusBadRequest},
		{"formato inválido", "type=sales_by_payment_method&format=csv", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := get(t, tt.query); w.Code != tt.status {
				t.Errorf("esperava %d, veio %d: %s", tt.status, w.Code, w.Body.String())
			}
		})
	}
}

func TestGenerateReport_JSON(t *testing.T) {
	w := get(t, "type=sales_by_payment_method&start_date=2026-10-01&end_date=2026-10-31")
	if w.Code != http.StatusOK {
		t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
	}

	var report domain.Report
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("json inválido: %v", err)
	}
	if report.StartDate != "2026-10-01" || report.EndDate != "2026-10-31" {
		t.Errorf("período inesperado: %s a %s", report.StartDate, report.EndDate)
	}
	if len(report.Rows) != 2 || report.Rows[0][0] != "Pix" {
		t.Errorf("linhas inesperadas: %v", report.Rows)
	}
	if share := report.Rows[0][5]; share != 75.0 {
		t.Errorf("participação do Pix: esperava 75, veio %v", share)
	}
}

func TestGenerateReport_Arquivos(t *testing.T) {
	tests := []struct {
		format      string
		contentType string
		magic       []byte
	}{
		{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", []byte("PK")},
		{"pdf", "application/pdf", []byte("%PDF")},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			w := get(t, "type=sales_by_payment_method&start_date=2026-10-01&end_date=2026-10-31&format="+tt.format)
			if w.Code != http.StatusOK {
				t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body.String())
			}
			if got := w.Header().Get("Content-Type"); got != tt.contentType {
				t.Errorf("content-type: esperava %s, veio %s", tt.contentType, got)
			}
			want := `filename="sales_by_payment_method_2026-10-01_2026-10-31.` + tt.format + `"`
			if got := w.Header().Get("Content-Disposition"); !strings.Contains(got, want) {
				t.Errorf("content-disposition: esperava %s, veio %s", want, got)
			}
			if !bytes.HasPrefix(w.Body.Bytes(), tt.magic) {
				t.Errorf("arquivo não começa com %q", tt.magic)
			}
		})
	}
}
