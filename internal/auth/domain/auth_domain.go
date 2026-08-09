package domain

import (
	"github.com/google/uuid"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Aud      string `json:"aud" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`  // Token de acesso
	RefreshToken string `json:"refresh_token"` // Token de renovação
	HasCompany   bool   `json:"has_company"`
	ExpiresIn    int64  `json:"expires_in"` // Tempo de expiração em segundos
	TokenType    string `json:"token_type"` // Tipo do token (Bearer)
}

type RegisterRequest struct {
	User struct {
		Name     string `json:"name" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Username string `json:"username" validate:"required"`
		Password string `json:"password" validate:"required,min=8"`
		Document string `json:"document" validate:"required"`
	} `json:"user" validate:"required"`

	Company struct {
		Name                string `json:"name" validate:"required"`
		TradeName           string `json:"trade_name"`
		Document            string `json:"document" validate:"required"`
		Email               string `json:"email" validate:"required,email"`
		Phone               string `json:"phone" validate:"required"`
		Website             string `json:"website"`
		AddressStreet       string `json:"address_street"`
		AddressNumber       string `json:"address_number"`
		AddressComplement   string `json:"address_complement"`
		AddressNeighborhood string `json:"address_neighborhood"`
		AddressCity         string `json:"address_city"`
		AddressState        string `json:"address_state"`
		AddressZipcode      string `json:"address_zipcode"`
		AddressCountry      string `json:"address_country"`
		Timezone            string `json:"timezone"`
	} `json:"company" validate:"required"`

	Payment struct {
		PlanID       uuid.UUID `json:"plan_id" validate:"required"`
		CardToken    string    `json:"card_token" validate:"required"`
		Type         string    `json:"type" validate:"required"`
		CardBrand    string    `json:"card_brand" validate:"required"`
		CardLastFour string    `json:"card_last_four" validate:"required"`
		CardExpMonth int32     `json:"card_exp_month" validate:"required"`
		CardExpYear  int32     `json:"card_exp_year" validate:"required"`
	} `json:"payment" validate:"required"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

type RegisterResponse struct {
	CompanyID uuid.UUID `json:"company_id"`
	// Status da assinatura no Stripe logo após a criação (ex: "incomplete").
	SubscriptionStatus string `json:"subscription_status"`
	// ClientSecret do PaymentIntent da primeira invoice. O frontend PRECISA
	// chamar stripe.confirmCardPayment(client_secret) com esse valor para
	// concluir a autenticação do cartão (incluindo 3D Secure); sem essa
	// confirmação a assinatura fica "incomplete" e expira em ~23h no Stripe.
	ClientSecret string `json:"client_secret,omitempty"`
	// RequiresAction indica que o frontend deve chamar confirmCardPayment
	// antes de considerar o cadastro concluído.
	RequiresAction bool `json:"requires_action"`
}
