package ports

import (
	"context"
	"time"

	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/DrummDaddy/task_service/internal/repo"
)

type UserRepository interface {
	Create(ctx context.Context, email string, passwordHash []byte) (uint64, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
}

type TeamRepository interface {
	CreateTeamWithOwner(ctx context.Context, name string, createdBy uint64) (uint64, error)
	ListTeamsByUser(ctx context.Context, userID uint64) ([]models.Team, error)
	GetUserRole(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error)
	IsMember(ctx context.Context, teamID, userID uint64) (bool, error)
	AddMember(ctx context.Context, teamID uint64, userID uint64, role models.TeamMemberRole) error
}

type TaskRepository interface {
	Create(ctx context.Context, t models.Task) (uint64, error)
	Get(ctx context.Context, id uint64) (models.Task, error)
	List(ctx context.Context, f models.TaskFilter) ([]models.Task, error)
	UpdateWithHistory(ctx context.Context, taskID uint64, upd repo.TaskUpdate, changeBy uint64) (models.Task, error)
	History(ctx context.Context, taskID uint64, limit, offset int) ([]models.TaskHistoryItem, error)
	AddComment(ctx context.Context, taskID, userID uint64, body string) (uint64, error)
	ListComments(ctx context.Context, taskID uint64, limit, offset int) ([]models.TaskComment, error)
}

type ReportsRepository interface {
	TeamStats(ctx context.Context, now time.Time) ([]models.TeamStatsRow, error)
	TopCreatorsLastMonth(ctx context.Context) ([]models.TopCreatorRow, error)
	TaskWithInvalidAssignee(ctx context.Context, limit, offset int) ([]models.IntegrityTaskRow, error)
}
