package task

import (
	"context"
	"encoding/json"
	"errors"

	"go-starter/internal"
	infraQueue "go-starter/internal/infrastructure/queue"
	"go-starter/internal/logger"
	"go-starter/internal/models"

	"github.com/gofrs/uuid"
)

// topic names this service produces and consumes
const TopicTaskCreated = "task.created"

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

// createdPayload is the message body for TopicTaskCreated.
type createdPayload struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
}

type service struct {
	repo     internal.Repository
	producer infraQueue.PushEvents
	log      logger.Logger
}

// TaskService combines the domain Service interface with queue.Consumer
// so main can pass it to StartConsumer without a type assertion.
type TaskService interface {
	Service
	infraQueue.Consumer
}

// NewService returns a TaskService. It also implements queue.Consumer so it can be
// passed directly to queue.ListenToEvents.StartConsumer in main.
func NewService(repo internal.Repository, producer infraQueue.PushEvents, log logger.Logger) TaskService {
	return &service{repo: repo, producer: producer, log: log}
}

// ── queue.Consumer implementation ────────────────────────────────────────────

func (s *service) GetTxnName() string { return "task-consumer" }

func (s *service) Unmarshal(_ context.Context, msg infraQueue.Message) (interface{}, error) {
	var p createdPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) ProcessData(ctx context.Context, topic string, data interface{}) error {
	switch topic {
	case TopicTaskCreated:
		p := data.(createdPayload)
		s.log.Info(ctx, "task", "ProcessData", "task created event received",
			logger.String("task_id", p.TaskID),
			logger.String("title", p.Title),
		)
		// TODO: downstream side-effects — notifications, audit log, etc.
	}
	return nil
}

// ── Service methods ───────────────────────────────────────────────────────────

func (s *service) Create(ctx context.Context, req CreateRequest) (models.Task, error) {
	if req.Title == "" {
		return models.Task{}, errors.New("title is required")
	}
	id, err := uuid.NewV4()
	if err != nil {
		return models.Task{}, err
	}
	t := models.Task{
		ID:          id.String(),
		Title:       req.Title,
		Description: req.Description,
		Status:      models.TaskStatusPending,
		AssigneeID:  req.AssigneeID,
	}
	created, err := s.repo.CreateTask(ctx, t)
	if err != nil {
		return models.Task{}, err
	}

	payload, _ := json.Marshal(createdPayload{TaskID: created.ID, Title: created.Title})
	_ = s.producer.Produce(ctx, infraQueue.Message{Topic: TopicTaskCreated, Payload: payload})

	return created, nil
}

func (s *service) Get(ctx context.Context, id string) (models.Task, error) {
	return s.repo.GetTaskByID(ctx, id)
}

func (s *service) List(ctx context.Context, filters []models.RepoFilter) ([]models.Task, error) {
	return s.repo.ListTasks(ctx, filters)
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (models.Task, error) {
	t, err := s.repo.GetTaskByID(ctx, id)
	if err != nil {
		return models.Task{}, err
	}
	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Status != nil {
		t.Status = *req.Status
	}
	if req.AssigneeID != nil {
		t.AssigneeID = *req.AssigneeID
	}
	return s.repo.UpdateTask(ctx, t)
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.DeleteTask(ctx, id)
}
