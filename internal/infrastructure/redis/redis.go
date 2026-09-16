// Package redis implements infrastructure.Cache on top of a single Redis
// connection (database 0), with redsync providing the distributed lock.
package redis

import (
	"context"
	"time"

	"go-starter/internal/constants"
	"go-starter/internal/helpers/app"
	"go-starter/internal/infrastructure"

	"github.com/go-redsync/redsync/v4"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

// defaultLockExpiry is the lock TTL used when the caller passes 0.
const defaultLockExpiry = 15 * time.Minute

type client struct {
	rdb        *redis.Client
	lockClient *redsync.Redsync
}

// New wraps a Redis connection as the app's Cache.
func New(rdb *redis.Client, lockClient *redsync.Redsync) infrastructure.Cache {
	return client{rdb: rdb, lockClient: lockClient}
}

func (c client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", constants.RecordDoesntExists
	}
	return value, err
}

func (c client) Set(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiry).Err()
}

func (c client) Delete(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.rdb.Exists(ctx, key).Result()
	return count > 0, err
}

func (c client) Close() error {
	return c.rdb.Close()
}

/*
DistributedLock acquires a Redis-backed mutex, runs execFunc, then releases it.

  - contention returns constants.LockAlreadyHeld, so callers can tell "someone
    else is doing this" apart from "the work failed";
  - a panic inside execFunc is converted to an error and the lock is still released;
  - expiry of 0 falls back to defaultLockExpiry.
*/
func (c client) DistributedLock(ctx context.Context, key string, expiry time.Duration, execFunc func() error) (err error) {
	if c.lockClient == nil {
		return errors.New("lock client is nil")
	}
	if expiry <= 0 {
		expiry = defaultLockExpiry
	}

	mutex := c.lockClient.NewMutex(key, redsync.WithExpiry(expiry))
	if lockErr := mutex.LockContext(ctx); lockErr != nil {
		return constants.LockAlreadyHeld
	}

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("recovered from panic: %v\n%s", r, app.StackTrace())
		}
		if _, unlockErr := mutex.UnlockContext(ctx); unlockErr != nil && err == nil {
			err = errors.Wrap(unlockErr, "error releasing lock")
		}
	}()

	return execFunc()
}
