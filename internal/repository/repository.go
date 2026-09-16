// Package repository is the only gateway to the database. Every domain's queries
// hang off the single `client` struct (one file per domain), which is what lets
// Atomic scope the whole DAO by swapping a single *gorm.DB.
package repository

import (
	"go-starter/internal"
	"go-starter/internal/infrastructure"
	"go-starter/internal/logger"

	"gorm.io/gorm"
)

type client struct {
	db     *gorm.DB
	cache  infrastructure.Cache
	logger logger.Logger
}

// New builds the repository over Postgres and Redis.
func New(db *gorm.DB, cache infrastructure.Cache, loggerObj logger.Logger) internal.Repository {
	return client{
		db:     db,
		cache:  cache,
		logger: loggerObj,
	}
}
