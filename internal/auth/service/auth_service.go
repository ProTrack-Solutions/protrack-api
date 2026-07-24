package service

import (
	"context"
	"errors"

	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/domain"
	companiesDomain "github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	companiesService "github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	userDomain "github.com/ProTrack-Solutions/protrack-api/internal/users/domain"
	userService "github.com/ProTrack-Solutions/protrack-api/internal/users/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type Service struct {
	userService      *userService.Service
	companiesService *companiesService.Service
	jwtManager       *jwt.JWTManager
	pool             *pgxpool.Pool
}

func NewService(userService *userService.Service, companiesService *companiesService.Service, jwtManager *jwt.JWTManager, pool *pgxpool.Pool) *Service {
	return &Service{
		userService:      userService,
		companiesService: companiesService,
		jwtManager:       jwtManager,
		pool:             pool,
	}
}

func (s *Service) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	if req.Aud == "" {
		return &domain.LoginResponse{}, errors.New("invalid aud")
	}

	user, err := s.userService.ValidatePassword(ctx, req.Email, req.Password)
	if err != nil {
		return &domain.LoginResponse{}, err
	}

	var hasCompany bool

	if user.CompanyID != uuid.Nil {
		hasCompany = true
	} else {
		hasCompany = false
	}

	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.CompanyID, user.Role, req.Aud)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate tokens")
		return nil, err
	}

	return &domain.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		HasCompany:   hasCompany,
		ExpiresIn:    tokenPair.ExpireIn,
		TokenType:    "Bearer",
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResponse, error) {
	tokenPair, err := s.jwtManager.RefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpireIn,
		TokenType:    "Bearer",
	}, nil
}

/* func (s *Service) Logout(ctx context.Context, token string) error {
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return err
	}

	expiresIn := time.Until(claims.ExpiresAt.Time)
	if expiresIn <= 0 {
		return nil
	}

	return nil
} */

func (s *Service) GetUserFromContext(ctx context.Context, id uuid.UUID) (userDomain.UserResponse, error) {
	user, err := s.userService.GetUserByID(ctx, id)
	if err != nil {
		return userDomain.UserResponse{}, err
	}

	return userDomain.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		Email:          user.Email,
		Username:       user.Username,
		Role:           user.Role,
		Status:         user.Status,
		CompanyID:      user.CompanyID,
		DepartmentID:   user.DepartmentID,
		LastLoginAt:    user.LastLoginAt,
		CreatedBy:      user.CreatedBy,
		UpdatedBy:      user.UpdatedBy,
		DeletedBy:      user.DeletedBy,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
		DeletedAt:      user.DeletedAt,
		DepartmentName: user.DepartmentName,
	}, nil
}

func (s *Service) Register(ctx context.Context, req domain.RegisterRequest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	companyId, err := s.companiesService.CreateCompanyTx(ctx, tx, companiesDomain.CreateCompanyParams{
		Name:                req.Company.Name,
		TradeName:           req.Company.TradeName,
		Document:            req.Company.Document,
		Email:               req.Company.Email,
		Phone:               req.Company.Phone,
		Website:             req.Company.Website,
		AddressStreet:       req.Company.AddressStreet,
		AddressNumber:       req.Company.AddressNumber,
		AddressComplement:   req.Company.AddressComplement,
		AddressNeighborhood: req.Company.AddressNeighborhood,
		AddressCity:         req.Company.AddressCity,
		AddressState:        req.Company.AddressState,
		AddressZipcode:      req.Company.AddressZipcode,
		AddressCountry:      req.Company.AddressCountry,
		Timezone:            req.Company.Timezone,
		Status:              "ACTIVE",
		CreatedBy:           uuid.Nil,
	})
	if err != nil {
		return err
	}

	err = s.userService.CreateUserTx(ctx, tx, userDomain.CreateUserParams{
		Name:         req.User.Name,
		Email:        req.User.Email,
		Username:     req.User.Username,
		PasswordHash: req.User.PasswordHash,
		Role:         "ADMIN",
		Status:       "ACTIVE",
		CompanyID:    companyId,
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
