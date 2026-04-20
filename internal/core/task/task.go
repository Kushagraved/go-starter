package task

import (
	"context"
	"errors"
	"go-starter/internal"
	"go-starter/internal/models"

	"github.com/gofrs/uuid"
)

// Service defines the business-logic contract for the Task domain.
type Service interface {
	Create(ctx context.Context, req CreateRequest) (models.Task, error)
	Get(ctx context.Context, id string) (models.Task, error)
	List(ctx context.Context, filters []models.RepoFilter) ([]models.Task, error)
	Update(ctx context.Context, id string, req UpdateRequest) (models.Task, error)
	Delete(ctx context.Context, id string) error
}

type CreateRequest struct {
	Title       string `json:"title"       validate:"required"`
	Description string `json:"description"`
	AssigneeID  string `json:"assignee_id"`
}

type UpdateRequest struct {
	Title       *string            `json:"title"`
	Description *string            `json:"description"`
	Status      *models.TaskStatus `json:"status"`
	AssigneeID  *string            `json:"assignee_id"`
}

type service struct {
	repo internal.Repository
}

func NewService(repo internal.Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req CreateRequest) (models.Task, error) {
	if req.Title == "" {
		return models.Task{}, errors.New("title is required")
	}
	id, err := uuid.NewV4()
	if err != nil {
		return models.Task{}, err
	}
	task := models.Task{
		ID:          id.String(),
		Title:       req.Title,
		Description: req.Description,
		Status:      models.TaskStatusPending,
		AssigneeID:  req.AssigneeID,
	}
	return s.repo.CreateTask(ctx, task)
}

func (s *service) Get(ctx context.Context, id string) (models.Task, error) {
	return s.repo.GetTaskByID(ctx, id)
}

func (s *service) List(ctx context.Context, filters []models.RepoFilter) ([]models.Task, error) {
	return s.repo.ListTasks(ctx, filters)
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (models.Task, error) {
	task, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		return models.Task{}, err
	}
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.AssigneeID != nil {
		task.AssigneeID = *req.AssigneeID
	}
	return s.repo.UpdateTask(ctx, task)
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.DeleteTask(ctx, id)
}
