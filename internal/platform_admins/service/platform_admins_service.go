package service

import (
	"context"
	"errors"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/platform_admins/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/platform_admins/repository"
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
)

type RepositoryInterface interface {
	GetPlatformAdminByEmail(ctx context.Context, email string) (db.PlatformAdmin, error)
}

type Service struct {
	repo       RepositoryInterface
	jwtManager *jwt.JWTManager
	cfg        *config.Config
}

func NewService(repo *repository.Repository, cfg *config.Config, jwtManager *jwt.JWTManager) *Service {
	return &Service{
		repo:       repo,
		cfg:        cfg,
		jwtManager: jwtManager,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (*domain.LoginResponse, error) {
	admin, err := s.repo.GetPlatformAdminByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	match, err := argon2id.ComparePasswordAndHash(password+s.cfg.SuperAdiminPepper, admin.PasswordHash)
	if err != nil || !match {
		return nil, errors.New("invalid credentials")
	}

	tokenPair, err := s.jwtManager.GenerateTokenPair(uuid.Nil, pgconv.PgUUIDToUUID(admin.ID), uuid.Nil, "SUPER_ADMIN", "platform")
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
