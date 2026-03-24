package ports

import (
	"context"

	"github.com/DrummDaddy/task_service/internal/models"
)

type TasksCache interface {
	GetTeamVersion(ctx context.Context, teamID uint64) (string, error)
	BumpTeamVersion(ctx context.Context, teamID uint64) error
	ListKey(teamID uint64, ver string, f models.TaskFilter) string
	GetTasks(ctx context.Context, key string) ([]models.Task, bool, error)
	SetTasks(ctx context.Context, key string, items []models.Task) error
}
