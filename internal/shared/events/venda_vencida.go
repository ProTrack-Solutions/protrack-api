package events

import (
	"time"

	"github.com/google/uuid"
)

type WhatsApp struct {
	IDSale       uuid.UUID `json:"id_sale"`
	CompanyID    uuid.UUID
	CompanyName  string    `json:"company_name"` // novo — pro cabeçalho
	CustomerName string    `json:"customer_name"`
	PhoneNumber  string    `json:"phone_number"`
	Value        float64   `json:"value"`
	DueDate      time.Time `json:"due_date"`
	PurchaseDate time.Time `json:"purchase_date"`
	ContactInfo  string    `json:"contact_info"` // novo — telefone/e-mail da empresa
}

type Announcement struct {
	CompanyID     uuid.UUID `json:"company_id"`
	Message       string    `json:"message"`
	Title         string    `json:"title"`
	Type          string    `json:"type"`
	TotalVencidas int       `json:"total_vencidas"`
	StartsAt      time.Time `json:"starts_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}
