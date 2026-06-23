package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/validate"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"

	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/users/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/users/repository"
)

type RepositoryInterface interface {
	CreateUsers(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	DeleteUser(ctx context.Context, id pgtype.UUID) error
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserById(ctx context.Context, id pgtype.UUID) (db.User, error)
	ListUsers(ctx context.Context) ([]db.User, error)
	UpdatePasswordHash(ctx context.Context, arg db.UpdatePasswordHashParams) error
	UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.User, error)
	UpdateUserCompanyAndRole(ctx context.Context, arg db.UpdateUserCompanyAndRoleParams) error
	UpdateLastLogin(ctx context.Context, id pgtype.UUID) error
	WithTx(tx db.DBTX) *repository.Repository
}

type Service struct {
	repo RepositoryInterface
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{
		repo: repo,
		pool: pool,
		cfg:  cfg,
	}
}

func (s *Service) CreateUser(ctx context.Context, req domain.CreateUserParams) (domain.UserResponse, error) {
	if err := validate.ValidPassword(req.PasswordHash); err != nil {
		return domain.UserResponse{}, err
	}

	is := validate.IsValidEmail(req.Email)
	if is == false {
		return domain.UserResponse{}, errors.New("invalid email")
	}

	// hashPassword, err := bcrypt.GenerateFromPassword([]byte(req.PasswordHash), 12)

	passwordPepper := req.PasswordHash + s.cfg.Pepper

	hashPassword, err := argon2id.CreateHash(passwordPepper, argon2id.DefaultParams)
	if err != nil {
		return domain.UserResponse{}, err
	}

	user, err := s.repo.CreateUsers(ctx, db.CreateUserParams{
		Name:         req.Name,
		Email:        req.Email,
		Username:     pgconv.ParseStringToPgText(req.Username),
		PasswordHash: string(hashPassword),
		Role:         req.Role,
		Status:       req.Status,
		CompanyID:    pgconv.ParseUUIDToPgType(req.CompanyID),
		DepartmentID: pgconv.ParseUUIDToPgType(req.DepartmentID),
		CreatedBy:    pgconv.ParseUUIDToPgType(req.CreatedBy),
		UpdatedBy:    pgconv.ParseUUIDToPgType(req.UpdatedBy),
		CreatedAt:    pgconv.TimeToPgTimestamptz(req.CreatedAt),
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

func (s *Service) UpdatePasswordHash(ctx context.Context, req domain.UpdatePasswordHashParams) error {
	if err := validate.ValidPassword(req.PasswordHash); err != nil {
		return err
	}

	return s.repo.UpdatePasswordHash(ctx, db.UpdatePasswordHashParams{
		ID:           pgconv.ParseUUIDToPgType(req.ID),
		PasswordHash: req.PasswordHash,
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
		Role:         user.Role,
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
		ID:        pgconv.PgUUIDToUUID(user.ID),
		CompanyID: pgconv.PgUUIDToUUID(user.CompanyID),
		Role:      user.Role,
	}, nil
}

func (s *Service) UpdateUserCompanyAndRole(ctx context.Context, req domain.UpdateUserCompanyAndRoleParams) error {
	return s.repo.UpdateUserCompanyAndRole(ctx, db.UpdateUserCompanyAndRoleParams{
		ID:        pgconv.ParseUUIDToPgType(req.ID),
		CompanyID: pgconv.ParseUUIDToPgType(req.CompanyID),
		Role:      req.Role,
	})
}
