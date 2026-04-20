package main

import (
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
	// driver="memory" → in-process channels; swap for SQS without touching anything else.
	mq := buildQueue(cfg.Settings.Queue.Driver, appLogger)

	// ── Storage ───────────────────────────────────────────────────────────────
	_ = infraStorage.NewLocal("./tmp/uploads")

	// ── Services ──────────────────────────────────────────────────────────────
	// Each service receives the producer so it can emit events,
	// and implements queue.Consumer so it can process events it subscribes to.
	taskService := task.NewService(repo, mq, appLogger)

	// ── Start consumers ───────────────────────────────────────────────────────
	// The service itself is the consumer — no separate handler wiring needed.
	mq.StartConsumer(taskService, 5)

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

// buildQueue returns a queue that implements both ListenToEvents and PushEvents.
// Swap "memory" for an SQS/Kafka implementation without touching anything else.
func buildQueue(driver string, log logger.Logger) interface {
	infraQueue.ListenToEvents
	infraQueue.PushEvents
} {
	switch driver {
	case "sqs":
		// TODO: return sqsQueue.New(cfg)
		panic("SQS queue driver not yet implemented")
	}
	return infraQueue.NewMemoryQueue(256, log)
}

func buildDSN(db config.DBSettings) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.DBName)
}
