package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Aplicações que podem logar. Cada usuário tem no máximo uma sessão ativa em
// cada uma: um login novo derruba o anterior da mesma aplicação.
const (
	AudienceWeb = "protrack-gerencial-web"
	AudienceApp = "protrack-gerencial-app"
	// Painel interno da plataforma (super admin), fora da regra de sessão única
	AudiencePlatform = "platform"
)

// IsUserAudience diz se o aud é de uma aplicação de usuário (web ou app)
func IsUserAudience(aud string) bool {
	return aud == AudienceWeb || aud == AudienceApp
}

type Claims struct {
	CompanyId    uuid.UUID `json:"company_id"`
	Role         string    `json:"role"`
	DepartmentId uuid.UUID `json:"department_id"`
	// Sessão do login que gerou o token; mantida nas renovações
	SessionId string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

// Audience é o aud do token (vazio se não houver)
func (c *Claims) AudienceValue() string {
	if len(c.Audience) == 0 {
		return ""
	}
	return c.Audience[0]
}

// TokenKey identifica o token na lista de revogados: o jti quando existe;
// tokens antigos, sem jti, seguem pelo par usuário:expiração.
func (c *Claims) TokenKey() string {
	if c.ID != "" {
		return c.ID
	}
	return fmt.Sprintf("%s:%d", c.Subject, c.ExpiresAt.Unix())
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpireIn     int64  `json:"expires_in"`
}

type JWTManager struct {
	secretKey       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey:       secretKey,
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
	}
}

// RefreshTTL é a validade do refresh token (e da sessão no Redis)
func (j *JWTManager) RefreshTTL() time.Duration {
	return j.refreshTokenTTL
}

func (j *JWTManager) GenerateTokenPair(departmentId, sub, companyId uuid.UUID, role string, aud string, sessionId string) (*TokenPair, error) {
	accessClaims := &Claims{
		CompanyId:    companyId,
		Role:         role,
		DepartmentId: departmentId,
		SessionId:    sessionId,

		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "protrack",
			Subject:   sub.String(),
			Audience:  []string{aud},
			ID:        uuid.NewString(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(j.secretKey))
	if err != nil {
		return nil, err
	}

	refreshClaims := &Claims{
		CompanyId:    companyId,
		Role:         role,
		DepartmentId: departmentId,
		SessionId:    sessionId,

		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "protrack",
			Subject:   sub.String(),
			Audience:  []string{aud},
			ID:        uuid.NewString(),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(j.secretKey))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpireIn:     int64(j.accessTokenTTL.Seconds()),
	}, nil
}

func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(j.secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (j *JWTManager) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, err
	}

	if len(claims.Audience) == 0 {
		return nil, errors.New("missing audience")
	}

	return j.GenerateTokenPair(claims.DepartmentId, userID, claims.CompanyId, claims.Role, claims.Audience[0], claims.SessionId)
}
