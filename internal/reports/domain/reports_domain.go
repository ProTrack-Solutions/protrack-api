package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidReportParams = errors.New("parâmetros do relatório inválidos")
	ErrReportNotFound      = errors.New("relatório não encontrado")
)

type ReportType string

const (
	ReportSales                ReportType = "sales"
	ReportSalesByDay           ReportType = "sales_by_day"
	ReportSalesByPaymentMethod ReportType = "sales_by_payment_method"
	ReportSalesBySeller        ReportType = "sales_by_seller"
	ReportSalesByCustomer      ReportType = "sales_by_customer"
	ReportProductsABC          ReportType = "products_abc"
	ReportProfitCategory       ReportType = "profit_category"
	ReportInventory            ReportType = "inventory"
	ReportStockMovements       ReportType = "stock_movements"
	ReportDRE                  ReportType = "dre"
	ReportPaymentsReceived     ReportType = "payments_received"
	ReportOpenInstallments     ReportType = "open_installments"
	ReportBillsPayable         ReportType = "bills_payable"
	ReportCashSessions         ReportType = "cash_sessions"
)

type ReportFormat string

const (
	FormatJSON ReportFormat = "json"
	FormatXLSX ReportFormat = "xlsx"
	FormatPDF  ReportFormat = "pdf"
)

// ColumnType diz como o valor da coluna deve ser formatado (tela, Excel e PDF)
type ColumnType string

const (
	ColumnText     ColumnType = "text"
	ColumnMoney    ColumnType = "money"
	ColumnNumber   ColumnType = "number"
	ColumnPercent  ColumnType = "percent"
	ColumnDate     ColumnType = "date"
	ColumnDateTime ColumnType = "datetime"
)

type ReportRequest struct {
	Type      ReportType   `form:"type" binding:"required" example:"sales"`
	StartDate string       `form:"start_date" example:"2026-10-01"` // YYYY-MM-DD, inclusivo
	EndDate   string       `form:"end_date" example:"2026-10-31"`   // YYYY-MM-DD, inclusivo
	Format    ReportFormat `form:"format" example:"json"`           // json (padrão), xlsx ou pdf
}

type Column struct {
	Key   string     `json:"key"`
	Label string     `json:"label"`
	Type  ColumnType `json:"type"`
}

// SummaryItem indicador de destaque mostrado acima da tabela
type SummaryItem struct {
	Label string     `json:"label"`
	Value float64    `json:"value"`
	Type  ColumnType `json:"type"`
}

// Report relatório pronto para exibir ou exportar. Cada linha de Rows segue a
// ordem de Columns; datas vão como time.Time no fuso da empresa. Totals tem o
// mesmo tamanho de Columns, com nil nas colunas que não somam.
type Report struct {
	Type        ReportType    `json:"type"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	CompanyName string        `json:"company_name"`
	UsesPeriod  bool          `json:"uses_period"`
	StartDate   string        `json:"start_date"`
	EndDate     string        `json:"end_date"`
	GeneratedAt time.Time     `json:"generated_at"`
	Columns     []Column      `json:"columns"`
	Rows        [][]any       `json:"rows"`
	Totals      []any         `json:"totals,omitempty"`
	Summary     []SummaryItem `json:"summary"`
	Notes       []string      `json:"notes,omitempty"`
}
