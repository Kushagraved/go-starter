package models

import (
	"time"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

// Task is the primary domain entity.
type Task struct {
	ID          string     `gorm:"primaryKey;type:uuid" json:"id"`
	Title       string     `gorm:"not null"             json:"title"`
	Description string     `                            json:"description"`
	Status      TaskStatus `gorm:"default:'pending'"    json:"status"`
	AssigneeID  string     `                            json:"assignee_id"`
	CreatedAt   time.Time  `                            json:"created_at"`
	UpdatedAt   time.Time  `                            json:"updated_at"`
}

// RepoFilter is a generic key/value filter used across repository queries.
type RepoFilter struct {
	Key   string
	Value interface{}
}
