package infrastructure

import (
	"context"
	"time"
)

// Cache is the Redis contract: key/value access plus distributed locking.
// Keep it small — add methods here as you actually need them.
type Cache interface {
	// Get returns constants.RecordDoesntExists when the key is absent.
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiry time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	// DistributedLock runs execFunc while holding a cluster-wide lock on key.
	// Returns constants.LockAlreadyHeld when the lock cannot be acquired.
	DistributedLock(ctx context.Context, key string, expiry time.Duration, execFunc func() error) error

	// Close releases the underlying connection pool. Owning this here means main
	// holds one value instead of juggling the interface and the raw client.
	Close() error
}
