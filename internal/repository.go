package internal

import (
	"context"
	"go-starter/internal/models"
)

// Repository is the top-level interface composed of all domain DAOs + infrastructure concerns.
type Repository interface {
	DAO
	Lock
}

// Lock provides distributed locking.
type Lock interface {
	DistributedLock(ctx context.Context, key string, fn func() error) error
}

// DAO composes all domain-specific repository interfaces.
type DAO interface {
	// Atomic wraps fn in a DB transaction; the DAO passed to fn shares the transaction.
	Atomic(ctx context.Context, fn func(store DAO) error) error
	TaskRepository
}

// TaskRepository is the data-access contract for the Task domain.
type TaskRepository interface {
	CreateTask(ctx context.Context, task models.Task) (models.Task, error)
	GetTaskByID(ctx context.Context, id string) (models.Task, error)
	ListTasks(ctx context.Context, filters []models.RepoFilter) ([]models.Task, error)
	UpdateTask(ctx context.Context, task models.Task) (models.Task, error)
	DeleteTask(ctx context.Context, id string) error
}
