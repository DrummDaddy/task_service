package usecase

import (
	"context"
	"strconv"
	"strings"

	"github.com/DrummDaddy/task_service/internal/core/ports"
	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/DrummDaddy/task_service/internal/repo"
)

type TaskUsecase struct {
	tasks ports.TaskRepository
	teams ports.TeamRepository
	cache ports.TasksCache
}

func NewTaskUsecase(tasks ports.TaskRepository, teams ports.TeamRepository, cache ports.TasksCache) *TaskUsecase {
	return &TaskUsecase{tasks: tasks, teams: teams, cache: cache}
}

type CreateTaskInput struct {
	UserID      uint64
	TeamID      uint64
	Title       string
	Description *string
	Status      models.TaskStatus
	AssigneeID  *uint64
}

func (uc *TaskUsecase) Create(ctx context.Context, in CreateTaskInput) (uint64, error) {
	title := strings.TrimSpace(in.Title)
	if in.TeamID == 0 || title == "" {
		return 0, repo.ErrConflict
	}
	if in.Status == "" {
		in.Status = models.StatusToDo
	}
	if in.Status != models.StatusToDo && in.Status != models.StatusInProgress && in.Status != models.StatusDone {
		return 0, repo.ErrConflict
	}
	isMember, err := uc.teams.IsMember(ctx, in.TeamID, in.UserID)
	if err != nil {
		return 0, err
	}
	if !isMember {
		return 0, repo.ErrConflict
	}
	if in.AssigneeID != nil {
		ok, err := uc.teams.IsMember(ctx, in.TeamID, *in.AssigneeID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, repo.ErrConflict
		}
	}

	id, err := uc.tasks.Create(ctx, models.Task{
		TeamID:      in.TeamID,
		Title:       title,
		Description: in.Description,
		Status:      in.Status,
		AssigneeID:  in.AssigneeID,
		CreatedBy:   in.UserID,
	})
	if err != nil {
		return 0, err
	}
	if uc.cache != nil {
		_ = uc.cache.BumpTeamVersion(ctx, in.TeamID)
	}
	return id, nil
}

type ListTaskInput struct {
	UserID   uint64
	TeamID   uint64
	Status   *models.TaskStatus
	Assignee *uint64
	Limit    int
	Offset   int
}

func (uc *TaskUsecase) List(ctx context.Context, in ListTaskInput) ([]models.Task, error) {
	if in.TeamID == 0 {
		return nil, repo.ErrConflict
	}
	isMember, err := uc.teams.IsMember(ctx, in.TeamID, in.UserID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, repo.ErrConflict
	}

	filter := models.TaskFilter{
		TeamID:     in.TeamID,
		Status:     in.Status,
		AssigneeID: in.Assignee,
		Limit:      in.Limit,
		Offset:     in.Offset,
	}

	if uc.cache != nil {
		if ver, err := uc.cache.GetTeamVersion(ctx, in.TeamID); err == nil {
			key := uc.cache.ListKey(in.TeamID, ver, filter)
			if items, ok, err := uc.cache.GetTasks(ctx, key); err == nil && ok {
				return items, nil
			}
		}
	}

	items, err := uc.tasks.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	if uc.cache != nil {
		if ver, err := uc.cache.GetTeamVersion(ctx, in.TeamID); err == nil {
			_ = uc.cache.SetTasks(ctx, uc.cache.ListKey(in.TeamID, ver, filter), items)
		}
	}
	return items, nil
}

type UpdateTaskInput struct {
	UserID      uint64
	TaskID      uint64
	Title       *string
	Description **string
	Status      *models.TaskStatus
	AssigneeID  **uint64
}

func (uc *TaskUsecase) Update(ctx context.Context, in UpdateTaskInput) (models.Task, error) {
	cur, err := uc.tasks.Get(ctx, in.TaskID)
	if err != nil {
		return models.Task{}, err
	}
	role, err := uc.teams.GetUserRole(ctx, cur.TeamID, in.UserID)
	if err != nil {
		return models.Task{}, repo.ErrNotFound
	}
	canEdit := role == models.RoleOwner || role == models.RoleAdmin ||
		cur.CreatedBy == in.UserID ||
		(cur.AssigneeID != nil && *cur.AssigneeID == in.UserID)
	if !canEdit {
		return models.Task{}, repo.ErrConflict
	}
	if in.Status != nil {
		if *in.Status != models.StatusToDo && *in.Status != models.StatusInProgress && *in.Status != models.StatusDone {
			return models.Task{}, repo.ErrConflict
		}
	}
	if in.AssigneeID != nil && *in.AssigneeID != nil {
		ok, err := uc.teams.IsMember(ctx, cur.TeamID, **in.AssigneeID)
		if err != nil {
			return models.Task{}, err
		}
		if !ok {
			return models.Task{}, repo.ErrConflict
		}
	}

	updated, err := uc.tasks.UpdateWithHistory(ctx, in.TaskID, repo.TaskUpdate{
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		AssigneeID:  in.AssigneeID,
	}, in.UserID)
	if err != nil {
		return models.Task{}, err
	}
	if uc.cache != nil {
		_ = uc.cache.BumpTeamVersion(ctx, cur.TeamID)
	}
	return updated, nil
}

func (uc *TaskUsecase) History(ctx context.Context, userID, taskID uint64, limit, offset int) ([]models.TaskHistoryItem, error) {
	cur, err := uc.tasks.Get(ctx, taskID)
	if err != nil {
		return nil, repo.ErrNotFound
	}
	isMember, err := uc.teams.IsMember(ctx, cur.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, repo.ErrConflict
	}
	return uc.tasks.History(ctx, taskID, limit, offset)
}

func (uc *TaskUsecase) AddComment(ctx context.Context, userID, taskID uint64, body string) (uint64, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return 0, repo.ErrConflict
	}
	cur, err := uc.tasks.Get(ctx, taskID)
	if err != nil {
		return 0, repo.ErrNotFound
	}
	isMember, err := uc.teams.IsMember(ctx, cur.TeamID, userID)
	if err != nil {
		return 0, err
	}
	if !isMember {
		return 0, repo.ErrConflict
	}
	return uc.tasks.AddComment(ctx, taskID, userID, body)
}

func (uc *TaskUsecase) ListComments(ctx context.Context, userID, taskID uint64, limit, offset int) ([]models.TaskComment, error) {
	cur, err := uc.tasks.Get(ctx, taskID)
	if err != nil {
		return nil, repo.ErrNotFound
	}
	isMember, err := uc.teams.IsMember(ctx, cur.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, repo.ErrConflict
	}
	return uc.tasks.ListComments(ctx, taskID, limit, offset)
}

func ParseUintParam(s string) (uint64, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return v, err
}
