// Command server is the entry point for the HTTP API and its queue consumers.
//
// The object graph is built by hand here, bottom-up:
//
//	config → logger → redis → postgres → aws → repository → services → controller → router
//
// No DI framework, no service locator. When a service needs a new dependency,
// thread it through below — the dependency graph stays greppable and the startup
// order explicit.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-starter/internal/config"
	"go-starter/internal/controller"
	"go-starter/internal/infrastructure"
	"go-starter/internal/infrastructure/awsclient"
	"go-starter/internal/infrastructure/filestorage"
	infraRedis "go-starter/internal/infrastructure/redis"
	infraSQS "go-starter/internal/infrastructure/sqs"
	"go-starter/internal/logger"
	"go-starter/internal/repository"
	"go-starter/router"

	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	cfgPath := flag.String("config-path", "config/dev/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fatal("config load failed", err)
	}
	settings := cfg.Settings

	appLogger := logger.New(settings.Logger.Level)

	// rootCtx is cancelled on SIGINT/SIGTERM, which stops the consumers.
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	appLogger.Info(rootCtx, "Main", "main", "starting service",
		logger.NewString("env", settings.Env), logger.NewString("port", settings.Server.Port))

	// ── Infrastructure ────────────────────────────────────────────────────────
	cache, err := newCache(settings.Redis)
	if err != nil {
		fatal("redis init failed", err)
	}
	defer cache.Close()

	db, err := newDatabase(settings.SQL)
	if err != nil {
		fatal("postgres init failed", err)
	}

	awsClients, err := awsclient.New(rootCtx, settings.AWS)
	if err != nil {
		fatal("aws init failed", err)
	}

	fileStorage := filestorage.New(awsClients.S3, appLogger)
	producer := infraSQS.NewProducerClient(awsClients.SQS, appLogger)

	// ── Repository ────────────────────────────────────────────────────────────
	repo := repository.New(db, cache, appLogger)

	// ── Services ──────────────────────────────────────────────────────────────
	// Construct core services here, then pass them to controller.Services and to
	// startConsumers.
	//   userService := user.NewService(repo, producer, fileStorage, appLogger)
	_, _, _ = repo, fileStorage, producer

	services := controller.Services{}

	// ── Consumers ─────────────────────────────────────────────────────────────
	startConsumers(rootCtx, settings, awsClients, appLogger, services)

	// ── HTTP ──────────────────────────────────────────────────────────────────
	srv := controller.New(services, appLogger)
	runServer(rootCtx, settings.Server, router.Init(srv, appLogger), appLogger)
}

// newCache opens the Redis connection (database 0) and wires redsync for locking.
func newCache(cfg config.RedisCfg) (infrastructure.Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        cfg.Host,
		Password:    cfg.Password,
		DB:          0,
		DialTimeout: cfg.DialTimeout,
		ReadTimeout: cfg.ReadTimeout,
	})

	// Ping up front so a bad address fails at boot, not on the first request.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return infraRedis.New(client, redsync.New(redsyncredis.NewPool(client))), nil
}

// newDatabase opens Postgres and applies pool settings. The schema is managed
// out of band via scripts/db/schema.sql, not auto-migration.
func newDatabase(cfg config.SQLCfg) (*gorm.DB, error) {
	primary := cfg.Primary
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		primary.Host, primary.Port, primary.User, primary.Password, primary.DBName, primary.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("resolve sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(primary.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(primary.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(primary.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	// No auto-migration: the schema is managed with explicit DDL in
	// scripts/db/schema.sql. Apply it out of band (see the Makefile db-schema
	// target / your migration process).
	return db, nil
}

/*
startConsumers starts every queue consumer.

It takes the same controller.Services the HTTP layer gets, because a consumer IS a
core service: the service that owns a domain implements infrastructure.Consumer
itself, so HTTP and queue traffic enter through one object.

To add one: implement infrastructure.Consumer on the core service, add a named
queue field to config.SQSCfg, and add a single startConsumer line below.
*/
func startConsumers(ctx context.Context, settings config.Settings, awsClients awsclient.Clients,
	appLogger logger.Logger, services controller.Services) {

	sqsCfg := settings.SQS
	if !sqsCfg.Enable {
		appLogger.Info(ctx, "Main", "startConsumers", "sqs disabled, no consumers started")
		return
	}

	// Register consumers here, one line each:
	// startConsumer(ctx, awsClients, appLogger, sqsCfg, sqsCfg.TestEvents, services.User)
	_ = services
}

/*
startConsumer builds and starts the consumer for a single queue. A failure to
resolve the queue is logged and swallowed so a misconfigured consumer cannot stop
the API from serving traffic.
*/
func startConsumer(ctx context.Context, awsClients awsclient.Clients, appLogger logger.Logger,
	sqsCfg config.SQSCfg, queue config.QueueCfg, consumer infrastructure.Consumer) {

	if !queue.Enable {
		appLogger.Info(ctx, "Main", "startConsumer", "queue disabled, skipping",
			logger.NewString("queue_name", queue.QueueName))
		return
	}

	listener, err := infraSQS.NewConsumerClient(ctx, awsClients.SQS, queue.QueueName, sqsCfg.AutoCommit, appLogger)
	if err != nil {
		appLogger.Error(ctx, "Main", "startConsumer", "failed to build consumer",
			logger.NewString("queue_name", queue.QueueName), logger.NewError(err))
		return
	}

	if queue.MaxMessages > 0 {
		listener = listener.WithMaxNumberOfMessages(queue.MaxMessages)
	}
	if queue.VisibilityTimeout > 0 {
		listener = listener.WithVisibilityTimeout(queue.VisibilityTimeout)
	}
	if queue.WaitTimeSeconds > 0 {
		listener = listener.WithWaitTimeSeconds(queue.WaitTimeSeconds)
	}
	if queue.RequireExpBackoff {
		listener = listener.WithExponentialBackoff()
	}

	workers := queue.Workers
	if workers <= 0 {
		workers = sqsCfg.Workers
	}
	listener.StartConsumer(ctx, consumer, workers)
}

// runServer serves HTTP until ctx is cancelled, then drains in-flight requests.
func runServer(ctx context.Context, cfg config.ServerCfg, handler http.Handler, appLogger logger.Logger) {
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		appLogger.Info(ctx, "Main", "runServer", "listening", logger.NewString("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		fatal("http server failed", err)
	case <-ctx.Done():
		appLogger.Info(context.Background(), "Main", "runServer", "shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error(shutdownCtx, "Main", "runServer", "graceful shutdown failed", logger.NewError(err))
		return
	}
	appLogger.Info(context.Background(), "Main", "runServer", "shutdown complete")
}

func fatal(message string, err error) {
	fmt.Fprintf(os.Stderr, "fatal: %s: %v\n", message, err)
	os.Exit(1)
}
