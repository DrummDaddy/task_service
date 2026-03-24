package models

import "time"

type TaskStatus string

const (
	StatusToDo       TaskStatus = "to_do"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

type Task struct {
	ID          uint64     `json:"id"`
	TeamID      uint64     `json:"team_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      TaskStatus `json:"status"`
	AssigneeID  *uint64    `json:"assignee_id"`
	CreatedBy   uint64     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TaskHistoryItem struct {
	ID        uint64    `json:"id"`
	TaskID    uint64    `json:"task_id"`
	ChangedBy uint64    `json:"changed_by"`
	FieldName string    `json:"field_name"`
	OldValue  *string   `json:"old_value"`
	NewValue  *string   `json:"new_value"`
	ChangedAt time.Time `json:"changed_at"`
}

type TaskComment struct {
	ID        uint64    `json:"id"`
	TaskID    uint64    `json:"task_id"`
	UserID    uint64    `json:"user_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskFilter struct {
	TeamID     uint64
	Status     *TaskStatus
	AssigneeID *uint64
	Limit      int
	Offset     int
}
