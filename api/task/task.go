package task

import "go-starter/internal/models"

// CreateTaskRequest is the JSON body for POST /tasks.
type CreateTaskRequest struct {
	Title       string `json:"title"        binding:"required"`
	Description string `json:"description"`
	AssigneeID  string `json:"assignee_id"`
}

// UpdateTaskRequest is the JSON body for PATCH /tasks/:id.
type UpdateTaskRequest struct {
	Title       *string            `json:"title"`
	Description *string            `json:"description"`
	Status      *models.TaskStatus `json:"status"`
	AssigneeID  *string            `json:"assignee_id"`
}

// TaskResponse is the standard task payload returned to callers.
type TaskResponse struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      models.TaskStatus `json:"status"`
	AssigneeID  string            `json:"assignee_id"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// ListTasksResponse wraps a slice of tasks.
type ListTasksResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Total int            `json:"total"`
}
