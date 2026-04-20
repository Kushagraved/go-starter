package internal

// Server is the top-level interface for all HTTP handler groups.
type Server interface {
	TaskServer
}

// TaskServer groups all task-related HTTP handlers.
type TaskServer interface {
	CreateTask(ctx interface{})
	GetTask(ctx interface{})
	ListTasks(ctx interface{})
	UpdateTask(ctx interface{})
	DeleteTask(ctx interface{})
}
