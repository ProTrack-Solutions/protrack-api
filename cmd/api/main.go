package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	accountsReceivableHandler "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/handler"
	accountsReceivableRepository "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/repository"
	accountsReceivableService "github.com/ProTrack-Solutions/protrack-api/internal/accounts_receivable/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/adapters/cache"
	redis_connection "github.com/ProTrack-Solutions/protrack-api/internal/adapters/redis"
	analyticsService "github.com/ProTrack-Solutions/protrack-api/internal/analytics/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/auth/adapters/jwt"
	authHandler "github.com/ProTrack-Solutions/protrack-api/internal/auth/handler"
	authService "github.com/ProTrack-Solutions/protrack-api/internal/auth/service"
	billCategoriesHandler "github.com/ProTrack-Solutions/protrack-api/internal/bill_categories/handler"
	billCategoriesRepository "github.com/ProTrack-Solutions/protrack-api/internal/bill_categories/repository"
	billCategoriesService "github.com/ProTrack-Solutions/protrack-api/internal/bill_categories/service"
	billsPayableHandler "github.com/ProTrack-Solutions/protrack-api/internal/bills_payable/handler"
	billsPayableRepository "github.com/ProTrack-Solutions/protrack-api/internal/bills_payable/repository"
	billsPayableService "github.com/ProTrack-Solutions/protrack-api/internal/bills_payable/service"
	cashFlowHandler "github.com/ProTrack-Solutions/protrack-api/internal/cash_flow/handler"
	cashFlowRepository "github.com/ProTrack-Solutions/protrack-api/internal/cash_flow/repository"
	cashFlowService "github.com/ProTrack-Solutions/protrack-api/internal/cash_flow/service"
	companiesHandler "github.com/ProTrack-Solutions/protrack-api/internal/companies/handler"
	companiesRepository "github.com/ProTrack-Solutions/protrack-api/internal/companies/repository"
	companiesService "github.com/ProTrack-Solutions/protrack-api/internal/companies/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/consumers"
	customersHandler "github.com/ProTrack-Solutions/protrack-api/internal/customers/handler"
	customersRepository "github.com/ProTrack-Solutions/protrack-api/internal/customers/repository"
	customersService "github.com/ProTrack-Solutions/protrack-api/internal/customers/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/database"
	departmentsHandler "github.com/ProTrack-Solutions/protrack-api/internal/departments/handler"
	departmentsRepository "github.com/ProTrack-Solutions/protrack-api/internal/departments/repository"
	departmentsService "github.com/ProTrack-Solutions/protrack-api/internal/departments/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger"
	paymentHistoryHandler "github.com/ProTrack-Solutions/protrack-api/internal/payment_history/handler"
	paymentHistoryRepository "github.com/ProTrack-Solutions/protrack-api/internal/payment_history/repository"
	paymentHistoryService "github.com/ProTrack-Solutions/protrack-api/internal/payment_history/service"
	paymentMethodsHandler "github.com/ProTrack-Solutions/protrack-api/internal/payment_methods/handler"
	paymentMethodsRepository "github.com/ProTrack-Solutions/protrack-api/internal/payment_methods/repository"
	paymentMethodsService "github.com/ProTrack-Solutions/protrack-api/internal/payment_methods/service"
	paymentsHandler "github.com/ProTrack-Solutions/protrack-api/internal/payments/handler"
	paymentsService "github.com/ProTrack-Solutions/protrack-api/internal/payments/service"
	productsHandler "github.com/ProTrack-Solutions/protrack-api/internal/products/handler"
	productsRepository "github.com/ProTrack-Solutions/protrack-api/internal/products/repository"
	productsService "github.com/ProTrack-Solutions/protrack-api/internal/products/service"
	productsCategoriesHandler "github.com/ProTrack-Solutions/protrack-api/internal/products_categories/handler"
	productsCategoriesRepository "github.com/ProTrack-Solutions/protrack-api/internal/products_categories/repository"
	productsCategoriesService "github.com/ProTrack-Solutions/protrack-api/internal/products_categories/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/rabbitmq"
	reportsHandler "github.com/ProTrack-Solutions/protrack-api/internal/reports/handler"
	reportsService "github.com/ProTrack-Solutions/protrack-api/internal/reports/service"
	saleItemsHandler "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/handler"
	saleItemsRepository "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/repository"
	saleItemsService "github.com/ProTrack-Solutions/protrack-api/internal/sale_items/service"
	salesHandler "github.com/ProTrack-Solutions/protrack-api/internal/sales/handler"
	salesRepository "github.com/ProTrack-Solutions/protrack-api/internal/sales/repository"
	salesService "github.com/ProTrack-Solutions/protrack-api/internal/sales/service"
	usersHandler "github.com/ProTrack-Solutions/protrack-api/internal/users/handler"
	usersRepository "github.com/ProTrack-Solutions/protrack-api/internal/users/repository"
	usersService "github.com/ProTrack-Solutions/protrack-api/internal/users/service"
	vendorsHandler "github.com/ProTrack-Solutions/protrack-api/internal/vendors/handler"
	vendorsRepository "github.com/ProTrack-Solutions/protrack-api/internal/vendors/repository"
	vendorsService "github.com/ProTrack-Solutions/protrack-api/internal/vendors/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/whatsapp"
	whatsappHandler "github.com/ProTrack-Solutions/protrack-api/internal/whatsapp/handler"
	whatsappService "github.com/ProTrack-Solutions/protrack-api/internal/whatsapp/service"
	"github.com/ProTrack-Solutions/protrack-api/internal/worker"
	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	_ "github.com/ProTrack-Solutions/protrack-api/docs"
	annoucementsHandler "github.com/ProTrack-Solutions/protrack-api/internal/annoucements/handler"
	annountmentsRepository "github.com/ProTrack-Solutions/protrack-api/internal/annoucements/repository"
	annountmentsService "github.com/ProTrack-Solutions/protrack-api/internal/annoucements/service"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           ProTrack API
// @version         1.0
// @description     API de gerenciamento financeiro e comercial ProTrack Solutions.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Suporte ProTrack
// @contact.email  suporte@protrack.com

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
		},
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Authorization", "Accept", "Page", "PerPage",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           0,
	}))

	r.HandleMethodNotAllowed = true
	r.Use(func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	logger.InitLogger("development")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading settings")
	}

	db, err := database.NewConnect(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	redis, err := redis_connection.NewRedisConnection(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to redis")
	}
	defer redis.Close()

	_, ch, err := rabbitmq.InitializeRabbitMQ(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to opem channel to rabbitmq")
	}
	defer ch.Close()

	whatsapp := whatsapp.NewWhatsapp(cfg)

	jwtManager := jwt.NewJWTManager(cfg.SecretKey)

	blacklist := cache.NewTokenBlackList(redis)

	usersRepository := usersRepository.NewRepository(db.Pool)
	companiesRepository := companiesRepository.NewRepository(db.Pool)
	departmentsRepository := departmentsRepository.NewRepository(db.Pool)
	productsCategoriesRepository := productsCategoriesRepository.NewRepository(db.Pool)
	productsRepository := productsRepository.NewRepository(db.Pool)
	customersRepository := customersRepository.NewRepository(db.Pool)
	salesRepository := salesRepository.NewRepository(db.Pool)
	saleItemsRepository := saleItemsRepository.NewRepository(db.Pool)
	paymentMethodsRepository := paymentMethodsRepository.NewRepository(db.Pool)
	vendorsRepository := vendorsRepository.NewRepository(db.Pool)
	billCategoriesRepository := billCategoriesRepository.NewRepository(db.Pool)
	billsPayableRepository := billsPayableRepository.NewRepository(db.Pool)
	paymentHistoryRepository := paymentHistoryRepository.NewRepository(db.Pool)
	accountsReceivableRepository := accountsReceivableRepository.NewRepository(db.Pool)
	cashFlowRepository := cashFlowRepository.NewRepository(db.Pool)
	annountmentsRepository := annountmentsRepository.NewRepository(db.Pool)

	cashFlowService := cashFlowService.NewService(cashFlowRepository, db.Pool)
	usersService := usersService.NewService(usersRepository, db.Pool, cfg)
	companiesService := companiesService.NewService(db.Pool, companiesRepository, usersRepository)
	departmentsService := departmentsService.NewService(departmentsRepository)
	productsCategoriesService := productsCategoriesService.NewService(productsCategoriesRepository)
	productsService := productsService.NewService(productsRepository, db.Pool)
	authService := authService.NewService(usersService, jwtManager)
	customersService := customersService.NewService(customersRepository, db.Pool)
	saleItemsService := saleItemsService.NewService(saleItemsRepository, db.Pool, productsRepository)
	accountsReceivableService := accountsReceivableService.NewService(accountsReceivableRepository, db.Pool)
	salesService := salesService.NewService(salesRepository, db.Pool, saleItemsService, customersService, accountsReceivableService, productsService, productsCategoriesService, companiesService, whatsapp)
	paymentMethodsService := paymentMethodsService.NewService(paymentMethodsRepository, db.Pool)
	vendorsService := vendorsService.NewService(vendorsRepository, db.Pool)
	billCategoriesService := billCategoriesService.NewService(billCategoriesRepository, db.Pool)
	billsPayableService := billsPayableService.NewService(billsPayableRepository, db.Pool)
	paymentHistoryService := paymentHistoryService.NewService(paymentHistoryRepository, db.Pool)
	paymentsService := paymentsService.NewService(db.Pool, paymentHistoryService, accountsReceivableService, customersService, salesService)
	analyticsService := analyticsService.NewService(productsService, saleItemsService)
	reportsService := reportsService.NewService(salesService, analyticsService, paymentHistoryService, productsService)
	whatsappService := whatsappService.NewService(cfg, companiesService)
	annountmentsService := annountmentsService.NewService(annountmentsRepository, db.Pool)

	cashFlowHandler := cashFlowHandler.NewHandler(cashFlowService, jwtManager, blacklist)
	usersHandler := usersHandler.NewHandler(usersService, jwtManager, blacklist)
	companiesHandler := companiesHandler.NewHandler(companiesService, jwtManager, blacklist)
	departmentsHandler := departmentsHandler.NewHandler(departmentsService, jwtManager, blacklist)
	productsCategoriesHandler := productsCategoriesHandler.NewHandler(productsCategoriesService, jwtManager, blacklist)
	productsHandler := productsHandler.NewHandler(productsService, jwtManager, blacklist)
	authHandler := authHandler.NewHandler(authService, jwtManager, blacklist)
	customersHandler := customersHandler.NewHandler(customersService, jwtManager, blacklist)
	salesHandler := salesHandler.NewHandler(salesService, jwtManager, blacklist)
	saleItemsHandler := saleItemsHandler.NewHandler(saleItemsService, jwtManager, blacklist)
	paymentMethodsHandler := paymentMethodsHandler.NewHandler(paymentMethodsService, jwtManager, blacklist)
	vendorsHandler := vendorsHandler.NewHandler(vendorsService, jwtManager, blacklist)
	billCategoriesHandler := billCategoriesHandler.NewHandler(billCategoriesService, jwtManager, blacklist)
	billsPayableHandler := billsPayableHandler.NewHandler(billsPayableService, jwtManager, blacklist)
	paymentHistoryHandler := paymentHistoryHandler.NewHandler(paymentHistoryService, jwtManager, blacklist)
	accountsReceivableHandler := accountsReceivableHandler.NewHandler(accountsReceivableService, jwtManager, blacklist)
	paymentsHandler := paymentsHandler.NewHandler(paymentsService, jwtManager, blacklist)
	reportsHandler := reportsHandler.NewHandler(reportsService, jwtManager, blacklist)
	whatsappHandler := whatsappHandler.NewHandler(whatsappService, jwtManager, blacklist)
	annoucementsHandler := annoucementsHandler.NewHandler(annountmentsService, jwtManager, blacklist)

	api := r.Group("/api/v1")
	usersHandler.RegisterRoutes(api)
	companiesHandler.RegisterRoutes(api)
	departmentsHandler.RegisterRoutes(api)
	productsCategoriesHandler.RegisterRoutes(api)
	productsHandler.RegisterRoute(api)
	authHandler.RegisterRoute(api)
	customersHandler.RegisterRoute(api)
	salesHandler.RegisterRoutes(api)
	saleItemsHandler.RegisterRoute(api)
	paymentMethodsHandler.RegisterRoutes(api)
	vendorsHandler.RegisterRoute(api)
	billCategoriesHandler.RegisterRoute(api)
	billsPayableHandler.RegisterRoute(api)
	paymentHistoryHandler.RegisterRoute(api)
	accountsReceivableHandler.RegisterRoute(api)
	paymentsHandler.RegisterRoute(api)
	reportsHandler.RegisterRoutes(api)
	cashFlowHandler.RegisterRoute(api)
	whatsappHandler.RegisterRoute(api)
	annoucementsHandler.RegisterRoutes(api)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	worker.StartOverdueMonitor(salesService, ch)
	worker.StartBillPayableOverdueMonitor(billsPayableService)
	consumers.StartWhatsAppConsumer(ch, whatsapp)
	consumers.StartAnnouncementsConsumer(ch, annountmentsService)

	srv := &http.Server{
		Addr:    ":" + cfg.ApiPort,
		Handler: r,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Msgf("GoFinance running at the port %s", cfg.ApiPort)
		log.Info().Msgf("Data Base: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

		// ListenAndServe bloqueia até que o servidor seja fechado
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	<-sigChan
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}
