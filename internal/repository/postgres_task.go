package repository

import (
	"context"
	"go-starter/internal/models"

	"gorm.io/gorm"
)

// postgresTaskRepo is the Postgres-backed implementation of TaskRepository.
type postgresTaskRepo struct {
	db *gorm.DB
}

func newPostgresTaskRepo(db *gorm.DB) *postgresTaskRepo {
	return &postgresTaskRepo{db: db}
}

func (r *postgresTaskRepo) CreateTask(ctx context.Context, task models.Task) (models.Task, error) {
	if err := r.db.WithContext(ctx).Create(&task).Error; err != nil {
		return models.Task{}, err
	}
	return task, nil
}

func (r *postgresTaskRepo) GetTaskByID(ctx context.Context, id string) (models.Task, error) {
	var task models.Task
	if err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		return models.Task{}, err
	}
	return task, nil
}

func (r *postgresTaskRepo) ListTasks(ctx context.Context, filters []models.RepoFilter) ([]models.Task, error) {
	q := r.db.WithContext(ctx).Model(&models.Task{})
	for _, f := range filters {
		q = q.Where(f.Key+" = ?", f.Value)
	}
	var tasks []models.Task
	if err := q.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *postgresTaskRepo) UpdateTask(ctx context.Context, task models.Task) (models.Task, error) {
	if err := r.db.WithContext(ctx).Save(&task).Error; err != nil {
		return models.Task{}, err
	}
	return task, nil
}

func (r *postgresTaskRepo) DeleteTask(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Task{}, "id = ?", id).Error
}
