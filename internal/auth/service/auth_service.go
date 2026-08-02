package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/domain"
	companiesDomain "github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	companiesService "github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	plansService "github.com/ProTrack-Solutions/protrack-api/internal/plans/service"
	stripeDomain "github.com/ProTrack-Solutions/protrack-api/internal/stripe/domain"
	stripeService "github.com/ProTrack-Solutions/protrack-api/internal/stripe/service"
	paymentMethodsDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/domain"
	paymentMethodsService "github.com/ProTrack-Solutions/protrack-api/internal/subscription_payment_methods/service"
	subscriptionDomain "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/domain"
	subscriptionService "github.com/ProTrack-Solutions/protrack-api/internal/subscriptions/service"
	userDomain "github.com/ProTrack-Solutions/protrack-api/internal/users/domain"
	userService "github.com/ProTrack-Solutions/protrack-api/internal/users/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type Service struct {
	userService           *userService.Service
	companiesService      *companiesService.Service
	paymentMethodsService *paymentMethodsService.Service
	subscriptionService   *subscriptionService.Service
	plansService          *plansService.Service
	stripeService         *stripeService.Service
	jwtManager            *jwt.JWTManager
	pool                  *pgxpool.Pool
}

func NewService(stripeService *stripeService.Service, userService *userService.Service, companiesService *companiesService.Service, paymentMethodsService *paymentMethodsService.Service, subscriptionService *subscriptionService.Service, plansService *plansService.Service, jwtManager *jwt.JWTManager, pool *pgxpool.Pool) *Service {
	return &Service{
		stripeService:         stripeService,
		userService:           userService,
		companiesService:      companiesService,
		paymentMethodsService: paymentMethodsService,
		subscriptionService:   subscriptionService,
		plansService:          plansService,
		jwtManager:            jwtManager,
		pool:                  pool,
	}
}

func (s *Service) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	if req.Aud == "" {
		return &domain.LoginResponse{}, errors.New("invalid aud")
	}

	user, err := s.userService.ValidatePassword(ctx, req.Email, req.Password)
	if err != nil {
		return &domain.LoginResponse{}, err
	}

	var hasCompany bool

	if user.CompanyID != uuid.Nil {
		hasCompany = true
	} else {
		hasCompany = false
	}

	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.CompanyID, user.Role, req.Aud)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate tokens")
		return nil, err
	}

	return &domain.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		HasCompany:   hasCompany,
		ExpiresIn:    tokenPair.ExpireIn,
		TokenType:    "Bearer",
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResponse, error) {
	tokenPair, err := s.jwtManager.RefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpireIn,
		TokenType:    "Bearer",
	}, nil
}

/* func (s *Service) Logout(ctx context.Context, token string) error {
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		return err
	}

	expiresIn := time.Until(claims.ExpiresAt.Time)
	if expiresIn <= 0 {
		return nil
	}

	return nil
} */

func (s *Service) GetUserFromContext(ctx context.Context, id uuid.UUID) (userDomain.UserResponse, error) {
	user, err := s.userService.GetUserByID(ctx, id)
	if err != nil {
		return userDomain.UserResponse{}, err
	}

	return userDomain.UserResponse{
		ID:             user.ID,
		Name:           user.Name,
		Email:          user.Email,
		Username:       user.Username,
		Role:           user.Role,
		Status:         user.Status,
		CompanyID:      user.CompanyID,
		DepartmentID:   user.DepartmentID,
		LastLoginAt:    user.LastLoginAt,
		CreatedBy:      user.CreatedBy,
		UpdatedBy:      user.UpdatedBy,
		DeletedBy:      user.DeletedBy,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
		DeletedAt:      user.DeletedAt,
		DepartmentName: user.DepartmentName,
	}, nil
}

func (s *Service) Register(ctx context.Context, req domain.RegisterRequest) error {
	plan, err := s.plansService.GetPlanByID(ctx, req.Payment.PlanID)
	if err != nil {
		return err
	}

	periodStart := time.Now()
	var periodEnd time.Time

	switch plan.BillingCycle {
	case "monthly":
		periodEnd = periodStart.AddDate(0, 1, 0)
	case "yearly":
		periodEnd = periodStart.AddDate(1, 0, 0)
	default:
		return fmt.Errorf("ciclo de cobrança inválido: %s", plan.BillingCycle)
	}

	stripe, err := s.stripeService.CreateSubscription(stripeDomain.CreateSubscriptionInput{
		Email:          req.Company.Email,
		Name:           req.Company.TradeName,
		CardToken:      req.Payment.CardToken,
		PriceID:        plan.ExternalPriceId,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	companyId, err := s.companiesService.CreateCompanyTx(ctx, tx, companiesDomain.CreateCompanyParams{
		Name:                req.Company.Name,
		TradeName:           req.Company.TradeName,
		Document:            req.Company.Document,
		Email:               req.Company.Email,
		Phone:               req.Company.Phone,
		Website:             req.Company.Website,
		AddressStreet:       req.Company.AddressStreet,
		AddressNumber:       req.Company.AddressNumber,
		AddressComplement:   req.Company.AddressComplement,
		AddressNeighborhood: req.Company.AddressNeighborhood,
		AddressCity:         req.Company.AddressCity,
		AddressState:        req.Company.AddressState,
		AddressZipcode:      req.Company.AddressZipcode,
		AddressCountry:      req.Company.AddressCountry,
		Timezone:            req.Company.Timezone,
		Status:              "ACTIVE",
		CreatedBy:           uuid.Nil,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Erro company")

		return err
	}

	userId, err := s.userService.CreateUserTx(ctx, tx, userDomain.CreateUserParams{
		Name:         req.User.Name,
		Email:        req.User.Email,
		Username:     req.User.Username,
		PasswordHash: req.User.Password,
		Role:         "ADMIN",
		Status:       "ACTIVE",
		CompanyID:    companyId,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Erro user")

		return err
	}

	paymentId, err := s.paymentMethodsService.CreateSubscriptionPaymentMethodTx(ctx, tx, companyId, userId, paymentMethodsDomain.CreateSubscriptionPaymentMethodRequest{
		GatewayPaymentMethodId: stripe.CustomerID,
		Type:                   req.Payment.Type,
		CardBrand:              req.Payment.CardBrand,
		CardLastFour:           req.Payment.CardLastFour,
		CardExpMonth:           req.Payment.CardExpMonth,
		CardExpYear:            req.Payment.CardExpYear,
		IsDefault:              true,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Erro payment method")
		return err
	}

	err = s.subscriptionService.CreateSubscription(ctx, tx, companyId, subscriptionDomain.CreateSubscriptionRequest{
		PlanId:                 req.Payment.PlanID,
		PaymentMethodsId:       paymentId,
		ExternalSubscriptionID: stripe.SubscriptionID,

		Status:             stripe.Status,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Erro subscription")
		return err
	}

	return tx.Commit(ctx)
}
