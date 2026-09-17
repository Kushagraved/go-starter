// Command script is the file-based CLI driver for the expense-sharing domain.
//
// It reads a file of commands (one per line) in the problem-statement grammar,
// executes each against the core service, and prints the result. Blank lines and
// lines starting with '#' are ignored.
//
//	go run ./cmd/script -config-path config/dev/config.yaml -file commands.txt
//
// The object graph is the same as cmd/server, minus the HTTP server and queue
// consumers: config → logger → redis → postgres → repository → service.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"go-starter/internal/config"
	"go-starter/internal/core/expensesharing"
	"go-starter/internal/infrastructure"
	infraRedis "go-starter/internal/infrastructure/redis"
	"go-starter/internal/logger"
	"go-starter/internal/repository"

	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis/goredis/v9"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	cfgPath := flag.String("config-path", "config/dev/config.yaml", "path to config file")
	filePath := flag.String("file", "scripts/expense-commands.txt", "path to a file of commands to execute")
	flag.Parse()

	if *filePath == "" {
		fatal("a command file is required: pass -file <path>", nil)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fatal("config load failed", err)
	}
	settings := cfg.Settings

	appLogger := logger.New(settings.Logger.Level)
	ctx := context.Background()

	cache, err := newCache(settings.Redis)
	if err != nil {
		fatal("redis init failed", err)
	}

	db, err := newDatabase(settings.SQL)
	if err != nil {
		fatal("postgres init failed", err)
	}

	repo := repository.New(db, cache, appLogger)
	svc := expensesharing.NewService(repo, appLogger)
	exec := &executor{svc: svc}

	if err := runFile(ctx, exec, *filePath); err != nil {
		fatal("execution failed", err)
	}
}

// runFile executes every command line in the file, printing output as it goes.
// A command error is printed and does not stop the run, so one bad line does not
// abort the whole file.
func runFile(ctx context.Context, exec *executor, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		out, err := exec.Run(ctx, line)
		if err != nil {
			fmt.Printf("line %d: error: %v\n", lineNo, err)
			continue
		}
		for _, l := range out {
			fmt.Println(l)
		}
	}
	return scanner.Err()
}

// newCache opens Redis and wires redsync for locking (same as cmd/server).
func newCache(cfg config.RedisCfg) (infrastructure.Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        cfg.Host,
		Password:    cfg.Password,
		DB:          0,
		DialTimeout: cfg.DialTimeout,
		ReadTimeout: cfg.ReadTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return infraRedis.New(client, redsync.New(redsyncredis.NewPool(client))), nil
}

// newDatabase opens Postgres with pool settings (same as cmd/server).
func newDatabase(cfg config.SQLCfg) (*gorm.DB, error) {
	p := cfg.Primary
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode)

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
	sqlDB.SetMaxOpenConns(p.MaxOpenConnections)
	sqlDB.SetMaxIdleConns(p.MaxIdleConnections)
	sqlDB.SetConnMaxLifetime(p.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

func fatal(message string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s: %v\n", message, err)
	} else {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", message)
	}
	os.Exit(1)
}
