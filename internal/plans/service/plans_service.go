package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	plansFeatureDomain "github.com/ProTrack-Solutions/protrack-api/internal/plan_features/domain"
	planFeatureService "github.com/ProTrack-Solutions/protrack-api/internal/plan_features/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/plans/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v86"
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
	ListFeaturesActiveByPlanID(ctx context.Context, planId uuid.UUID) ([]plansFeatureDomain.PlanFeatureResponse, error)
}

type Service struct {
	client              *stripe.Client
	repo                RepositoryInterface
	plansFeatureService PlanFeatureServiceInterface
	pool                *pgxpool.Pool
}

func NewService(client *stripe.Client, repo *repository.Repository, plansFeatureService *planFeatureService.Service, pool *pgxpool.Pool) *Service {
	return &Service{
		client:              client,
		repo:                repo,
		plansFeatureService: plansFeatureService,

		pool: pool,
	}
}

func (s *Service) CreatePlans(ctx context.Context, req domain.CreatePlanRequest) error {
	priceCents := math.Round(req.ValueAmount * 100)

	var intervalStripe string
	switch strings.ToLower(req.BillingCycle) {
	case "monthly":
		intervalStripe = string(stripe.PriceRecurringIntervalMonth)
	case "yearly":
		intervalStripe = string(stripe.PriceRecurringIntervalYear)
	default:
		return fmt.Errorf("billing_cycle inválido: %s", req.BillingCycle)
	}

	var productID, defaultPriceID string
	if s.client != nil {
		params := &stripe.ProductCreateParams{
			Name:        stripe.String(req.Name),
			Description: stripe.String(req.Description),
			DefaultPriceData: &stripe.ProductCreateDefaultPriceDataParams{
				Currency:   stripe.String(req.Currency),
				UnitAmount: stripe.Int64(int64(priceCents)),
				Recurring: &stripe.ProductCreateDefaultPriceDataRecurringParams{
					Interval:      stripe.String(intervalStripe),
					IntervalCount: stripe.Int64(1),
				},
			},
			Params: stripe.Params{
				Expand: []*string{stripe.String("default_price")},
			},
		}

		product, err := s.client.V1Products.Create(ctx, params)
		if err != nil {
			return err
		}
		productID = product.ID
		if product.DefaultPrice != nil {
			defaultPriceID = product.DefaultPrice.ID
		}
	}

	if s.pool != nil {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		txRepo := s.repo.WithTx(tx)

		id, err := txRepo.CreatePlans(ctx, db.CreatePlanParams{
			Name:            req.Name,
			Description:     pgconv.ParseStringToPgText(req.Description),
			Currency:        pgconv.ParseStringToPgText(req.Currency),
			BillingCycle:    req.BillingCycle,
			Active:          pgconv.BoolToPgBool(true),
			ExternalID:      productID,
			ExternalPriceID: defaultPriceID,
			Highlight:       req.Highlight,
			Icon:            req.Icon,
			PriceCents:      int32(priceCents),
		})
		if err != nil {
			if s.client != nil && productID != "" {
				s.client.V1Products.Delete(ctx, productID, nil)
			}
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
		Name:            req.Name,
		Description:     pgconv.ParseStringToPgText(req.Description),
		Currency:        pgconv.ParseStringToPgText(req.Currency),
		BillingCycle:    req.BillingCycle,
		Active:          pgconv.BoolToPgBool(true),
		ExternalID:      productID,
		ExternalPriceID: defaultPriceID,
		Highlight:       req.Highlight,
		Icon:            req.Icon,
		PriceCents:      int32(priceCents),
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
		ID:              pgconv.PgUUIDToUUID(plan.ID),
		Name:            plan.Name,
		Description:     pgconv.ParsePgTextToString(plan.Description),
		PriceCents:      plan.PriceCents,
		Currency:        pgconv.ParsePgTextToString(plan.Currency),
		BillingCycle:    plan.BillingCycle,
		Active:          pgconv.PgBoolToBool(plan.Active),
		CreatedAt:       pgconv.PgTimestamptzToTime(plan.CreatedAt),
		UpdatedAt:       pgconv.PgTimestamptzToTime(plan.UpdatedAt),
		ExternalId:      plan.ExternalID,
		ExternalPriceId: plan.ExternalPriceID,
		Features:        features,
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
			features, err = s.plansFeatureService.ListFeaturesActiveByPlanID(ctx, planId)
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
	domain.ApplyUpdatePlanParams(req, &arg)

	priceChanged := req.ValueAmount != 0 && int32(math.Round(req.ValueAmount*100)) != currentPlan.PriceCents
	cycleChanged := req.BillingCycle != "" && req.BillingCycle != currentPlan.BillingCycle

	newPriceID := currentPlan.ExternalPriceID

	if s.client != nil {
		// 1. Atualiza dados simples do produto no Stripe (nome/descrição)
		productParams := &stripe.ProductUpdateParams{}
		hasProductChange := false
		if req.Name != "" && req.Name != currentPlan.Name {
			productParams.Name = stripe.String(req.Name)
			hasProductChange = true
		}
		if req.Description != "" {
			productParams.Description = stripe.String(req.Description)
			hasProductChange = true
		}
		if hasProductChange {
			if _, err := s.client.V1Products.Update(ctx, currentPlan.ExternalID, productParams); err != nil {
				return fmt.Errorf("erro ao atualizar produto no Stripe: %w", err)
			}
		}

		// 2. Se preço ou ciclo mudou, cria Price novo e arquiva o antigo
		if priceChanged || cycleChanged {
			var intervalStripe string
			cycle := currentPlan.BillingCycle
			if cycleChanged {
				cycle = req.BillingCycle
			}
			switch strings.ToLower(cycle) {
			case "monthly":
				intervalStripe = string(stripe.PriceRecurringIntervalMonth)
			case "yearly":
				intervalStripe = string(stripe.PriceRecurringIntervalYear)
			default:
				return fmt.Errorf("billing_cycle inválido: %s", cycle)
			}

			amountCents := int64(currentPlan.PriceCents)
			if priceChanged {
				amountCents = int64(math.Round(req.ValueAmount * 100))
			}

			priceParams := &stripe.PriceCreateParams{
				Currency:   stripe.String(currentPlan.Currency.String),
				Product:    stripe.String(currentPlan.ExternalID),
				UnitAmount: stripe.Int64(amountCents),
				Recurring: &stripe.PriceCreateRecurringParams{
					Interval: stripe.String(intervalStripe),
				},
			}

			newPrice, err := s.client.V1Prices.Create(ctx, priceParams)
			if err != nil {
				return fmt.Errorf("erro ao criar novo preço no Stripe: %w", err)
			}

			// seta o novo price como default do produto
			_, err = s.client.V1Products.Update(ctx, currentPlan.ExternalID, &stripe.ProductUpdateParams{
				DefaultPrice: stripe.String(newPrice.ID),
			})
			if err != nil {
				return fmt.Errorf("erro ao definir novo preço padrão: %w", err)
			}

			// arquiva o preço antigo (impede uso em novas assinaturas)
			if currentPlan.ExternalPriceID != "" {
				_, err = s.client.V1Prices.Update(ctx, currentPlan.ExternalPriceID, &stripe.PriceUpdateParams{
					Active: stripe.Bool(false),
				})
				if err != nil {
					return fmt.Errorf("erro ao arquivar preço antigo: %w", err)
				}
			}

			newPriceID = newPrice.ID
			arg.PriceCents = int32(amountCents)
		}
	}

	return s.repo.UpdatePlan(ctx, db.UpdatePlanParams{
		ID:              pgconv.ParseUUIDToPgType(planId),
		Name:            arg.Name,
		Description:     arg.Description,
		PriceCents:      arg.PriceCents,
		Currency:        arg.Currency,
		BillingCycle:    arg.BillingCycle,
		ExternalPriceID: newPriceID,
	})
}

func (s *Service) TogglePlanActiveStatus(ctx context.Context, planId uuid.UUID, active bool) error {
	if s.client != nil {
		currentPlan, err := s.repo.GetPlanByID(ctx, pgconv.ParseUUIDToPgType(planId))
		if err != nil {
			return err
		}

		_, err = s.client.V1Products.Update(ctx, currentPlan.ExternalID, &stripe.ProductUpdateParams{
			Active: stripe.Bool(active),
		})
		if err != nil {
			return fmt.Errorf("erro ao atualizar status do produto no Stripe: %w", err)
		}
	}

	return s.repo.TogglePlanActiveStatus(ctx, db.TogglePlanActiveStatusParams{
		ID:     pgconv.ParseUUIDToPgType(planId),
		Active: pgconv.BoolToPgBool(active),
	})
}
