package domain

import (
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	Status       any       `json:"status"`
	CompanyID    uuid.UUID `json:"company_id"`
	DepartmentID uuid.UUID `json:"department_id"`
	LastLoginAt  time.Time `json:"last_login_at"`
	CreatedBy    uuid.UUID `json:"created_by"`
	UpdatedBy    uuid.UUID `json:"updated_by"`
	DeletedBy    uuid.UUID `json:"deleted_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    time.Time `json:"deleted_at"`
}

type CreateUserParams struct {
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	Status       any       `json:"status"`
	CompanyID    uuid.UUID `json:"company_id"`
	DepartmentID uuid.UUID `json:"department_id"`
	CreatedBy    uuid.UUID `json:"created_by"`
	UpdatedBy    uuid.UUID `json:"updated_by"`
	CreatedAt    time.Time `json:"created_at"`
}

type UpdatePasswordHashParams struct {
	ID           uuid.UUID `json:"id"`
	PasswordHash string    `json:"password_hash"`
}

type UpdateUserRequest struct {
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	Status       any       `json:"status"`
	DepartmentID uuid.UUID `json:"department_id"`
	UpdatedBy    uuid.UUID `json:"updated_by"`
}

type UserResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	Status       any       `json:"status"`
	CompanyID    uuid.UUID `json:"company_id"`
	DepartmentID uuid.UUID `json:"department_id"`
	LastLoginAt  time.Time `json:"last_login_at"`
	CreatedBy    uuid.UUID `json:"created_by"`
	UpdatedBy    uuid.UUID `json:"updated_by"`
	DeletedBy    uuid.UUID `json:"deleted_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    time.Time `json:"deleted_at"`
}

func ApplyUpdateUserParams(req UpdateUserRequest, arg *db.UpdateUserParams) {
	if req.Name != "" {
		arg.Name = req.Name
	}

	if req.Email != "" {
		arg.Email = req.Email
	}

	if req.Username != "" {
		arg.Username = pgconv.ParseStringToPgText(req.Username)
	}

	if req.Role != "" {
		arg.Role = req.Role
	}

	if req.Status != nil {
		arg.Status = req.Status
	}

	if req.DepartmentID != (uuid.UUID{}) {
		arg.DepartmentID = pgconv.ParseUUIDToPgType(req.DepartmentID)
	}

	if req.UpdatedBy != (uuid.UUID{}) {
		arg.UpdatedBy = pgconv.ParseUUIDToPgType(req.UpdatedBy)
	}
}

type UpdateUserCompanyAndRoleParams struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`
	Role      string    `json:"role"`
}
