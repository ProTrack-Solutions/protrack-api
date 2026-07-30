package domain

import "time"

type MPPreApprovalRequest struct {
	Reason            string             `json:"reason"`
	ExternalReference string             `json:"external_reference"`
	PayerEmail        string             `json:"payer_email"`
	CardTokenID       string             `json:"card_token_id"`
	Status            string             `json:"status"` // "authorized"
	BackURL           string             `json:"back_url"`
	AutoRecurring     MPAutoRecurringReq `json:"auto_recurring"`
}

type MPAutoRecurringReq struct {
	Frequency         int     `json:"frequency"`          // Ex: 1
	FrequencyType     string  `json:"frequency_type"`     // "months" ou "years"
	TransactionAmount float64 `json:"transaction_amount"` // Ex: 49.90
	CurrencyID        string  `json:"currency_id"`        // "BRL"
}

type MPPreApprovalResponse struct {
	ID              string    `json:"id"` // mp_subscription_id
	PayerEmail      string    `json:"payer_email"`
	Status          string    `json:"status"` // "authorized"
	Reason          string    `json:"reason"`
	DateCreated     time.Time `json:"date_created"`
	NextPaymentDate time.Time `json:"next_payment_date"`
}
