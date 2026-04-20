package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"go-starter/internal/config"
	"go-starter/internal/controller"
	"go-starter/internal/core/task"
	infraQueue "go-starter/internal/infrastructure/queue"
	infraRedis "go-starter/internal/infrastructure/redis"
	infraStorage "go-starter/internal/infrastructure/storage"
	"go-starter/internal/logger"
	"go-starter/internal/repository"
	"go-starter/router"
)

func main() {
	cfgPath := flag.String("config-path", "config/dev/config.yaml", "path to config file")
	flag.Parse()

	cfg := config.Load(*cfgPath)
	log.Printf("starting go-starter env=%s db=%s queue=%s",
		cfg.Settings.Env, cfg.Settings.DBDriver, cfg.Settings.Queue.Driver)

	// ── Logger ────────────────────────────────────────────────────────────────
	appLogger := logger.New(cfg.Settings.Logger.Level)

	// ── Redis ─────────────────────────────────────────────────────────────────
	redisClient := infraRedis.New(
		cfg.Settings.Redis.Host,
		cfg.Settings.Redis.Password,
		cfg.Settings.Redis.DB,
	)

	// ── Repository ────────────────────────────────────────────────────────────
	dsn := buildDSN(cfg.Settings.SQL.Primary)
	repo, err := repository.New(cfg.Settings.DBDriver, dsn, redisClient)
	if err != nil {
		log.Fatalf("repository init failed: %v", err)
	}

	// ── Queue ─────────────────────────────────────────────────────────────────
	// driver="memory" → in-process channels; swap for SQS client when needed.
	var queueDriver string
	if cfg.Settings.Queue.Driver != "" {
		queueDriver = cfg.Settings.Queue.Driver
	} else {
		queueDriver = "memory"
	}
	var producer infraQueue.Producer
	if queueDriver == "memory" {
		mq := infraQueue.NewMemoryQueue(256)
		producer = mq
		// wire consumers
		ctx := context.Background()
		_ = mq.Subscribe(ctx, "task_events", func(_ context.Context, msg infraQueue.Message) error {
			appLogger.Info(ctx, "queue", "task_events", "received event",
				logger.String("payload", string(msg.Payload)))
			return nil
		})
	}
	_ = producer // available for injection into services that need to publish events

	// ── Storage ───────────────────────────────────────────────────────────────
	// local disk in dev; swap for S3 implementation in prod.
	_ = infraStorage.NewLocal("./tmp/uploads")

	// ── Services ──────────────────────────────────────────────────────────────
	taskService := task.NewService(repo)

	// ── Controller ────────────────────────────────────────────────────────────
	srv := controller.New(controller.Services{
		Task: taskService,
	}, appLogger)

	// ── Router ────────────────────────────────────────────────────────────────
	r := router.Init(srv, appLogger)

	port := cfg.Settings.Server.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func buildDSN(db config.DBSettings) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.DBName)
}
