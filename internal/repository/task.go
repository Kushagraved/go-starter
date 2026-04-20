package repository

import (
	"context"
	"errors"
	"go-starter/internal/models"
	"sync"
	"time"
)

// inMemoryTaskRepo is the in-process implementation of TaskRepository.
type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	store map[string]models.Task
}

func newInMemoryTaskRepo() *inMemoryTaskRepo {
	return &inMemoryTaskRepo{store: make(map[string]models.Task)}
}

func (r *inMemoryTaskRepo) CreateTask(_ context.Context, task models.Task) (models.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	r.store[task.ID] = task
	return task, nil
}

func (r *inMemoryTaskRepo) GetTaskByID(_ context.Context, id string) (models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.store[id]
	if !ok {
		return models.Task{}, errors.New("task not found")
	}
	return t, nil
}

func (r *inMemoryTaskRepo) ListTasks(_ context.Context, filters []models.RepoFilter) ([]models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]models.Task, 0, len(r.store))
	for _, t := range r.store {
		if matchesFilters(t, filters) {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *inMemoryTaskRepo) UpdateTask(_ context.Context, task models.Task) (models.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[task.ID]; !ok {
		return models.Task{}, errors.New("task not found")
	}
	task.UpdatedAt = time.Now()
	r.store[task.ID] = task
	return task, nil
}

func (r *inMemoryTaskRepo) DeleteTask(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[id]; !ok {
		return errors.New("task not found")
	}
	delete(r.store, id)
	return nil
}

// matchesFilters applies RepoFilter key/value pairs against a Task.
func matchesFilters(t models.Task, filters []models.RepoFilter) bool {
	for _, f := range filters {
		switch f.Key {
		case "status":
			if string(t.Status) != f.Value.(string) {
				return false
			}
		case "assignee_id":
			if t.AssigneeID != f.Value.(string) {
				return false
			}
		}
	}
	return true
}
