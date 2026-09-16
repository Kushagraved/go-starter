package repository

import (
	"context"
	"time"

	"go-starter/internal"
	"go-starter/internal/logger"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

/*
Atomic opens a transaction and hands fn a DAO bound to it. The scoped DAO is this
same client with `db` pointing at the transaction, so every domain method
participates without any per-method plumbing.

Commit and rollback happen in a deferred call, which means an error return, an
explicit panic, and a nil-pointer panic all roll back the same way.
*/
func (c client) Atomic(ctx context.Context, fn func(store internal.DAO) error) (err error) {
	tx := c.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		c.logger.Error(ctx, "DAO", "Atomic", "error opening transaction", logger.NewError(tx.Error))
		return tx.Error
	}

	defer func() {
		err = c.finish(ctx, tx, err)
	}()

	return fn(client{db: tx, cache: c.cache, logger: c.logger})
}

// finish rolls back on error or panic, otherwise commits.
func (c client) finish(ctx context.Context, tx *gorm.DB, err error) error {
	if r := recover(); r != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			c.logger.Error(ctx, "DAO", "Atomic", "panic in caller and rollback failed",
				logger.NewInterface("panic", r), logger.NewError(rbErr))
		}
		return errors.Errorf("panic in transaction: %v", r)
	}

	if err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			c.logger.Error(ctx, "DAO", "Atomic", "error rolling back transaction",
				logger.NewError(err), logger.NewString("rollback_error", rbErr.Error()))
		}
		return err
	}

	if commitErr := tx.Commit().Error; commitErr != nil {
		c.logger.Error(ctx, "DAO", "Atomic", "error committing transaction", logger.NewError(commitErr))
		return commitErr
	}
	return nil
}

// DistributedLock delegates to Redis so services depend only on the repository.
func (c client) DistributedLock(ctx context.Context, key string, expiry time.Duration, execFunc func() error) error {
	return c.cache.DistributedLock(ctx, key, expiry, execFunc)
}
