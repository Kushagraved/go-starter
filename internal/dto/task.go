package dto

import (
	apitask "go-starter/api/task"
	"go-starter/internal/models"
	"time"
)

func ToTaskResponse(t models.Task) apitask.TaskResponse {
	return apitask.TaskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		AssigneeID:  t.AssigneeID,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
}

func ToTaskListResponse(tasks []models.Task) apitask.ListTasksResponse {
	items := make([]apitask.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, ToTaskResponse(t))
	}
	return apitask.ListTasksResponse{Tasks: items, Total: len(items)}
}
