package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"pekan/backend/internal/platform/config"
	"pekan/backend/internal/platform/db"
	"pekan/backend/internal/platform/access"
	admininfra "pekan/backend/internal/modules/core/admin/infra"
	adminusecase "pekan/backend/internal/modules/core/admin/usecase"
	whatsappinfra "pekan/backend/internal/modules/finance/whatsapp/infra"
	whatsappusecase "pekan/backend/internal/modules/finance/whatsapp/usecase"
	"pekan/backend/internal/platform/audit"
	"pekan/backend/internal/platform/security"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	conn, err := db.NewPostgres(cfg.DatabaseURL, db.PoolConfig{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer conn.Close()

	cipher, err := security.NewCipher(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("failed to create cipher: %v", err)
	}

	authorizer := access.NewAuthorizer()
	auditLogger := audit.NewDBLogger(conn)

	// Admin service (rdb and storage are nil as they are not needed for GetGlobalSettingRaw)
	adminRepo := admininfra.NewRepositoryPG(conn)
	adminUC := adminusecase.NewService(adminRepo, auditLogger, cfg.ReceiptScanSecret, nil, nil, conn)

	// WhatsApp Chatbot Queue Service
	whatsappRepo := whatsappinfra.NewRepositoryPG(conn, cipher)
	whatsappUC := whatsappusecase.NewService(whatsappRepo, authorizer, adminUC)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("[PEKAN-AI] Asynchronous AI Queue Worker Service started")
	
	// Start worker goroutines (configurable via AI_QUEUE_WORKERS env var, default to 4)
	numWorkers := 4
	if envVal := os.Getenv("AI_QUEUE_WORKERS"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			numWorkers = val
		}
	}
	log.Printf("[PEKAN-AI] Running %d concurrent worker threads!", numWorkers)
	whatsappUC.StartQueueWorker(ctx, numWorkers)

	// Block main thread until context is cancelled
	<-ctx.Done()

	log.Printf("[PEKAN-AI] AI Queue Worker stopping gracefully")
}
