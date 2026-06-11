package domain

import (
	"time"

	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	db "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/database/sqlc"
	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/domain/enums"
	"github.com/google/uuid"
)

type ProductCategory struct {
	ID        uuid.UUID    `json:"id"`
	CompanyID uuid.UUID    `json:"company_id"`
	Name      string       `json:"name"`
	Color     string       `json:"color"`
	Status    enums.Status `json:"status"`
	CreatedBy uuid.UUID    `json:"created_by"`
	UpdatedBy uuid.UUID    `json:"updated_by"`
	DeletedBy uuid.UUID    `json:"deleted_by"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	DeletedAt time.Time    `json:"deleted_at"`
}

type CreateProductCategoryRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type DeleteProductCategoryRequest struct {
	ID        uuid.UUID `json:"id"`
	DeletedBy uuid.UUID `json:"deleted_by"`
}

type SetProductCategoryStatusRequest struct {
	ID     uuid.UUID    `json:"id"`
	Status enums.Status `json:"status"`
}

type UpdateProductCategoryRequest struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	UpdatedBy uuid.UUID `json:"updated_by"`
}

type ProductCategoryResponse struct {
	ID        uuid.UUID    `json:"id"`
	CompanyID uuid.UUID    `json:"company_id"`
	Name      string       `json:"name"`
	Color     string       `json:"color"`
	Status    enums.Status `json:"status"`
	CreatedBy uuid.UUID    `json:"created_by"`
	UpdatedBy uuid.UUID    `json:"updated_by"`
	DeletedBy uuid.UUID    `json:"deleted_by"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	DeletedAt time.Time    `json:"deleted_at"`
}

func ApplyUpdateProductCategoryParams(
	req UpdateProductCategoryRequest,
	arg *db.UpdateProductCategoryParams,
) {
	if req.Name != "" {
		arg.Name = req.Name
	}

	if req.Color != "" {
		arg.Color = pgconv.ParseStringToPgText(req.Color)
	}

	if req.UpdatedBy != (uuid.UUID{}) {
		arg.UpdatedBy = pgconv.ParseUUIDToPgType(req.UpdatedBy)
	}
}
