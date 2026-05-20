package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pekan/backend/internal/platform/config"
	"pekan/backend/internal/platform/db"
	"pekan/backend/internal/platform/filescan"
	"pekan/backend/internal/platform/storage"
	reminderinfra "pekan/backend/internal/modules/finance/reminders/infra"
	reminderworker "pekan/backend/internal/modules/finance/reminders/worker"
	notificationinfra "pekan/backend/internal/modules/finance/notifications/infra"
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

	objectStorage := storage.NewProvider(cfg)
	scanService := filescan.NewService(
		conn,
		objectStorage,
		filescan.NewSignatureScanner(),
		cfg.FileScanPollInterval,
		cfg.FileScanMaxAttempts,
		cfg.FileScanRetryDelay,
	)
	reminderRepo := reminderinfra.NewRepositoryPG(conn)
	notificationRepo := notificationinfra.NewRepositoryPG(conn)
	reminderService := reminderworker.NewService(reminderRepo, notificationRepo, cfg.ReminderPollInterval)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	log.Printf("file scan worker started")
	go func() {
		errCh <- scanService.Run(ctx)
	}()
	log.Printf("reminder worker started")
	go func() {
		errCh <- reminderService.Run(ctx)
	}()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			log.Fatalf("worker stopped with error: %v", err)
		}
	}
	log.Printf("workers stopped")
}
