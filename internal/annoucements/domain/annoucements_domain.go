package domain

import (
	"time"

	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/google/uuid"
)

type AnnouncementType string

const (
	AnnouncementTypeInfo        AnnouncementType = "info"
	AnnouncementTypeWarning     AnnouncementType = "warning"
	AnnouncementTypeSuccess     AnnouncementType = "success"
	AnnouncementTypeMaintenance AnnouncementType = "maintenance"
)

type CreateAnnouncementsRequest struct {
	Title     string    `json:"title" binding:"required,max=150" example:"Manutenção Programada"`
	Content   string    `json:"content" binding:"required" example:"O sistema ficará instável das 02h às 04h."`
	Type      string    `json:"type" binding:"required" enums:"info,warning,success,maintenance" example:"info"`
	StartsAt  time.Time `json:"starts_at" example:"2026-07-07T12:00:00Z"`
	ExpiresAt time.Time `json:"expires_at" example:"2026-07-08T12:00:00Z"`
}

type ListAnnoucementsResponse struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Type      string    `json:"type" enums:"info,warning,success,maintenance" example:"info"`
	IsActive  bool      `json:"is_active"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type ListAnnoucementsPaginateResponse struct {
	globalDomain.PaginatedResponse[ListAnnoucementsResponse]
}
