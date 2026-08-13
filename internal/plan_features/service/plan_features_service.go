package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/plan_features/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/plan_features/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreatePlanFeatures(ctx context.Context, arg db.CreatePlanFeatureParams) error
	ListFeaturesByPlanID(ctx context.Context, planId pgtype.UUID) ([]db.PlanFeature, error)
	DeletePlanFeature(ctx context.Context, id pgtype.UUID) error
	ListFeaturesActiveByPlanID(ctx context.Context, planId pgtype.UUID) ([]db.PlanFeature, error)
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

func (s *Service) CreatePlanFeatureTx(ctx context.Context, tx db.DBTX, planId uuid.UUID, req []domain.CreatePlanFeatureRequest) error {
	repoTx := db.New(tx)
	for _, pf := range req {
		if err := repoTx.CreatePlanFeature(ctx, db.CreatePlanFeatureParams{
			PlanID:       pgconv.ParseUUIDToPgType(planId),
			Name:         pf.Name,
			IsEnabled:    pf.IsEnabled,
			DisplayOrder: pf.DisplayOrder,
			FeatureKey:   pf.FeatureKey,
			LimitValue:   pgconv.IntToPgInt4(int(pf.LimitValue)),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) ListFeaturesByPlanID(ctx context.Context, planId uuid.UUID) ([]domain.PlanFeatureResponse, error) {
	features, err := s.repo.ListFeaturesByPlanID(ctx, pgconv.ParseUUIDToPgType(planId))
	if err != nil {
		return []domain.PlanFeatureResponse{}, err
	}

	var response []domain.PlanFeatureResponse

	for _, fe := range features {
		response = append(response, domain.PlanFeatureResponse{
			ID:           pgconv.PgUUIDToUUID(fe.ID),
			PlanID:       pgconv.PgUUIDToUUID(fe.PlanID),
			Name:         fe.Name,
			IsEnabled:    fe.IsEnabled,
			DisplayOrder: fe.DisplayOrder,
			CreatedAt:    pgconv.PgTimestamptzToTime(fe.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(fe.UpdatedAt),
			FeatureKey:   fe.FeatureKey,
			LimitValue:   int64(fe.LimitValue.Int32),
		})
	}

	return response, nil
}

func (s *Service) DeletePlanFeature(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeletePlanFeature(ctx, pgconv.ParseUUIDToPgType(id))
}

func (s *Service) ListFeaturesActiveByPlanID(ctx context.Context, planId uuid.UUID) ([]domain.PlanFeatureResponse, error) {
	features, err := s.repo.ListFeaturesActiveByPlanID(ctx, pgconv.ParseUUIDToPgType(planId))
	if err != nil {
		return []domain.PlanFeatureResponse{}, err
	}

	var response []domain.PlanFeatureResponse

	for _, fe := range features {
		response = append(response, domain.PlanFeatureResponse{
			ID:           pgconv.PgUUIDToUUID(fe.ID),
			PlanID:       pgconv.PgUUIDToUUID(fe.PlanID),
			Name:         fe.Name,
			IsEnabled:    fe.IsEnabled,
			DisplayOrder: fe.DisplayOrder,
			CreatedAt:    pgconv.PgTimestamptzToTime(fe.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(fe.UpdatedAt),
			FeatureKey:   fe.FeatureKey,
			LimitValue:   int64(fe.LimitValue.Int32),
		})
	}

	return response, nil
}
