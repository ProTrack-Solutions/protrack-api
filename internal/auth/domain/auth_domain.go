package domain

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
		Name         string `json:"name"`
		Email        string `json:"email"`
		Username     string `json:"username"`
		PasswordHash string `json:"password"`
		Document     string `json:"document"`
	} `json:"user"`
	Company struct {
		Name                string `json:"name"`
		TradeName           string `json:"trade_name"`
		Document            string `json:"document"`
		Email               string `json:"email"`
		Phone               string `json:"phone"`
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
	} `json:"company"`
}
