package service

import (
	"context"
	"time"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RepositoryInterface interface {
	CreatePlans(ctx context.Context, arg db.CreatePlanParams) error
	GetPlanByID(ctx context.Context, planId pgtype.UUID) (db.Plan, error)
	ListPlans(ctx context.Context) ([]db.Plan, error)
	ListPlansByActiveStatus(ctx context.Context, active pgtype.Bool) ([]db.Plan, error)
	UpdatePlan(ctx context.Context, arg db.UpdatePlanParams) error
	TogglePlanActiveStatus(ctx context.Context, arg db.TogglePlanActiveStatusParams) error
}

type Service struct {
	repo RepositoryInterface
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreatePlans(ctx context.Context, req domain.CreatePlanRequest) error {
	priceCents := req.ValueAmount * 100

	if err := s.repo.CreatePlans(ctx, db.CreatePlanParams{
		Name:         req.Name,
		Description:  pgconv.ParseStringToPgText(req.Description),
		Currency:     pgconv.ParseStringToPgText(req.Currency),
		BillingCycle: req.BillingCycle,
		Active:       pgconv.BoolToPgBool(true),
		CreatedAt:    pgconv.TimeToPgTimestamptz(time.Now()),
		PriceCents:   int32(priceCents),
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) GetPlanByID(ctx context.Context, planId uuid.UUID) (domain.PlanResponse, error) {
	plan, err := s.repo.GetPlanByID(ctx, pgconv.ParseUUIDToPgType(planId))
	if err != nil {
		return domain.PlanResponse{}, err
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
	}, nil
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.PlanResponse, error) {
	plans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, err
	}

	var planResponses []domain.PlanResponse
	for _, plan := range plans {
		planResponses = append(planResponses, domain.PlanResponse{
			ID:           pgconv.PgUUIDToUUID(plan.ID),
			Name:         plan.Name,
			Description:  pgconv.ParsePgTextToString(plan.Description),
			PriceCents:   plan.PriceCents,
			Currency:     pgconv.ParsePgTextToString(plan.Currency),
			BillingCycle: plan.BillingCycle,
			Active:       pgconv.PgBoolToBool(plan.Active),
			CreatedAt:    pgconv.PgTimestamptzToTime(plan.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(plan.UpdatedAt),
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
		planResponses = append(planResponses, domain.PlanResponse{
			ID:           pgconv.PgUUIDToUUID(plan.ID),
			Name:         plan.Name,
			Description:  pgconv.ParsePgTextToString(plan.Description),
			PriceCents:   plan.PriceCents,
			Currency:     pgconv.ParsePgTextToString(plan.Currency),
			BillingCycle: plan.BillingCycle,
			Active:       pgconv.PgBoolToBool(plan.Active),
			CreatedAt:    pgconv.PgTimestamptzToTime(plan.CreatedAt),
			UpdatedAt:    pgconv.PgTimestamptzToTime(plan.UpdatedAt),
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
