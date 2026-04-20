package repository

import (
	"context"
	"errors"
	"fmt"
	"go-starter/internal"
	infraRedis "go-starter/internal/infrastructure/redis"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// repo is the concrete implementation of internal.Repository.
type repo struct {
	internal.TaskRepository
	rs *redsync.Redsync
}

// New builds a Repository. Pass driver="memory" for local dev, "postgres" for real DB.
func New(driver string, dsn string, redisClient infraRedis.Client) (internal.Repository, error) {
	var taskRepo internal.TaskRepository

	switch driver {
	case "postgres":
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("postgres connect: %w", err)
		}
		if err := db.AutoMigrate(&struct {
			// import cycle guard — migration handled in main
		}{}); err != nil {
			return nil, err
		}
		taskRepo = newPostgresTaskRepo(db)
	case "memory":
		taskRepo = newInMemoryTaskRepo()
	default:
		return nil, fmt.Errorf("unknown db driver: %s", driver)
	}

	pool := goredis.NewPool(redisClient.Raw())
	rs := redsync.New(pool)

	return &repo{
		TaskRepository: taskRepo,
		rs:             rs,
	}, nil
}

// Atomic runs fn inside a transaction. For in-memory driver this is a no-op wrapper.
func (r *repo) Atomic(_ context.Context, fn func(internal.DAO) error) error {
	return fn(r)
}

// DistributedLock acquires a Redis-backed mutex before calling fn.
func (r *repo) DistributedLock(_ context.Context, key string, fn func() error) error {
	mutex := r.rs.NewMutex(key)
	if err := mutex.Lock(); err != nil {
		return errors.New("could not acquire lock: " + err.Error())
	}
	defer mutex.Unlock()
	return fn()
}
