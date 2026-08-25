package domain

import (
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/domain/enums"
	"github.com/google/uuid"
)

type Department struct {
	ID          uuid.UUID `json:"id"`
	CompanyID   uuid.UUID `json:"company_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedBy   uuid.UUID `json:"created_by"`
	UpdatedBy   uuid.UUID `json:"updated_by"`
	DeletedBy   uuid.UUID `json:"deleted_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   time.Time `json:"deleted_at"`
}

type CreateDepartmentParams struct {
	Name        string                                   `json:"name"`
	Description string                                   `json:"description"`
	ModuleCode  []departmentModulesDomain.ModuleResponse `json:"modules"`
}

type SetStatusDepartmentParams struct {
	Status enums.Status `json:"Status"`
}

type UpdateDepartmentParams struct {
	Name        string                                   `json:"name"`
	Description string                                   `json:"description"`
	ModuleCode  []departmentModulesDomain.ModuleResponse `json:"modules"`
}

type DepartmentResponse struct {
	ID          uuid.UUID                                `json:"id"`
	CompanyID   uuid.UUID                                `json:"company_id"`
	Name        string                                   `json:"name"`
	Description string                                   `json:"description"`
	Status      string                                   `json:"status"`
	CreatedBy   uuid.UUID                                `json:"created_by"`
	UpdatedBy   uuid.UUID                                `json:"updated_by"`
	DeletedBy   uuid.UUID                                `json:"deleted_by"`
	CreatedAt   time.Time                                `json:"created_at"`
	UpdatedAt   time.Time                                `json:"updated_at"`
	DeletedAt   time.Time                                `json:"deleted_at"`
	Modules     []departmentModulesDomain.ModuleResponse `json:"modules"`
}

func ApplyUpdateProductCategoryParams(
	req UpdateDepartmentParams,
	arg *db.UpdateDepartmentParams,
) {
	if req.Name != "" {
		arg.Name = req.Name
	}

	if req.Description != "" {
		arg.Description = pgconv.ParseStringToPgText(req.Description)
	}
}
