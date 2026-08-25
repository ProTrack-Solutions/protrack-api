package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/domain"
	companiesDomain "github.com/ProTrack-Solutions/protrack-api/internal/companies/domain"
	companiesService "github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	companySettingsDomain "github.com/ProTrack-Solutions/protrack-api/internal/company_settings/domain"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	departmentModulesDomain "github.com/ProTrack-Solutions/protrack-api/internal/department_modules/domain"
	plansService "github.com/ProTrack-Solutions/protrack-api/internal/plans/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/shared/events"
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
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

var (
	// ErrPasswordResetRateLimited é retornado quando o e-mail/IP excedeu o
	// número de solicitações de reset permitidas na janela de tempo.
	ErrPasswordResetRateLimited = errors.New("muitas tentativas, aguarde alguns minutos")
	// ErrInvalidResetToken é retornado quando o token de reset é inválido,
	// já foi usado ou expirou.
	ErrInvalidResetToken = errors.New("link inválido ou expirado")
)

type Service struct {
	userService           *userService.Service
	companiesService      *companiesService.Service
	paymentMethodsService *paymentMethodsService.Service
	subscriptionService   *subscriptionService.Service
	plansService          *plansService.Service
	stripeService         *stripeService.Service
	companySettings       companySettingsDomain.ServiceInterface
	departmentModules     departmentModulesDomain.ServiceInterface
	jwtManager            *jwt.JWTManager
	pool                  *pgxpool.Pool

	passwordResetStore *cache.PasswordResetStore
	rateLimiter        *cache.RateLimiter
	blacklist          *cache.TokenBlacklist
	amqpChan           *amqp091.Channel
	cfg                *config.Config
}

func NewService(stripeService *stripeService.Service,
	userService *userService.Service,
	companiesService *companiesService.Service,
	paymentMethodsService *paymentMethodsService.Service,
	subscriptionService *subscriptionService.Service,
	plansService *plansService.Service,
	jwtManager *jwt.JWTManager,
	pool *pgxpool.Pool,
	passwordResetStore *cache.PasswordResetStore,
	rateLimiter *cache.RateLimiter,
	blacklist *cache.TokenBlacklist,
	amqpChan *amqp091.Channel,
	cfg *config.Config,
	companySettings companySettingsDomain.ServiceInterface,
	departmentModules departmentModulesDomain.ServiceInterface,
) *Service {
	return &Service{
		stripeService:         stripeService,
		userService:           userService,
		companiesService:      companiesService,
		paymentMethodsService: paymentMethodsService,
		subscriptionService:   subscriptionService,
		plansService:          plansService,
		jwtManager:            jwtManager,
		pool:                  pool,
		passwordResetStore:    passwordResetStore,
		rateLimiter:           rateLimiter,
		blacklist:             blacklist,
		amqpChan:              amqpChan,
		cfg:                   cfg,
		companySettings:       companySettings,
		departmentModules:     departmentModules,
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

	subscription, err := s.subscriptionService.GetSubscriptionByCompanyID(ctx, user.CompanyID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get subscription")
		return nil, err
	}

	switch subscription.Status {
	case "canceled":
		return nil, fmt.Errorf("subscription canceled")
	case "paused":
		return nil, fmt.Errorf("subscription paused")
	}

	if subscription.CurrentPeriodEnd.Before(time.Now()) {
		return nil, fmt.Errorf("subscription expired")
	}

	var hasCompany bool

	if user.CompanyID != uuid.Nil {
		hasCompany = true
	} else {
		hasCompany = false
	}

	tokenPair, err := s.jwtManager.GenerateTokenPair(user.DepartmentID, user.ID, user.CompanyID, user.Role, req.Aud)
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

func (s *Service) RefreshToken(ctx context.Context, refreshToken string, userID uuid.UUID) (*domain.LoginResponse, error) {
	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		return &domain.LoginResponse{}, err
	}

	subscription, err := s.subscriptionService.GetSubscriptionByCompanyID(ctx, user.CompanyID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get subscription")
		return nil, err
	}

	switch subscription.Status {
	case "canceled":
		return nil, fmt.Errorf("subscription canceled")
	case "paused":
		return nil, fmt.Errorf("subscription paused")
	}

	if subscription.CurrentPeriodEnd.Before(time.Now()) {
		return nil, fmt.Errorf("subscription expired")
	}

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

	// Módulos liberados para o departamento do usuário, mesma fonte usada pelo
	// middleware RequireModule. ADMIN não depende disso (bypassa no backend e
	// deve bypassar no front também), então erro/ausência de departamento aqui
	// não impede o login: o usuário só fica sem módulos extras no front.
	var moduleCodes []string
	if user.DepartmentID != uuid.Nil {
		modules, err := s.departmentModules.ListModulesByDepartment(ctx, user.DepartmentID)
		if err != nil {
			log.Warn().Err(err).Str("department_id", user.DepartmentID.String()).Msg("failed to load department modules for /me")
		} else {
			moduleCodes = make([]string, 0, len(modules))
			for _, module := range modules {
				moduleCodes = append(moduleCodes, module.Code)
			}
		}
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
		Modules:        moduleCodes,
	}, nil
}

func (s *Service) Register(ctx context.Context, req domain.RegisterRequest) (*domain.RegisterResponse, error) {
	plan, err := s.plansService.GetPlanByID(ctx, req.Payment.PlanID)
	if err != nil {
		return nil, err
	}

	periodStart := time.Now()
	var periodEnd time.Time

	switch plan.BillingCycle {
	case "monthly":
		periodEnd = periodStart.AddDate(0, 1, 0)
	case "yearly":
		periodEnd = periodStart.AddDate(1, 0, 0)
	default:
		return nil, fmt.Errorf("ciclo de cobrança inválido: %s", plan.BillingCycle)
	}

	stripe, err := s.stripeService.CreateSubscription(stripeDomain.CreateSubscriptionInput{
		Email:          req.Company.Email,
		Name:           req.Company.TradeName,
		CardToken:      req.Payment.CardToken,
		PriceID:        plan.ExternalPriceId,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
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
		ExternalCompanyId:   stripe.CustomerID,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Erro company")

		return nil, err
	}

	userId, err := s.userService.CreateUserTx(ctx, tx, companyId, userDomain.CreateUserParams{
		Name:         req.User.Name,
		Email:        req.User.Email,
		Username:     req.User.Username,
		PasswordHash: req.User.Password,
		Role:         "ADMIN",
		Status:       "ACTIVE",
	})
	if err != nil {
		log.Debug().Err(err).Msg("Erro user")

		return nil, err
	}

	paymentId, err := s.paymentMethodsService.CreateSubscriptionPaymentMethodTx(ctx, tx, companyId, userId, paymentMethodsDomain.CreateSubscriptionPaymentMethodRequest{
		GatewayPaymentMethodId: stripe.GatewayPaymentMethodId,
		Type:                   req.Payment.Type,
		CardBrand:              req.Payment.CardBrand,
		CardLastFour:           req.Payment.CardLastFour,
		CardExpMonth:           req.Payment.CardExpMonth,
		CardExpYear:            req.Payment.CardExpYear,
		IsDefault:              true,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Erro payment method")
		return nil, err
	}

	err = s.subscriptionService.CreateSubscription(ctx, tx, companyId, subscriptionDomain.CreateSubscriptionRequest{
		PlanId:                 req.Payment.PlanID,
		PaymentMethodsId:       paymentId,
		ExternalSubscriptionID: stripe.SubscriptionID,
		Status:                 stripe.Status,
		CurrentPeriodStart:     periodStart,
		CurrentPeriodEnd:       periodEnd,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Erro subscription")
		return nil, err
	}

	if err = s.companySettings.SetDefaultSettings(ctx, tx, companyId); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &domain.RegisterResponse{
		CompanyID:          companyId,
		SubscriptionStatus: stripe.Status,
		ClientSecret:       stripe.ClientSecret,
		RequiresAction:     stripe.ClientSecret != "",
	}, nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	allowed, err := s.rateLimiter.Allow(ctx, "forgot_password:"+email, 3, time.Hour)
	if err != nil {
		log.Error().Err(err).Msg("Falha ao verificar rate limit de forgot-password")
		return fmt.Errorf("checking rate limit: %w", err)
	}

	if !allowed {
		return ErrPasswordResetRateLimited
	}

	user, err := s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}

	token, err := s.passwordResetStore.GenerateToken()
	if err != nil {
		return fmt.Errorf("generating reset token: %w", err)
	}

	if err := s.passwordResetStore.Store(ctx, user.ID.String(), token); err != nil {
		return fmt.Errorf("storing reset token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.FrontendURL, token)

	event := events.PasswordResetEmail{
		UserID:   user.ID.String(),
		Name:     user.Name,
		Email:    user.Email,
		ResetURL: resetURL,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("serializing reset email event: %w", err)
	}

	if err := s.amqpChan.PublishWithContext(
		ctx,
		"protrack.ex.eventos",
		"email.password_reset",
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
		},
	); err != nil {
		log.Error().Err(err).Str("user_id", user.ID.String()).Msg("Falha ao publicar evento de reset de senha no RabbitMQ")
		return fmt.Errorf("publishing reset email event: %w", err)
	}

	return nil
}

func (s *Service) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	userIDStr, err := s.passwordResetStore.Validate(ctx, token)
	if err != nil {
		return ErrInvalidResetToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ErrInvalidResetToken
	}

	if err := s.userService.ResetPasswordHash(ctx, userID, newPassword); err != nil {
		return err
	}

	if err = s.passwordResetStore.Invalidate(ctx, token); err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Falha ao invalidar token de reset após uso")
	}

	if err = s.blacklist.ClearUserTokens(ctx, userID.String()); err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Falha ao revogar sessões ativas após reset de senha")
	}

	user, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Falha ao buscar usuário para notificação pós-reset")
		return err
	}

	event := events.PasswordChangedEmail{
		UserID: userID.String(),
		Name:   user.Name,
		Email:  user.Email,
	}

	body, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("Falha ao serializar evento de senha alterada")
		return nil
	}

	if err := s.amqpChan.PublishWithContext(
		ctx,
		"protrack.ex.eventos",
		"email.password_changed",
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
		},
	); err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Falha ao publicar evento de senha alterada no RabbitMQ")
	}
	return nil
}
