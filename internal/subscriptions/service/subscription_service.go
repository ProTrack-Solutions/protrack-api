package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateSubscription(ctx context.Context, arg db.CreateSubscriptionParams) error
	GetSubscriptionById(ctx context.Context, id pgtype.UUID) (db.Subscription, error)
	UpdateSubscriptionPlan(ctx context.Context, arg db.UpdateSubscriptionPlanParams) error
	UpdateSubscriptionMethod(ctx context.Context, arg db.UpdateSubscriptionMethodParams) error
	UpdateSubscriptionStatus(ctx context.Context, arg db.UpdateSubscriptionStatusParams) error
	CancelSubscription(ctx context.Context, id pgtype.UUID) error
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

func (s *Service) CreateSubscription(ctx context.Context, tx db.DBTX, companyId uuid.UUID, req domain.CreateSubscriptionRequest) error {
	repoTx := db.New(tx)

	return repoTx.CreateSubscription(ctx, db.CreateSubscriptionParams{
		CompanyID:              pgconv.ParseUUIDToPgType(companyId),
		PlanID:                 pgconv.ParseUUIDToPgType(req.PlanId),
		PaymentMethodID:        pgconv.ParseUUIDToPgType(req.PaymentMethodsId),
		ExternalSubscriptionID: pgconv.ParseStringToPgText(req.ExternalSubscriptionID),
		Status:                 req.Status,
		CurrentPeriodStart:     pgconv.TimeToPgTimestamptz(req.CurrentPeriodStart),
		CurrentPeriodEnd:       pgconv.TimeToPgTimestamptz(req.CurrentPeriodEnd),
	})
}

func (s *Service) GetSubscriptionById(ctx context.Context, id uuid.UUID) (domain.SubscriptionResponse, error) {
	subscription, err := s.repo.GetSubscriptionById(ctx, pgconv.ParseUUIDToPgType(id))
	if err != nil {
		return domain.SubscriptionResponse{}, err
	}

	return domain.SubscriptionResponse{
		ID:                     pgconv.PgUUIDToUUID(subscription.ID),
		CompanyID:              pgconv.PgUUIDToUUID(subscription.CompanyID),
		PlanID:                 pgconv.PgUUIDToUUID(subscription.PlanID),
		PaymentMethodID:        pgconv.PgUUIDToUUID(subscription.PaymentMethodID),
		ExternalSubscriptionID: pgconv.ParsePgTextToString(subscription.ExternalSubscriptionID),
		Status:                 subscription.Status,
		CurrentPeriodStart:     pgconv.PgTimestamptzToTime(subscription.CurrentPeriodStart),
		CurrentPeriodEnd:       pgconv.PgTimestamptzToTime(subscription.CurrentPeriodEnd),
		CanceledAt:             pgconv.PgTimestamptzToTime(subscription.CanceledAt),
		CreatedAt:              pgconv.PgTimestamptzToTime(subscription.CreatedAt),
		UpdatedAt:              pgconv.PgTimestamptzToTime(subscription.UpdatedAt),
	}, nil
}

func (s *Service) UpdateSubscriptionPlan(ctx context.Context, id uuid.UUID, req domain.UpdateSubscriptionPlanRequest) error {
	return s.repo.UpdateSubscriptionPlan(ctx, db.UpdateSubscriptionPlanParams{
		ID:     pgconv.ParseUUIDToPgType(id),
		PlanID: pgconv.ParseUUIDToPgType(req.PlanId),
	})
}

func (s *Service) UpdateSubscriptionMethod(ctx context.Context, id uuid.UUID, req domain.UpdateSubscriptionMethodRequest) error {
	return s.repo.UpdateSubscriptionMethod(ctx, db.UpdateSubscriptionMethodParams{
		ID:              pgconv.ParseUUIDToPgType(id),
		PaymentMethodID: pgconv.ParseUUIDToPgType(req.PaymentMethodId),
	})
}

func (s *Service) UpdateSubscriptionStatus(ctx context.Context, id uuid.UUID, req domain.UpdateSubscriptionStatusRequest) error {
	return s.repo.UpdateSubscriptionStatus(ctx, db.UpdateSubscriptionStatusParams{
		ID:     pgconv.ParseUUIDToPgType(id),
		Status: req.Status,
	})
}

func (s *Service) CancelSubscription(ctx context.Context, id uuid.UUID) error {
	return s.repo.CancelSubscription(ctx, pgconv.ParseUUIDToPgType(id))
}
