package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/ProTrack-Solutions/protrack-api/internal/annoucements/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/annoucements/repository"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	globalDomain "github.com/ProTrack-Solutions/protrack-api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateAnnoucements(ctx context.Context, arg db.CreateAnnouncementsParams) error
	ListAnnoucements(ctx context.Context, arg db.ListAnnoucementsParams) ([]db.ListAnnoucementsRow, error)
	DeleteAnnoucements(ctx context.Context, arg db.DeleteAnnoucementsParams) error
	CountAnnoucementsByCompany(ctx context.Context, companyId pgtype.UUID) (int64, error)
}

type Service struct {
	repo RepositoryInterface
	pool *pgxpool.Pool
}

func NewService(repo *repository.Repository, pool *pgxpool.Pool) *Service {
	return &Service{
		repo: repo,
		pool: pool,
	}
}

func (s *Service) CreateAnnoucements(ctx context.Context, userId uuid.UUID, companyId uuid.UUID, req domain.CreateAnnouncementsRequest) error {
	return s.repo.CreateAnnoucements(ctx, db.CreateAnnouncementsParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Title:     req.Title,
		Content:   req.Content,
		Type:      db.AnnouncementType(req.Type),
		Column5:   req.StartsAt,
		ExpiresAt: pgconv.TimeToPgTimestamptz(req.ExpiresAt),
		IsActive:  true,
		CreatedBy: pgconv.ParseUUIDToPgType(userId),
	})
}

func (s *Service) ListAnnoucements(ctx context.Context, companyId uuid.UUID, pagination globalDomain.PaginationParams) (domain.ListAnnoucementsPaginateResponse, error) {
	if pagination.Page < 1 {
		pagination.Page = 1
	}
	if pagination.PerPage < 1 {
		pagination.PerPage = 10
	}

	total, err := s.repo.CountAnnoucementsByCompany(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return domain.ListAnnoucementsPaginateResponse{}, err
	}

	annoucements, err := s.repo.ListAnnoucements(ctx, db.ListAnnoucementsParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
		Limit:     pagination.PerPage,
		Offset:    (pagination.Page - 1) * pagination.PerPage,
	})
	if err != nil {
		return domain.ListAnnoucementsPaginateResponse{}, err
	}

	var response []domain.ListAnnoucementsResponse

	for _, annoucement := range annoucements {
		response = append(response, domain.ListAnnoucementsResponse{
			ID:        pgconv.PgUUIDToUUID(annoucement.ID),
			Title:     annoucement.Title,
			Type:      string(annoucement.Type),
			IsActive:  annoucement.IsActive,
			StartsAt:  pgconv.PgTimestamptzToTime(annoucement.StartsAt),
			ExpiresAt: pgconv.PgTimestamptzToTime(annoucement.ExpiresAt),
			CreatedAt: pgconv.PgTimestamptzToTime(annoucement.CreatedAt),
		})
	}

	paginationResponse := globalDomain.NewPaginatedResponse(response, total, pagination)

	return domain.ListAnnoucementsPaginateResponse{
		PaginatedResponse: paginationResponse,
	}, nil
}

func (s *Service) DeleteAnnoucements(ctx context.Context, Id uuid.UUID, companyId uuid.UUID, userId uuid.UUID) error {
	return s.repo.DeleteAnnoucements(ctx, db.DeleteAnnoucementsParams{
		DeletedBy: pgconv.ParseUUIDToPgType(userId),
		ID:        pgconv.ParseUUIDToPgType(Id),
		CompanyID: pgconv.ParseUUIDToPgType(companyId),
	})
}
