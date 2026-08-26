package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/validate"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	plansService "github.com/ProTrack-Solutions/protrack-api/internal/plans/service"
	subscriptionsService "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/service"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/users/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/users/repository"
)

type RepositoryInterface interface {
	CreateUsers(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	DeleteUser(ctx context.Context, id pgtype.UUID) error
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserById(ctx context.Context, id pgtype.UUID) (db.GetUserByIDRow, error)
	ListUsers(ctx context.Context) ([]db.User, error)
	UpdatePasswordHash(ctx context.Context, arg db.UpdatePasswordHashParams) error
	UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.User, error)
	UpdateUserCompanyAndRole(ctx context.Context, arg db.UpdateUserCompanyAndRoleParams) error
	UpdateLastLogin(ctx context.Context, id pgtype.UUID) error
	CountUsersByCompanyID(ctx context.Context, companyID pgtype.UUID) (int64, error)
	CountUsers(ctx context.Context) (int64, error)
	UpdateOwnProfile(ctx context.Context, req db.UpdateOwnProfileParams) error
	ListUsersByCompanyID(ctx context.Context, companyId pgtype.UUID) ([]db.User, error)
	WithTx(tx db.DBTX) *repository.Repository
}

type Service struct {
	plansService         *plansService.Service
	subscriptionsService *subscriptionsService.Service
	repo                 RepositoryInterface
	pool                 *pgxpool.Pool
	cfg                  *config.Config
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool, cfg *config.Config, plansService *plansService.Service, subscriptionsService *subscriptionsService.Service) *Service {
	return &Service{
		repo:                 repo,
		pool:                 pool,
		cfg:                  cfg,
		plansService:         plansService,
		subscriptionsService: subscriptionsService,
	}
}

func (s *Service) CreateUser(ctx context.Context, userId, companyId uuid.UUID, req domain.CreateUserParams) (domain.UserResponse, error) {
	sub, err := s.subscriptionsService.GetSubscriptionByCompanyID(ctx, companyId)
	if err != nil {
		return domain.UserResponse{}, err
	}

	plan, err := s.plansService.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return domain.UserResponse{}, err
	}

	contUsers, err := s.repo.CountUsersByCompanyID(ctx, pgconv.OptionalUUIDToPgType(companyId))
	if err != nil {
		return domain.UserResponse{}, err
	}

	for _, pf := range plan.Features {
		if pf.FeatureKey == "max_users" {
			if pf.LimitValue == contUsers {
				return domain.UserResponse{}, errors.New("users limit reached for plan")
			}
			break
		}
	}

	if err := validate.ValidPassword(req.Password); err != nil {
		return domain.UserResponse{}, err
	}

	is := validate.IsValidEmail(req.Email)
	if is == false {
		return domain.UserResponse{}, errors.New("invalid email")
	}

	// hashPassword, err := bcrypt.GenerateFromPassword([]byte(req.PasswordHash), 12)

	passwordPepper := req.Password + s.cfg.Pepper

	hashPassword, err := argon2id.CreateHash(passwordPepper, argon2id.DefaultParams)
	if err != nil {
		return domain.UserResponse{}, err
	}

	user, err := s.repo.CreateUsers(ctx, db.CreateUserParams{
		Name:         req.Name,
		Email:        req.Email,
		Username:     pgconv.ParseStringToPgText(req.Username),
		PasswordHash: string(hashPassword),
		Role:         "USER",
		Status:       "ACTIVE",
		CompanyID:    pgconv.ParseUUIDToPgType(companyId),
		DepartmentID: pgconv.ParseUUIDToPgType(req.DepartmentID),
		CreatedBy:    pgconv.ParseUUIDToPgType(userId),
		UpdatedBy:    pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return domain.UserResponse{}, err
	}

	return domain.UserResponse{
		ID:           pgconv.PgUUIDToUUID(user.ID),
		Name:         user.Name,
		Email:        user.Email,
		Username:     pgconv.ParsePgTextToString(user.Username),
		Role:         user.Role,
		Status:       user.Status,
		CompanyID:    pgconv.PgUUIDToUUID(user.CompanyID),
		DepartmentID: pgconv.PgUUIDToUUID(user.DepartmentID),
		LastLoginAt:  pgconv.PgTimestamptzToTime(user.LastLoginAt),
		CreatedBy:    pgconv.PgUUIDToUUID(user.CreatedBy),
		UpdatedBy:    pgconv.PgUUIDToUUID(user.UpdatedBy),
		DeletedBy:    pgconv.PgUUIDToUUID(user.DeletedBy),
		CreatedAt:    pgconv.PgTimestamptzToTime(user.CreatedAt),
		UpdatedAt:    pgconv.PgTimestamptzToTime(user.UpdatedAt),
		DeletedAt:    pgconv.PgTimestamptzToTime(user.DeletedAt),
	}, nil
}

func (s *Service) DeleteUser(ctx context.Context, id pgtype.UUID) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (domain.UserResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return domain.UserResponse{}, err
	}

	return domain.UserResponse{
		ID:           pgconv.PgUUIDToUUID(user.ID),
		Name:         user.Name,
		Email:        user.Email,
		Username:     pgconv.ParsePgTextToString(user.Username),
		Role:         user.Role,
		Status:       user.Status,
		CompanyID:    pgconv.PgUUIDToUUID(user.CompanyID),
		DepartmentID: pgconv.PgUUIDToUUID(user.DepartmentID),
		LastLoginAt:  pgconv.PgTimestamptzToTime(user.LastLoginAt),
		CreatedBy:    pgconv.PgUUIDToUUID(user.CreatedBy),
		UpdatedBy:    pgconv.PgUUIDToUUID(user.UpdatedBy),
		DeletedBy:    pgconv.PgUUIDToUUID(user.DeletedBy),
		CreatedAt:    pgconv.PgTimestamptzToTime(user.CreatedAt),
		UpdatedAt:    pgconv.PgTimestamptzToTime(user.UpdatedAt),
		DeletedAt:    pgconv.PgTimestamptzToTime(user.DeletedAt),
	}, nil
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (domain.UserResponse, error) {
	user, err := s.repo.GetUserById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.UserResponse{}, err
	}

	return domain.UserResponse{
		ID:             pgconv.PgUUIDToUUID(user.ID),
		Name:           user.Name,
		Email:          user.Email,
		Username:       pgconv.ParsePgTextToString(user.Username),
		Role:           user.Role,
		Status:         user.Status,
		CompanyID:      pgconv.PgUUIDToUUID(user.CompanyID),
		DepartmentID:   pgconv.PgUUIDToUUID(user.DepartmentID),
		LastLoginAt:    pgconv.PgTimestamptzToTime(user.LastLoginAt),
		CreatedBy:      pgconv.PgUUIDToUUID(user.CreatedBy),
		UpdatedBy:      pgconv.PgUUIDToUUID(user.UpdatedBy),
		DeletedBy:      pgconv.PgUUIDToUUID(user.DeletedBy),
		CreatedAt:      pgconv.PgTimestamptzToTime(user.CreatedAt),
		UpdatedAt:      pgconv.PgTimestamptzToTime(user.UpdatedAt),
		DeletedAt:      pgconv.PgTimestamptzToTime(user.DeletedAt),
		DepartmentName: pgconv.ParsePgTextToString(user.DepartmentName),
	}, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]domain.UserResponse, error) {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return []domain.UserResponse{}, err
	}

	var response []domain.UserResponse

	for _, user := range users {
		response = append(response, domain.UserResponse{
			ID:           pgconv.PgUUIDToUUID(user.ID),
			Name:         user.Name,
			Email:        user.Email,
			Username:     pgconv.ParsePgTextToString(user.Username),
			Role:         user.Role,
			Status:       user.Status,
			CompanyID:    pgconv.PgUUIDToUUID(user.CompanyID),
			DepartmentID: pgconv.PgUUIDToUUID(user.DepartmentID),
			LastLoginAt:  pgconv.PgTimestamptzToTime(user.LastLoginAt),
			CreatedBy:    pgconv.PgUUIDToUUID(user.CreatedBy),
			UpdatedBy:    pgconv.PgUUIDToUUID(user.UpdatedBy),
			DeletedBy:    pgconv.PgUUIDToUUID(user.DeletedBy),
			CreatedAt:    pgconv.PgTimestamptzToTime(user.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(user.UpdatedAt),
			DeletedAt:    pgconv.PgTimestamptzToTime(user.DeletedAt),
		})
	}

	return response, nil
}

func (s *Service) UpdatePasswordHash(ctx context.Context, userId uuid.UUID, req domain.UpdatePasswordParams) error {
	user, err := s.repo.GetUserById(ctx, pgconv.ParseUUIDToPgType(userId))
	if err != nil {
		return err
	}

	currentPasswordPepper := req.CurrentPassword + s.cfg.Pepper

	match, err := argon2id.ComparePasswordAndHash(currentPasswordPepper, user.PasswordHash)
	if err != nil {
		return errors.New("invalid credentials")
	}
	if !match {
		return errors.New("invalid credentials")
	}

	if err := validate.ValidPassword(req.Password); err != nil {
		return err
	}

	passwordPepper := req.Password + s.cfg.Pepper

	hashPassword, err := argon2id.CreateHash(passwordPepper, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	return s.repo.UpdatePasswordHash(ctx, db.UpdatePasswordHashParams{
		ID:           pgconv.ParseUUIDToPgType(userId),
		PasswordHash: hashPassword,
	})
}

func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, req domain.UpdateUserRequest) (domain.UserResponse, error) {
	user, err := s.repo.GetUserById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserResponse{}, fmt.Errorf("user not found")
		}
		return domain.UserResponse{}, err
	}

	if req.Email != "" {
		if !validate.IsValidEmail(req.Email) {
			return domain.UserResponse{}, errors.New("invalid email format")
		}
		existingUser, errEmail := s.repo.GetUserByEmail(ctx, req.Email)
		if errEmail == nil && existingUser.ID.Bytes != id {
			return domain.UserResponse{}, fmt.Errorf("email already in use")
		}
	}

	arg := db.UpdateUserParams{
		ID:           pgconv.ParseUUIDToPgType(id),
		Name:         user.Name,
		Email:        user.Email,
		Username:     user.Username,
		Status:       user.Status,
		DepartmentID: user.DepartmentID,
		UpdatedBy:    user.UpdatedBy,
	}

	domain.ApplyUpdateUserParams(req, &arg)

	updatedUser, err := s.repo.UpdateUser(ctx, arg)
	if err != nil {
		return domain.UserResponse{}, fmt.Errorf("failed to update user: %w", err)
	}

	return domain.UserResponse{
		ID:           pgconv.PgUUIDToUUID(updatedUser.ID),
		Name:         updatedUser.Name,
		Email:        updatedUser.Email,
		Username:     pgconv.ParsePgTextToString(updatedUser.Username),
		Role:         updatedUser.Role,
		Status:       updatedUser.Status,
		CompanyID:    pgconv.PgUUIDToUUID(updatedUser.CompanyID),
		DepartmentID: pgconv.PgUUIDToUUID(updatedUser.DepartmentID),
		LastLoginAt:  pgconv.PgTimestamptzToTime(updatedUser.LastLoginAt),
		CreatedBy:    pgconv.PgUUIDToUUID(updatedUser.CreatedBy),
		UpdatedBy:    pgconv.PgUUIDToUUID(updatedUser.UpdatedBy),
		DeletedBy:    pgconv.PgUUIDToUUID(updatedUser.DeletedBy),
		CreatedAt:    pgconv.PgTimestamptzToTime(updatedUser.CreatedAt),
		UpdatedAt:    pgconv.PgTimestamptzToTime(updatedUser.UpdatedAt),
		DeletedAt:    pgconv.PgTimestamptzToTime(updatedUser.DeletedAt),
	}, nil
}

func (s *Service) ValidatePassword(ctx context.Context, email string, password string) (domain.UserResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.UserResponse{}, err
	}
	defer tx.Rollback(ctx)

	txRepo := s.repo.WithTx(tx)

	user, err := txRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return domain.UserResponse{}, errors.New("invalid credentials")
	}

	/* err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		log.Error().Err(err).Msg("caiu no segundo if")
		return domain.UserResponse{}, errors.New("invalid credentials")
	} */

	passwordPepper := password + s.cfg.Pepper

	match, err := argon2id.ComparePasswordAndHash(passwordPepper, user.PasswordHash)
	if err != nil {
		return domain.UserResponse{}, errors.New("invalid credentials")
	}
	if !match {
		return domain.UserResponse{}, errors.New("invalid credentials")
	}

	if err := txRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		return domain.UserResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.UserResponse{}, err
	}

	return domain.UserResponse{
		ID:           pgconv.PgUUIDToUUID(user.ID),
		CompanyID:    pgconv.PgUUIDToUUID(user.CompanyID),
		DepartmentID: pgconv.PgUUIDToUUID(user.DepartmentID),
		Role:         user.Role,
	}, nil
}

func (s *Service) UpdateUserCompanyAndRole(ctx context.Context, req domain.UpdateUserCompanyAndRoleParams) error {
	return s.repo.UpdateUserCompanyAndRole(ctx, db.UpdateUserCompanyAndRoleParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		Role:      req.Role,
	})
}

// ResetPasswordHash define uma nova senha para o usuário SEM exigir a senha
// atual. Deve ser chamado apenas após validar um token de recuperação de
// senha (fluxo "esqueci minha senha") — para troca de senha autenticada,
// use UpdatePasswordHash.
func (s *Service) ResetPasswordHash(ctx context.Context, userId uuid.UUID, newPassword string) error {
	if err := validate.ValidPassword(newPassword); err != nil {
		return err
	}

	passwordPepper := newPassword + s.cfg.Pepper

	hashPassword, err := argon2id.CreateHash(passwordPepper, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	return s.repo.UpdatePasswordHash(ctx, db.UpdatePasswordHashParams{
		ID:           pgconv.ParseUUIDToPgType(userId),
		PasswordHash: hashPassword,
	})
}

func (s *Service) CreateUserTx(ctx context.Context, tx db.DBTX, companyId uuid.UUID, req domain.CreateUserParams) (uuid.UUID, error) {
	repoTx := db.New(tx)

	log.Info().Str("password", req.Password).Msg("password")

	if err := validate.ValidPassword(req.Password); err != nil {
		return uuid.Nil, err
	}

	is := validate.IsValidEmail(req.Email)
	if is == false {
		return uuid.Nil, errors.New("invalid email")
	}

	// hashPassword, err := bcrypt.GenerateFromPassword([]byte(req.PasswordHash), 12)

	passwordPepper := req.Password + s.cfg.Pepper

	hashPassword, err := argon2id.CreateHash(passwordPepper, argon2id.DefaultParams)
	if err != nil {
		return uuid.Nil, err
	}

	user, err := repoTx.CreateUser(ctx, db.CreateUserParams{
		Name:         req.Name,
		Email:        req.Email,
		Username:     pgconv.ParseStringToPgText(req.Username),
		PasswordHash: string(hashPassword),
		Role:         "ADMIN",
		Status:       "ACTIVE",
		CompanyID:    pgconv.ParseUUIDToPgType(companyId),
		CreatedAt:    pgconv.TimeToPgTimestamptz(time.Now()),
		CreatedBy:    pgconv.ParseUUIDToPgType(uuid.Nil),
	})
	if err != nil {
		return uuid.Nil, err
	}

	return pgconv.PgUUIDToUUID(user.ID), nil
}

func (s *Service) UpdateOwnProfile(ctx context.Context, userId uuid.UUID, req domain.UpdateOwnProfileRequest) error {
	user, err := s.repo.GetUserById(ctx, pgconv.ParseUUIDToPgType(userId))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user not found")
		}
		return err
	}

	arg := db.UpdateOwnProfileParams{
		Name:     user.Name,
		Email:    user.Email,
		Username: user.Username,
	}

	domain.ApplyUpdateOwnProfile(req, &arg)

	return s.repo.UpdateOwnProfile(ctx, arg)
}

func (s *Service) CountUsers(ctx context.Context) (int64, error) {
	return s.repo.CountUsers(ctx)
}

func (s *Service) ListUsersByCOmpany(ctx context.Context, companyId uuid.UUID) ([]domain.UserResponse, error) {
	users, err := s.repo.ListUsersByCompanyID(ctx, pgconv.OptionalUUIDToPgType(companyId))
	if err != nil {
		return []domain.UserResponse{}, err
	}

	var response []domain.UserResponse

	for _, user := range users {
		response = append(response, domain.UserResponse{
			ID:           pgconv.PgUUIDToUUID(user.ID),
			Name:         user.Name,
			Email:        user.Email,
			Username:     pgconv.ParsePgTextToString(user.Username),
			Role:         user.Role,
			Status:       user.Status,
			CompanyID:    pgconv.PgUUIDToUUID(user.CompanyID),
			DepartmentID: pgconv.PgUUIDToUUID(user.DepartmentID),
			LastLoginAt:  pgconv.PgTimestamptzToTime(user.LastLoginAt),
			CreatedBy:    pgconv.PgUUIDToUUID(user.CreatedBy),
			UpdatedBy:    pgconv.PgUUIDToUUID(user.UpdatedBy),
			DeletedBy:    pgconv.PgUUIDToUUID(user.DeletedBy),
			CreatedAt:    pgconv.PgTimestamptzToTime(user.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(user.UpdatedAt),
			DeletedAt:    pgconv.PgTimestamptzToTime(user.DeletedAt),
		})
	}

	return response, nil
}
