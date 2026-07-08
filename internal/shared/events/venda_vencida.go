package events

import (
	"time"

	"github.com/google/uuid"
)

type WhatsApp struct {
	IDSale       uuid.UUID `json:"id_sale"`
	CompanyID    uuid.UUID
	CustomerName string    `json:"customer_name"`
	PhoneNumber  string    `json:"phone_number"`
	Value        float64   `json:"value"`
	DueDate      time.Time `json:"due_date"`
	InstanceName string
	Message      string
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
