package service

import (
	"context"

	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	db "github.com/ProTrack-Solutions/protrack-api/internal/database/sqlc"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryInterface interface {
	CreateSubscriptionPaymentMethod(ctx context.Context, arg db.CreateSubscriptionPaymentMethodParams) (pgtype.UUID, error)
	GetSubscriptionPaymentMethodByCompanyId(ctx context.Context, companyId pgtype.UUID) ([]db.SubscriptionPaymentMethod, error)
	GetSubscriptionPaymentMethodById(ctx context.Context, paymentMethodId pgtype.UUID) (db.SubscriptionPaymentMethod, error)
	UpdateSubscriptionPaymentMethod(ctx context.Context, arg db.UpdateSubscriptionPaymentMethodParams) error
	SetDefaultSubscriptionPaymentMethod(ctx context.Context, arg db.SetDefaultSubscriptionPaymentMethodParams) error
	DeleteSubscriptionPaymentMethod(ctx context.Context, paymentMethodId pgtype.UUID) error
	WithTx(tx db.DBTX) *repository.Repository
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

func (s *Service) CreateSubscriptionPaymentMethodTx(ctx context.Context, tx db.DBTX, companyId uuid.UUID, userId uuid.UUID, req domain.CreateSubscriptionPaymentMethodRequest) (uuid.UUID, error) {
	RepoTx := s.repo.WithTx(tx)

	paymentId, err := RepoTx.CreateSubscriptionPaymentMethod(ctx, db.CreateSubscriptionPaymentMethodParams{
		CompanyID:              pgconv.ParseUUIDToPgType(companyId),
		GatewayPaymentMethodID: req.GatewayPaymentMethodId,
		Type:                   req.Type,
		CardBrand:              pgconv.ParseStringToPgType(req.CardBrand),
		CardLast4:              pgconv.ParseStringToPgType(req.CardLastFour),
		CardExpMonth:           pgconv.IntToPgInt4(int(req.CardExpMonth)),
		CardExpYear:            pgconv.IntToPgInt4(int(req.CardExpYear)),
		IsDefault:              pgconv.BoolToPgBool(req.IsDefault),
		CreatedBy:              pgconv.ParseUUIDToPgType(userId),
	})
	if err != nil {
		return uuid.Nil, err
	}

	return pgconv.PgUUIDToUUID(paymentId), nil
}

func (s *Service) CreateSubscriptionPaymentMethod(ctx context.Context, companyId uuid.UUID, userId uuid.UUID, req domain.CreateSubscriptionPaymentMethodRequest) error {
	methods, err := s.repo.GetSubscriptionPaymentMethodByCompanyId(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return err
	}

	for _, mt := range methods {
		if mt.IsDefault == pgconv.BoolToPgBool(true) {
			if err := s.repo.SetDefaultSubscriptionPaymentMethod(ctx, db.SetDefaultSubscriptionPaymentMethodParams{
				CompanyID: pgconv.ParseUUIDToPgType(companyId),
				IsDefault: pgconv.BoolToPgBool(false),
				UpdatedBy: pgconv.ParseUUIDToPgType(userId),
			}); err != nil {
				return err
			}
		}
	}

	_, err = s.repo.CreateSubscriptionPaymentMethod(ctx, db.CreateSubscriptionPaymentMethodParams{
		CompanyID:              pgconv.ParseUUIDToPgType(companyId),
		GatewayPaymentMethodID: req.GatewayPaymentMethodId,
		Type:                   req.Type,
		CardBrand:              pgconv.ParseStringToPgType(req.CardBrand),
		CardLast4:              pgconv.ParseStringToPgType(req.CardLastFour),
		CardExpMonth:           pgconv.IntToPgInt4(int(req.CardExpMonth)),
		CardExpYear:            pgconv.IntToPgInt4(int(req.CardExpYear)),
		IsDefault:              pgconv.BoolToPgBool(req.IsDefault),
		CreatedBy:              pgconv.ParseUUIDToPgType(userId),
	})

	return err
}

func (s *Service) GetSubscriptionPaymentMethodByCompanyId(ctx context.Context, companyId uuid.UUID) ([]domain.SubscriptionPaymentMethodResponse, error) {
	paymentMethods, err := s.repo.GetSubscriptionPaymentMethodByCompanyId(ctx, pgconv.ParseUUIDToPgType(companyId))
	if err != nil {
		return nil, err
	}

	var response []domain.SubscriptionPaymentMethodResponse
	for _, pm := range paymentMethods {
		response = append(response, domain.SubscriptionPaymentMethodResponse{
			ID:                     pgconv.PgUUIDToUUID(pm.CompanyID),
			CompanyID:              pgconv.PgUUIDToUUID(pm.CompanyID),
			GatewayPaymentMethodID: pm.GatewayPaymentMethodID,
			Type:                   pm.Type,
			CardBrand:              pgconv.ParsePgTextToString(pm.CardBrand),
			CardLast4:              pgconv.ParsePgTextToString(pm.CardLast4),
			CardExpMonth:           int64(pgconv.PgInt4ToInt(pm.CardExpMonth)),
			CardExpYear:            int64(pgconv.PgInt4ToInt(pm.CardExpYear)),
			IsDefault:              pgconv.PgBoolToBool(pm.IsDefault),
			CreatedAt:              pgconv.PgTimestamptzToTime(pm.CreatedAt),
			UpdatedAt:              pgconv.PgTimestamptzToTime(pm.UpdatedAt),
		})
	}

	return response, nil
}

func (s *Service) GetSubscriptionPaymentMethodById(ctx context.Context, paymentMethodId uuid.UUID) (domain.SubscriptionPaymentMethodResponse, error) {
	paymentMethod, err := s.repo.GetSubscriptionPaymentMethodById(ctx, pgconv.ParseUUIDToPgType(paymentMethodId))
	if err != nil {
		return domain.SubscriptionPaymentMethodResponse{}, err
	}

	return domain.SubscriptionPaymentMethodResponse{
		ID:                     pgconv.PgUUIDToUUID(paymentMethod.ID),
		CompanyID:              pgconv.PgUUIDToUUID(paymentMethod.CompanyID),
		GatewayPaymentMethodID: paymentMethod.GatewayPaymentMethodID,
		Type:                   paymentMethod.Type,
		CardBrand:              pgconv.ParsePgTextToString(paymentMethod.CardBrand),
		CardLast4:              pgconv.ParsePgTextToString(paymentMethod.CardLast4),
		CardExpMonth:           int64(pgconv.PgInt4ToInt(paymentMethod.CardExpMonth)),
		CardExpYear:            int64(pgconv.PgInt4ToInt(paymentMethod.CardExpYear)),
		IsDefault:              pgconv.PgBoolToBool(paymentMethod.IsDefault),
		CreatedAt:              pgconv.PgTimestamptzToTime(paymentMethod.CreatedAt),
		UpdatedAt:              pgconv.PgTimestamptzToTime(paymentMethod.UpdatedAt),
	}, nil
}

func (s *Service) UpdateSubscriptionPaymentMethod(ctx context.Context, userId uuid.UUID, paymentMethodId uuid.UUID, req domain.UpdateSubscriptionPaymentMethodRequest) error {
	currentPaymentMethod, err := s.repo.GetSubscriptionPaymentMethodById(ctx, pgconv.ParseUUIDToPgType(paymentMethodId))
	if err != nil {
		return err
	}

	arg := db.UpdateSubscriptionPaymentMethodParams{
		GatewayPaymentMethodID: currentPaymentMethod.GatewayPaymentMethodID,
		Type:                   currentPaymentMethod.Type,
		CardBrand:              currentPaymentMethod.CardBrand,
		CardLast4:              currentPaymentMethod.CardLast4,
		CardExpMonth:           currentPaymentMethod.CardExpMonth,
		CardExpYear:            currentPaymentMethod.CardExpYear,
		UpdatedBy:              pgconv.ParseUUIDToPgType(userId),
	}

	domain.ApplyUpdateSubscriptionPaymentMethodParams(req, &arg)

	if err := s.repo.UpdateSubscriptionPaymentMethod(ctx, arg); err != nil {
		return err
	}

	return nil
}

func (s *Service) SetDefaultSubscriptionPaymentMethod(ctx context.Context, companyID uuid.UUID, userId uuid.UUID, paymentMethodID uuid.UUID) error {
	return s.repo.SetDefaultSubscriptionPaymentMethod(ctx, db.SetDefaultSubscriptionPaymentMethodParams{
		CompanyID: pgconv.ParseUUIDToPgType(companyID),
		IsDefault: pgconv.BoolToPgBool(true),
		UpdatedBy: pgconv.ParseUUIDToPgType(userId),
	})
}

func (s *Service) DeleteSubscriptionPaymentMethod(ctx context.Context, paymentMethodId uuid.UUID) error {
	return s.repo.DeleteSubscriptionPaymentMethod(ctx, pgconv.ParseUUIDToPgType(paymentMethodId))
}
