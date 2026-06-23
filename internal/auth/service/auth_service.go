package service

import (
	"context"
	"errors"

	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/domain"
	userDomain "github.com/ProTrack-Solutions/protrack-api/internal/users/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/users/service"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Service struct {
	userService *service.Service
	jwtManager  *jwt.JWTManager
}

func NewService(userService *service.Service, jwtManager *jwt.JWTManager) *Service {
	return &Service{
		userService: userService,
		jwtManager:  jwtManager,
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
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		Username:     user.Username,
		Role:         user.Role,
		Status:       user.Status,
		CompanyID:    user.CompanyID,
		DepartmentID: user.DepartmentID,
		LastLoginAt:  user.LastLoginAt,
		CreatedBy:    user.CreatedBy,
		UpdatedBy:    user.UpdatedBy,
		DeletedBy:    user.DeletedBy,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		DeletedAt:    user.DeletedAt,
	}, nil
}
