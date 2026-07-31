package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	plansFeatureDomain "github.com/ProTrack-Solutions/protrack-api/internal/plan_features/domain"
	planFeatureService "github.com/ProTrack-Solutions/protrack-api/internal/plan_features/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreatePlans(ctx context.Context, arg db.CreatePlanParams) (pgtype.UUID, error)
	GetPlanByID(ctx context.Context, planId pgtype.UUID) (db.Plan, error)
	ListPlans(ctx context.Context) ([]db.Plan, error)
	ListPlansByActiveStatus(ctx context.Context, active pgtype.Bool) ([]db.Plan, error)
	UpdatePlan(ctx context.Context, arg db.UpdatePlanParams) error
	TogglePlanActiveStatus(ctx context.Context, arg db.TogglePlanActiveStatusParams) error
	WithTx(tx db.DBTX) *repository.Repository
}

type PlanFeatureServiceInterface interface {
	CreatePlanFeatureTx(ctx context.Context, tx db.DBTX, planId uuid.UUID, req []plansFeatureDomain.CreatePlanFeatureRequest) error
	ListFeaturesByPlanID(ctx context.Context, planId uuid.UUID) ([]plansFeatureDomain.PlanFeatureResponse, error)
}

type Service struct {
	repo                RepositoryInterface
	plansFeatureService PlanFeatureServiceInterface
	pool                *pgxpool.Pool
}

func NewService(repo *repository.Repository, plansFeatureService *planFeatureService.Service, pool *pgxpool.Pool) *Service {
	return &Service{
		repo:                repo,
		plansFeatureService: plansFeatureService,
		pool:                pool,
	}
}

func (s *Service) CreatePlans(ctx context.Context, req domain.CreatePlanRequest) error {
	priceCents := req.ValueAmount * 100

	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		txRepo := s.repo.WithTx(tx)

		id, err := txRepo.CreatePlans(ctx, db.CreatePlanParams{
			Name:         req.Name,
			Description:  pgconv.ParseStringToPgText(req.Description),
			Currency:     pgconv.ParseStringToPgText(req.Currency),
			BillingCycle: req.BillingCycle,
			Active:       pgconv.BoolToPgBool(true),
			ExternalID:   req.ExternalID,
			Highlight:    req.Highlight,
			Icon:         req.Icon,
			PriceCents:   int32(priceCents),
		})
		if err != nil {
			return err
		}

		if s.plansFeatureService != nil && len(req.Features) > 0 {
			if err := s.plansFeatureService.CreatePlanFeatureTx(ctx, tx, pgconv.PgUUIDToUUID(id), req.Features); err != nil {
				return err
			}
		}

		return tx.Commit(ctx)
	}

	id, err := s.repo.CreatePlans(ctx, db.CreatePlanParams{
		Name:         req.Name,
		Description:  pgconv.ParseStringToPgText(req.Description),
		Currency:     pgconv.ParseStringToPgText(req.Currency),
		BillingCycle: req.BillingCycle,
		Active:       pgconv.BoolToPgBool(true),
		ExternalID:   req.ExternalID,
		Highlight:    req.Highlight,
		Icon:         req.Icon,
		PriceCents:   int32(priceCents),
	})
	if err != nil {
		return err
	}

	if s.plansFeatureService != nil && len(req.Features) > 0 {
		if err := s.plansFeatureService.CreatePlanFeatureTx(ctx, nil, pgconv.PgUUIDToUUID(id), req.Features); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetPlanByID(ctx context.Context, planId uuid.UUID) (domain.PlanResponse, error) {
	plan, err := s.repo.GetPlanByID(ctx, pgconv.ParseUUIDToPgType(planId))
	if err != nil {
		return domain.PlanResponse{}, err
	}

	var features []plansFeatureDomain.PlanFeatureResponse
	if s.plansFeatureService != nil {
		features, err = s.plansFeatureService.ListFeaturesByPlanID(ctx, planId)
		if err != nil {
			return domain.PlanResponse{}, err
		}
	}

	return domain.PlanResponse{
		ID:           pgconv.PgUUIDToUUID(plan.ID),
		Name:         plan.Name,
		Description:  pgconv.ParsePgTextToString(plan.Description),
		PriceCents:   plan.PriceCents,
		Currency:     pgconv.ParsePgTextToString(plan.Currency),
		BillingCycle: plan.BillingCycle,
		Active:       pgconv.PgBoolToBool(plan.Active),
		CreatedAt:    pgconv.PgTimestamptzToTime(plan.CreatedAt),
		UpdatedAt:    pgconv.PgTimestamptzToTime(plan.UpdatedAt),
		ExternalId:   plan.ExternalID,
		Features:     features,
	}, nil
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.PlanResponse, error) {
	plans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, err
	}

	var planResponses []domain.PlanResponse
	for _, plan := range plans {
		planId := pgconv.PgUUIDToUUID(plan.ID)

		var features []plansFeatureDomain.PlanFeatureResponse
		if s.plansFeatureService != nil {
			features, err = s.plansFeatureService.ListFeaturesByPlanID(ctx, planId)
			if err != nil {
				return nil, err
			}
		}

		planResponses = append(planResponses, domain.PlanResponse{
			ID:           planId,
			Name:         plan.Name,
			Description:  pgconv.ParsePgTextToString(plan.Description),
			PriceCents:   plan.PriceCents,
			Currency:     pgconv.ParsePgTextToString(plan.Currency),
			BillingCycle: plan.BillingCycle,
			Active:       pgconv.PgBoolToBool(plan.Active),
			CreatedAt:    pgconv.PgTimestamptzToTime(plan.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(plan.UpdatedAt),
			Features:     features,
		})
	}

	return planResponses, nil
}

func (s *Service) ListPlansByActiveStatus(ctx context.Context, active bool) ([]domain.PlanResponse, error) {
	plans, err := s.repo.ListPlansByActiveStatus(ctx, pgconv.BoolToPgBool(active))
	if err != nil {
		return nil, err
	}

	var planResponses []domain.PlanResponse
	for _, plan := range plans {
		planId := pgconv.PgUUIDToUUID(plan.ID)

		var features []plansFeatureDomain.PlanFeatureResponse
		if s.plansFeatureService != nil {
			features, err = s.plansFeatureService.ListFeaturesByPlanID(ctx, planId)
			if err != nil {
				return nil, err
			}
		}

		planResponses = append(planResponses, domain.PlanResponse{
			ID:           planId,
			Name:         plan.Name,
			Description:  pgconv.ParsePgTextToString(plan.Description),
			PriceCents:   plan.PriceCents,
			Currency:     pgconv.ParsePgTextToString(plan.Currency),
			BillingCycle: plan.BillingCycle,
			Active:       pgconv.PgBoolToBool(plan.Active),
			CreatedAt:    pgconv.PgTimestamptzToTime(plan.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(plan.UpdatedAt),
			ExternalId:   plan.ExternalID,
			Highlight:    plan.Highlight,
			Icon:         plan.Icon,
			Features:     features,
		})
	}

	return planResponses, nil
}

func (s *Service) UpdatePlan(ctx context.Context, planId uuid.UUID, req domain.UpdatePlanParams) error {
	currentPlan, err := s.repo.GetPlanByID(ctx, pgconv.ParseUUIDToPgType(planId))
	if err != nil {
		return err
	}

	arg := db.UpdatePlanParams{
		ID:           currentPlan.ID,
		Name:         currentPlan.Name,
		Description:  currentPlan.Description,
		Currency:     currentPlan.Currency,
		BillingCycle: currentPlan.BillingCycle,
		PriceCents:   currentPlan.PriceCents,
	}

	priceCents := req.ValueAmount * 100

	if priceCents == 0 {
		priceCents = float64(currentPlan.PriceCents)
	}

	domain.ApplyUpdatePlanParams(req, &arg)

	return s.repo.UpdatePlan(ctx, db.UpdatePlanParams{
		ID:           pgconv.ParseUUIDToPgType(planId),
		Name:         arg.Name,
		Description:  arg.Description,
		PriceCents:   int32(priceCents),
		Currency:     arg.Currency,
		BillingCycle: arg.BillingCycle,
	})
}

func (s *Service) TogglePlanActiveStatus(ctx context.Context, planId uuid.UUID, active bool) error {
	return s.repo.TogglePlanActiveStatus(ctx, db.TogglePlanActiveStatusParams{
		ID:     pgconv.ParseUUIDToPgType(planId),
		Active: pgconv.BoolToPgBool(active),
	})
}
