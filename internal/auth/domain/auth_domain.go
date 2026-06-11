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
