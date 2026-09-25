package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	authhttp "pekan/backend/internal/modules/core/auth/delivery/http"
	authinfra "pekan/backend/internal/modules/core/auth/infra"
	authusecase "pekan/backend/internal/modules/core/auth/usecase"
	adminhttp "pekan/backend/internal/modules/core/admin/delivery/http"
	admininfra "pekan/backend/internal/modules/core/admin/infra"
	adminusecase "pekan/backend/internal/modules/core/admin/usecase"
	entitlementhttp "pekan/backend/internal/modules/core/subscription/delivery/http"
	entitlementinfra "pekan/backend/internal/modules/core/subscription/infra"
	entitlementusecase "pekan/backend/internal/modules/core/subscription/usecase"
	financeattachmenthttp "pekan/backend/internal/modules/finance/attachments/delivery/http"
	financeattachmentinfra "pekan/backend/internal/modules/finance/attachments/infra"
	financeattachmentusecase "pekan/backend/internal/modules/finance/attachments/usecase"
	financebudgethttp "pekan/backend/internal/modules/finance/budgets/delivery/http"
	financebudgetinfra "pekan/backend/internal/modules/finance/budgets/infra"
	financebudgetusecase "pekan/backend/internal/modules/finance/budgets/usecase"
	financedashboardhttp "pekan/backend/internal/modules/finance/dashboard/delivery/http"
	financedashboardinfra "pekan/backend/internal/modules/finance/dashboard/infra"
	financedashboardusecase "pekan/backend/internal/modules/finance/dashboard/usecase"
	financemasterhttp "pekan/backend/internal/modules/finance/master/delivery/http"
	financemasterinfra "pekan/backend/internal/modules/finance/master/infra"
	financemasterusecase "pekan/backend/internal/modules/finance/master/usecase"
	financenotificationhttp "pekan/backend/internal/modules/finance/notifications/delivery/http"
	financenotificationinfra "pekan/backend/internal/modules/finance/notifications/infra"
	financenotificationusecase "pekan/backend/internal/modules/finance/notifications/usecase"
	financereceipthttp "pekan/backend/internal/modules/finance/receipts/delivery/http"
	financereceiptinfra "pekan/backend/internal/modules/finance/receipts/infra"
	financereceiptusecase "pekan/backend/internal/modules/finance/receipts/usecase"
	financereminderhttp "pekan/backend/internal/modules/finance/reminders/delivery/http"
	financereminderinfra "pekan/backend/internal/modules/finance/reminders/infra"
	financereminderusecase "pekan/backend/internal/modules/finance/reminders/usecase"
	financereporthttp "pekan/backend/internal/modules/finance/reports/delivery/http"
	financereportinfra "pekan/backend/internal/modules/finance/reports/infra"
	financereportusecase "pekan/backend/internal/modules/finance/reports/usecase"
	financesavingshttp "pekan/backend/internal/modules/finance/savings/delivery/http"
	financesavingsinfra "pekan/backend/internal/modules/finance/savings/infra"
	financesavingsusecase "pekan/backend/internal/modules/finance/savings/usecase"
	financesettingshttp "pekan/backend/internal/modules/finance/settings/delivery/http"
	financesettingsinfra "pekan/backend/internal/modules/finance/settings/infra"
	financesettingsusecase "pekan/backend/internal/modules/finance/settings/usecase"
	financewhatsapphttp "pekan/backend/internal/modules/finance/whatsapp/delivery/http"
	financewhatsappinfra "pekan/backend/internal/modules/finance/whatsapp/infra"
	financewhatsappusecase "pekan/backend/internal/modules/finance/whatsapp/usecase"
	transactionhttp "pekan/backend/internal/modules/finance/transactions/delivery/http"
	"pekan/backend/internal/modules/finance/transactions/infra"
	"pekan/backend/internal/modules/finance/transactions/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/audit"
	"pekan/backend/internal/platform/auth"
	"pekan/backend/internal/platform/config"
	"pekan/backend/internal/platform/db"
	"pekan/backend/internal/platform/health"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/session"
	"pekan/backend/internal/platform/storage"
	"pekan/backend/internal/platform/security"
)

type Server struct {
	httpServer *http.Server
	db         *sql.DB
	adminUC    *adminusecase.Service
}


func NewServer(cfg config.Config) (*Server, error) {
	conn, err := db.NewPostgres(cfg.DatabaseURL, db.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
	})
	if err != nil {
		return nil, err
	}

	// Update 'pekanhonet' to 'pekan' in database automatically on startup for seamless migration
	if _, err := conn.Exec("UPDATE public.tenants SET code = 'pekan' WHERE code = 'pekanhonet'"); err != nil {
		log.Printf("[Startup] Warning: failed to auto-migrate tenant code from pekanhonet to pekan: %v", err)
	}

	storageProvider := storage.NewProvider(cfg)
	rateLimitStore := middleware.RateLimitStore(middleware.NewMemoryRateLimitStore())
	var rdb *redis.Client
	var sessionStore session.Store
	if cfg.RateLimitRedisURL != "" {
		redisStore, err := middleware.NewRedisRateLimitStore(cfg.RateLimitRedisURL, cfg.RateLimitRedisPrefix, rateLimitStore)
		if err != nil {
			return nil, err
		}
		rateLimitStore = redisStore
		rdb = redisStore.Client()
		sessionStore = session.NewRedisStore(rdb)
	}

	authSvc := auth.NewService(cfg.JWTIssuer, cfg.JWTSecret, cfg.AccessTokenTTL)
	authorizer := access.NewAuthorizer()
	auditLogger := audit.NewDBLogger(conn)

	// Initialize cipher for data-at-rest encryption (use JWTSecret as the base key)
	cipher, err := security.NewCipher(cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	authRepo := authinfra.NewRepositoryPG(conn, cipher)
	authUC := authusecase.NewService(authRepo, authSvc, cfg.RefreshTokenTTL, auditLogger)
	loginLimiter := middleware.NewIPRateLimiterWithStore(authhttp.DefaultLoginRateLimitPerMinute, time.Minute, rateLimitStore, auditLogger)
	refreshLimiter := middleware.NewIPRateLimiterWithStore(authhttp.DefaultRefreshRateLimitPerMinute, time.Minute, rateLimitStore, auditLogger)
	authHandler := authhttp.NewHandler(
		authUC,
		authhttp.WithRateLimiters(loginLimiter, refreshLimiter),
		authhttp.WithAuditLogger(auditLogger),
	)
	entitlementRepo := entitlementinfra.NewRepositoryPG(conn)
	entitlementUC := entitlementusecase.NewService(entitlementRepo, authorizer, auditLogger)
	entitlementHandler := entitlementhttp.NewHandler(entitlementUC)

	adminRepo := admininfra.NewRepositoryPG(conn)

	// Load Optimization Config from Database if available to override Environment Variables
	ctx := context.Background()
	if optRaw, _, err := adminRepo.GetGlobalSetting(ctx, "optimization_config"); err == nil && optRaw != "" {
		var optMap map[string]string
		if json.Unmarshal([]byte(optRaw), &optMap) == nil {
			if limitStr, ok := optMap["api_rate_limit"]; ok && limitStr != "" {
				if limitVal, err := strconv.Atoi(limitStr); err == nil && limitVal >= 0 {
					cfg.APIRateLimitPerMinute = limitVal
				}
			}
		}
	}

	// Load Storage Configuration from Database if available to override Environment Variables
	if activeProv, _, err := adminRepo.GetGlobalSetting(ctx, "storage_active_provider"); err == nil && activeProv != "" {
		cfg.StorageProvider = activeProv
		if activeProv == "s3" {
			if s3Raw, _, err := adminRepo.GetGlobalSetting(ctx, "storage_s3_config"); err == nil && s3Raw != "" {
				var s3Map map[string]string
				if json.Unmarshal([]byte(s3Raw), &s3Map) == nil {
					cfg.StorageS3Region = s3Map["region"]
					cfg.StorageS3Bucket = s3Map["bucket"]
					cfg.StorageS3AccessKey = s3Map["accessKey"]
					cfg.StorageS3SecretKey = s3Map["secretKey"]
					cfg.StorageS3Endpoint = s3Map["endpoint"]
				}
			}
		} else if activeProv == "gdrive" {
			if gdRaw, _, err := adminRepo.GetGlobalSetting(ctx, "storage_gdrive_config"); err == nil && gdRaw != "" {
				var gdMap map[string]string
				if json.Unmarshal([]byte(gdRaw), &gdMap) == nil {
					cfg.StorageDriveFolder = gdMap["folderId"]
					cfg.StorageGDriveCredentials = gdMap["credentialsJson"]
				}
			}
		} else if activeProv == "local" {
			if locRaw, _, err := adminRepo.GetGlobalSetting(ctx, "storage_local_config"); err == nil && locRaw != "" {
				var locMap map[string]string
				if json.Unmarshal([]byte(locRaw), &locMap) == nil {
					cfg.StorageLocalPath = locMap["path"]
				}
			}
		}
	}

	adminUC := adminusecase.NewService(adminRepo, auditLogger, cfg.ReceiptScanSecret, storageProvider, rdb, conn)
	adminHandler := adminhttp.NewHandler(adminUC, authSvc, cfg.AdminSecret, loginLimiter)

	// Wire admin service into auth service for OTP delivery, tenant bootstrapping, and session store
	authUC.WithDependencies(adminUC, adminUC, nil, sessionStore)

	financeMasterRepo := financemasterinfra.NewRepositoryPG(conn)
	financeMasterUC := financemasterusecase.NewService(financeMasterRepo, authorizer, auditLogger)
	financeMasterHandler := financemasterhttp.NewHandler(financeMasterUC)

	receiptRepo := financereceiptinfra.NewRepositoryPG(conn)

	transactionRepo := infra.NewRepositoryPG(conn)
	transactionUC := usecase.NewService(transactionRepo, authorizer, auditLogger)
	transactionAttachmentUC := usecase.NewAttachmentService(transactionRepo, receiptRepo, authorizer, auditLogger, storageProvider)
	transactionHandler := transactionhttp.NewHandler(transactionUC, transactionAttachmentUC)

	financeAttachmentRepo := financeattachmentinfra.NewRepositoryPG(conn)
	financeAttachmentUC := financeattachmentusecase.NewService(financeAttachmentRepo, authorizer, auditLogger, storageProvider)
	financeAttachmentHandler := financeattachmenthttp.NewHandler(financeAttachmentUC)

	savingsRepo := financesavingsinfra.NewRepositoryPG(conn)
	savingsUC := financesavingsusecase.NewService(savingsRepo, authorizer, auditLogger)
	savingsHandler := financesavingshttp.NewHandler(savingsUC, transactionRepo)

	budgetRepo := financebudgetinfra.NewRepositoryPG(conn)
	budgetUC := financebudgetusecase.NewService(budgetRepo, authorizer, auditLogger)
	budgetHandler := financebudgethttp.NewHandler(budgetUC)

	transactionUC.WithBudgetChecker(budgetUC)

	reminderRepo := financereminderinfra.NewRepositoryPG(conn)
	reminderUC := financereminderusecase.NewService(reminderRepo, authorizer, auditLogger, storageProvider)
	reminderHandler := financereminderhttp.NewHandler(reminderUC)

	notificationRepo := financenotificationinfra.NewRepositoryPG(conn)
	notificationUC := financenotificationusecase.NewService(notificationRepo, authorizer, auditLogger)
	notificationHandler := financenotificationhttp.NewHandler(notificationUC)

	settingsRepo := financesettingsinfra.NewRepositoryPG(conn)
	settingsUC := financesettingsusecase.NewService(settingsRepo, authorizer, auditLogger)
	settingsHandler := financesettingshttp.NewHandler(settingsUC)

	receiptUC := financereceiptusecase.NewService(receiptRepo, authorizer, auditLogger, cfg.ReceiptScanSecret, append([]string{cfg.JWTSecret}, cfg.ReceiptScanLegacySecrets...), storageProvider)
	receiptHandler := financereceipthttp.NewHandler(receiptUC)

	reportRepo := financereportinfra.NewRepositoryPG(conn)
	reportExportRepo := financereportinfra.NewTransactionExportPG(conn)
	reportUC := financereportusecase.NewService(reportRepo, reportExportRepo, authorizer, auditLogger, storageProvider)
	reportHandler := financereporthttp.NewHandler(reportUC)

	dashboardRepo := financedashboardinfra.NewRepositoryPG(conn)
	dashboardUC := financedashboardusecase.NewService(dashboardRepo, authorizer, rdb)
	dashboardHandler := financedashboardhttp.NewHandler(dashboardUC)

	whatsappRepo := financewhatsappinfra.NewRepositoryPG(conn, cipher)
	whatsappUC := financewhatsappusecase.NewService(whatsappRepo, authorizer, adminUC)
	whatsappHandler := financewhatsapphttp.NewHandler(whatsappUC)
	whatsappWebhookHandler := financewhatsapphttp.NewWebhookHandler(whatsappUC)

	logger := middleware.NewLogger()

	healthChecker := health.NewChecker(conn, rdb)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.StructuredLogger(logger))
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	router.Use(middleware.RequestBodyLimit(cfg.RequestBodyMaxBytes))
	router.Use(middleware.Recovery)
	router.Use(chimw.StripSlashes)

	router.Route("/api/v1", func(r chi.Router) {
		if cfg.APIRequestTimeout > 0 {
			r.Use(chimw.Timeout(cfg.APIRequestTimeout))
		}

		if cfg.APIRateLimitPerMinute > 0 {
			r.Use(middleware.NewIPRateLimiterWithStore(
				cfg.APIRateLimitPerMinute,
				time.Duration(cfg.APIRateLimitWindowSeconds)*time.Second,
				rateLimitStore,
				auditLogger,
			))
		}

		r.Get("/healthz", healthChecker.Handle)
		r.Get("/livez", healthChecker.HandleLive)
		
		// Public Routes (with specific rate limits)
		r.Group(func(r chi.Router) {
			authHandler.RegisterPublicRoutes(r)
			adminHandler.RegisterRoutes(r)
			r.Get("/branding", adminHandler.GetPublicBranding)
		})
		
		r.Post("/webhook/whatsapp", whatsappWebhookHandler.HandleIncomingMessage)

		// Protected Routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authSvc, sessionStore))
			r.Use(middleware.Tenant)
			r.Use(middleware.AuditContext)
			
			authHandler.RegisterProtectedRoutes(r)
			entitlementHandler.RegisterRoutes(r)
			financeMasterHandler.RegisterRoutes(r)
			transactionHandler.RegisterRoutes(r)
			financeAttachmentHandler.RegisterRoutes(r)
			savingsHandler.RegisterRoutes(r)
			budgetHandler.RegisterRoutes(r)
			reminderHandler.RegisterRoutes(r)
			notificationHandler.RegisterRoutes(r)
			settingsHandler.RegisterRoutes(r)
			receiptHandler.RegisterRoutes(r)
			reportHandler.RegisterRoutes(r)
			dashboardHandler.RegisterRoutes(r)
			whatsappHandler.RegisterRoutes(r)
		})
	})

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	return &Server{
		httpServer: server,
		db:         conn,
		adminUC:    adminUC,
	}, nil
}


func (s *Server) Run() error {
	fmt.Printf("API listening on %s\n", s.httpServer.Addr)
	
	// Start Background Workers
	go s.StartDatabaseStatsWorker()

	return s.httpServer.ListenAndServe()
}

func (s *Server) StartDatabaseStatsWorker() {
	// Initial capture on startup
	log.Println("[Worker] capturing initial database stats...")
	if err := s.adminUC.RecordDatabaseStats(context.Background()); err != nil {
		log.Printf("[Worker] failed to capture initial database stats: %v", err)
	}

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("[Worker] capturing periodic database stats...")
		if err := s.adminUC.RecordDatabaseStats(context.Background()); err != nil {
			log.Printf("[Worker] failed to capture periodic database stats: %v", err)
		}
	}
}

