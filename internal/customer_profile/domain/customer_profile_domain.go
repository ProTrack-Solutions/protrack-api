package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrCustomerNotFound = errors.New("cliente não encontrado")

type BrowseParams struct {
	Page    int32  `form:"page" binding:"omitempty,gte=1"`
	PerPage int32  `form:"per_page" binding:"omitempty,gte=1,lte=100"`
	Search  string `form:"search" binding:"max=100"`
	Status  string `form:"status" binding:"omitempty,oneof=active inactive all"`
	// debtors: com saldo devedor; overdue: com parcela vencida; credit: com
	// vale-troca; no_purchases: nunca comprou
	Segment string `form:"segment" binding:"omitempty,oneof=all debtors overdue credit no_purchases"`
	Sort    string `form:"sort" binding:"omitempty,oneof=recent name last_purchase total_spent balance"`
}

type Overview struct {
	ActiveCount   int32   `json:"active_count"`
	InactiveCount int32   `json:"inactive_count"`
	NewThisMonth  int32   `json:"new_this_month"`
	NewLastMonth  int32   `json:"new_last_month"`
	DebtorsCount  int32   `json:"debtors_count"`
	DebtTotal     float64 `json:"debt_total"`
	OverdueCount  int32   `json:"overdue_count"`
	OverdueTotal  float64 `json:"overdue_total"`
	CreditCount   int32   `json:"credit_count"`
	CreditTotal   float64 `json:"credit_total"`
	// Clientes que compraram nos últimos 30 dias
	RecentBuyers int32 `json:"recent_buyers"`
}

type CustomerRow struct {
	ID             uuid.UUID  `json:"id"`
	FullName       string     `json:"full_name"`
	Cpf            string     `json:"cpf"`
	Email          string     `json:"email"`
	Whatsapp       string     `json:"whatsapp"`
	MobilePhone    string     `json:"mobile_phone"`
	City           string     `json:"city"`
	State          string     `json:"state"`
	BalanceDue     float64    `json:"balance_due"`
	OverdueTotal   float64    `json:"overdue_total"`
	StoreCredit    float64    `json:"store_credit"`
	PurchasesCount int32      `json:"purchases_count"`
	PurchasesTotal float64    `json:"purchases_total"`
	LastPurchaseAt *time.Time `json:"last_purchase_at"`
	CreatedAt      time.Time  `json:"created_at"`
	Active         bool       `json:"active"`
}

type BrowseResponse struct {
	Data    []CustomerRow `json:"data"`
	Page    int32         `json:"page"`
	PerPage int32         `json:"per_page"`
	Total   int64         `json:"total"`
}

type Installment struct {
	ID                uuid.UUID `json:"id"`
	SaleID            uuid.UUID `json:"sale_id"`
	InstallmentNumber int32     `json:"installment_number"`
	TotalInstallments int32     `json:"total_installments"`
	DueDate           string    `json:"due_date"`
	TotalAmount       float64   `json:"total_amount"`
	Balance           float64   `json:"balance"`
	Status            string    `json:"status"`
	Overdue           bool      `json:"overdue"`
}

type RecentSale struct {
	ID            uuid.UUID `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	TotalAmount   float64   `json:"total_amount"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	ItemsCount    int32     `json:"items_count"`
	Products      string    `json:"products"`
	ReturnedTotal float64   `json:"returned_total"`
}

type CreditMovement struct {
	ID           uuid.UUID  `json:"id"`
	Type         string     `json:"type"`
	Amount       float64    `json:"amount"`
	BalanceAfter float64    `json:"balance_after"`
	SaleReturnID *uuid.UUID `json:"sale_return_id"`
	ReturnNumber *int32     `json:"return_number"`
	SaleID       *uuid.UUID `json:"sale_id"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Profile struct {
	ID                  uuid.UUID `json:"id"`
	FullName            string    `json:"full_name"`
	Cpf                 string    `json:"cpf"`
	Rg                  string    `json:"rg"`
	BirthDate           *string   `json:"birth_date"`
	Gender              string    `json:"gender"`
	MaritalStatus       string    `json:"marital_status"`
	Email               string    `json:"email"`
	Whatsapp            string    `json:"whatsapp"`
	MobilePhone         string    `json:"mobile_phone"`
	HomePhone           string    `json:"home_phone"`
	AddressStreet       string    `json:"address_street"`
	AddressNumber       string    `json:"address_number"`
	AddressComplement   string    `json:"address_complement"`
	AddressNeighborhood string    `json:"address_neighborhood"`
	AddressCity         string    `json:"address_city"`
	AddressState        string    `json:"address_state"`
	AddressZipcode      string    `json:"address_zipcode"`
	CreatedAt           time.Time `json:"created_at"`
	Active              bool      `json:"active"`

	BalanceDue   float64 `json:"balance_due"`
	OverdueTotal float64 `json:"overdue_total"`
	StoreCredit  float64 `json:"store_credit"`

	PurchasesCount  int32      `json:"purchases_count"`
	PurchasesTotal  float64    `json:"purchases_total"`
	ReturnedTotal   float64    `json:"returned_total"`
	AverageTicket   float64    `json:"average_ticket"`
	FirstPurchaseAt *time.Time `json:"first_purchase_at"`
	LastPurchaseAt  *time.Time `json:"last_purchase_at"`

	// Perfil de compra: o que, quando e como o cliente compra
	Insights Insights `json:"insights"`

	OpenInstallments []Installment    `json:"open_installments"`
	RecentSales      []RecentSale     `json:"recent_sales"`
	CreditMovements  []CreditMovement `json:"credit_movements"`
}
