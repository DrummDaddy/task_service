package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/stretchr/testify/require"
)

type mockTeamRepo struct {
	isMemberFn   func(ctx context.Context, teamID, userID uint64) (bool, error)
	getRoleFn    func(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error)
	addMemberFn  func(ctx context.Context, teamID, userID uint64, role models.TeamMemberRole) error
	listTeamsFn  func(ctx context.Context, userID uint64) ([]models.Team, error)
	createTeamFn func(ctx context.Context, name string, createdBy uint64) (uint64, error)
}

func (m *mockTeamRepo) IsMember(ctx context.Context, teamID, userID uint64) (bool, error) {
	if m.isMemberFn != nil {
		return m.isMemberFn(ctx, teamID, userID)
	}
	return false, nil
}
func (m *mockTeamRepo) GetUserRole(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error) {
	if m.getRoleFn != nil {
		return m.getRoleFn(ctx, teamID, userID)
	}
	return "", errors.New("no role")
}
func (m *mockTeamRepo) AddMember(ctx context.Context, teamID, userID uint64, role models.TeamMemberRole) error {
	if m.addMemberFn != nil {
		return m.addMemberFn(ctx, teamID, userID, role)
	}
	return nil
}
func (m *mockTeamRepo) ListTeamsByUser(ctx context.Context, userID uint64) ([]models.Team, error) {
	if m.listTeamsFn != nil {
		return m.listTeamsFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockTeamRepo) CreateTeamWithOwner(ctx context.Context, name string, createdBy uint64) (uint64, error) {
	if m.createTeamFn != nil {
		return m.createTeamFn(ctx, name, createdBy)
	}
	return 0, nil
}

type mockTaskRepo struct {
	createFn            func(ctx context.Context, t models.Task) (uint64, error)
	getFn               func(ctx context.Context, id uint64) (models.Task, error)
	listFn              func(ctx context.Context, f models.TaskFilter) ([]models.Task, error)
	updateWithHistoryFn func(ctx context.Context, taskID uint64, upd repo.TaskUpdate, changedBy uint64) (models.Task, error)
	historyFn           func(ctx context.Context, taskID uint64, limit, offset int) ([]models.TaskHistoryItem, error)
	addCommentFn        func(ctx context.Context, taskID, userID uint64, body string) (uint64, error)
	listCommentsFn      func(ctx context.Context, taskID uint64, limit, offset int) ([]models.TaskComment, error)

	listCalled bool
}

func (m *mockTaskRepo) Create(ctx context.Context, t models.Task) (uint64, error) {
	return m.createFn(ctx, t)
}
func (m *mockTaskRepo) Get(ctx context.Context, id uint64) (models.Task, error) {
	return m.getFn(ctx, id)
}
func (m *mockTaskRepo) List(ctx context.Context, f models.TaskFilter) ([]models.Task, error) {
	m.listCalled = true
	return m.listFn(ctx, f)
}
func (m *mockTaskRepo) UpdateWithHistory(ctx context.Context, taskID uint64, upd repo.TaskUpdate, changedBy uint64) (models.Task, error) {
	return m.updateWithHistoryFn(ctx, taskID, upd, changedBy)
}
func (m *mockTaskRepo) History(ctx context.Context, taskID uint64, limit, offset int) ([]models.TaskHistoryItem, error) {
	return m.historyFn(ctx, taskID, limit, offset)
}
func (m *mockTaskRepo) AddComment(ctx context.Context, taskID, userID uint64, body string) (uint64, error) {
	return m.addCommentFn(ctx, taskID, userID, body)
}
func (m *mockTaskRepo) ListComments(ctx context.Context, taskID uint64, limit, offset int) ([]models.TaskComment, error) {
	return m.listCommentsFn(ctx, taskID, limit, offset)
}

type mockCache struct {
	getVerFn  func(ctx context.Context, teamID uint64) (string, error)
	bumpFn    func(ctx context.Context, teamID uint64) error
	listKeyFn func(teamID uint64, ver string, f models.TaskFilter) string
	getFn     func(ctx context.Context, key string) ([]models.Task, bool, error)
	setFn     func(ctx context.Context, key string, items []models.Task) error

	bumped bool
	setKey string
}

func (m *mockCache) GetTeamVersion(ctx context.Context, teamID uint64) (string, error) {
	return m.getVerFn(ctx, teamID)
}
func (m *mockCache) BumpTeamVersion(ctx context.Context, teamID uint64) error {
	m.bumped = true
	if m.bumpFn != nil {
		return m.bumpFn(ctx, teamID)
	}
	return nil
}
func (m *mockCache) ListKey(teamID uint64, ver string, f models.TaskFilter) string {
	if m.listKeyFn != nil {
		return m.listKeyFn(teamID, ver, f)
	}
	return "k"
}
func (m *mockCache) GetTasks(ctx context.Context, key string) ([]models.Task, bool, error) {
	return m.getFn(ctx, key)
}
func (m *mockCache) SetTasks(ctx context.Context, key string, items []models.Task) error {
	m.setKey = key
	if m.setFn != nil {
		return m.setFn(ctx, key, items)
	}
	return nil
}

func TestTaskUsecase_Create_Success(t *testing.T) {
	team := &mockTeamRepo{
		isMemberFn: func(ctx context.Context, teamID, userID uint64) (bool, error) { return true, nil },
	}
	task := &mockTaskRepo{
		createFn: func(ctx context.Context, t models.Task) (uint64, error) { return 42, nil },
	}
	cache := &mockCache{
		getVerFn: func(ctx context.Context, teamID uint64) (string, error) { return "1", nil },
	}
	uc := NewTaskUsecase(task, team, cache)
	id, err := uc.Create(context.Background(), CreateTaskInput{
		UserID: 1, TeamID: 1, Title: "Title", Status: models.StatusToDo,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(42), id)
	require.True(t, cache.bumped)
}

func TestTaskUsecase_Create_InvalidStatus(t *testing.T) {
	uc := NewTaskUsecase(&mockTaskRepo{}, &mockTeamRepo{}, &mockCache{})
	_, err := uc.Create(context.Background(), CreateTaskInput{
		UserID: 1, TeamID: 1, Title: "t", Status: "bad",
	})
	require.ErrorIs(t, err, repo.ErrConflict)
}

func TestTaskUsecase_List_CacheHit(t *testing.T) {
	items := []models.Task{{ID: 1, TeamID: 1, Title: "t"}}
	task := &mockTaskRepo{
		listFn: func(ctx context.Context, f models.TaskFilter) ([]models.Task, error) { return nil, nil },
	}
	cache := &mockCache{
		getVerFn: func(ctx context.Context, teamID uint64) (string, error) { return "1", nil },
		getFn:    func(ctx context.Context, key string) ([]models.Task, bool, error) { return items, true, nil },
	}
	team := &mockTeamRepo{
		isMemberFn: func(ctx context.Context, teamID, userID uint64) (bool, error) { return true, nil },
	}
	uc := NewTaskUsecase(task, team, cache)

	out, err := uc.List(context.Background(), ListTaskInput{UserID: 1, TeamID: 1})
	require.NoError(t, err)
	require.Equal(t, items, out)
	require.False(t, task.listCalled, "repo.List should not be called on cache hit")
}

func TestTaskUsecase_List_CacheMiss(t *testing.T) {
	items := []models.Task{{ID: 2, TeamID: 1, Title: "x"}}
	task := &mockTaskRepo{
		listFn: func(ctx context.Context, f models.TaskFilter) ([]models.Task, error) { return items, nil },
	}
	cache := &mockCache{
		getVerFn: func(ctx context.Context, teamID uint64) (string, error) { return "1", nil },
		getFn:    func(ctx context.Context, key string) ([]models.Task, bool, error) { return nil, false, nil },
		setFn:    func(ctx context.Context, key string, items []models.Task) error { return nil },
	}
	team := &mockTeamRepo{
		isMemberFn: func(ctx context.Context, teamID, userID uint64) (bool, error) { return true, nil },
	}
	uc := NewTaskUsecase(task, team, cache)

	out, err := uc.List(context.Background(), ListTaskInput{UserID: 1, TeamID: 1})
	require.NoError(t, err)
	require.Equal(t, items, out)
	require.True(t, task.listCalled)
	require.NotEmpty(t, cache.setKey)
}

func TestTaskUsecase_Update_Permissions(t *testing.T) {
	cur := models.Task{ID: 1, TeamID: 1, Title: "old", Status: models.StatusToDo, CreatedBy: 1}
	task := &mockTaskRepo{
		getFn: func(ctx context.Context, id uint64) (models.Task, error) { return cur, nil },
		updateWithHistoryFn: func(ctx context.Context, taskID uint64, upd repo.TaskUpdate, changedBy uint64) (models.Task, error) {
			return models.Task{ID: 1, TeamID: 1, Title: "new", Status: models.StatusDone, CreatedBy: 1}, nil
		},
	}
	team := &mockTeamRepo{
		getRoleFn: func(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error) {
			return models.RoleMember, nil
		},
	}
	cache := &mockCache{}
	uc := NewTaskUsecase(task, team, cache)

	newTitle := "new"
	newStatus := models.StatusDone
	out, err := uc.Update(context.Background(), UpdateTaskInput{
		UserID: 1, TaskID: 1,
		Title: &newTitle, Status: &newStatus,
	})
	require.NoError(t, err)
	require.Equal(t, "new", out.Title)
}

func TestTaskUsecase_Update_Forbidden(t *testing.T) {
	cur := models.Task{ID: 1, TeamID: 1, Title: "old", Status: models.StatusToDo, CreatedBy: 2}
	task := &mockTaskRepo{
		getFn: func(ctx context.Context, id uint64) (models.Task, error) { return cur, nil },
	}
	team := &mockTeamRepo{
		getRoleFn: func(ctx context.Context, teamID, userID uint64) (models.TeamMemberRole, error) {
			return models.RoleMember, nil
		},
	}
	uc := NewTaskUsecase(task, team, &mockCache{})
	_, err := uc.Update(context.Background(), UpdateTaskInput{UserID: 1, TaskID: 1})
	require.ErrorIs(t, err, repo.ErrConflict)
}

func TestTaskUsecase_History(t *testing.T) {
	cur := models.Task{ID: 1, TeamID: 1}
	task := &mockTaskRepo{
		getFn: func(ctx context.Context, id uint64) (models.Task, error) { return cur, nil },
		historyFn: func(ctx context.Context, taskID uint64, limit, offset int) ([]models.TaskHistoryItem, error) {
			return []models.TaskHistoryItem{{ID: 1, TaskID: 1, FieldName: "title"}}, nil
		},
	}
	team := &mockTeamRepo{
		isMemberFn: func(ctx context.Context, teamID, userID uint64) (bool, error) { return true, nil },
	}
	uc := NewTaskUsecase(task, team, &mockCache{})
	items, err := uc.History(context.Background(), 1, 1, 10, 0)
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestTaskUsecase_AddComment_Validation(t *testing.T) {
	uc := NewTaskUsecase(&mockTaskRepo{}, &mockTeamRepo{}, &mockCache{})
	_, err := uc.AddComment(context.Background(), 1, 1, "")
	require.ErrorIs(t, err, repo.ErrConflict)
}

func TestTaskUsecase_AddComment_Success(t *testing.T) {
	cur := models.Task{ID: 1, TeamID: 1}
	task := &mockTaskRepo{
		getFn: func(ctx context.Context, id uint64) (models.Task, error) { return cur, nil },
		addCommentFn: func(ctx context.Context, taskID, userID uint64, body string) (uint64, error) {
			return 55, nil
		},
	}
	team := &mockTeamRepo{
		isMemberFn: func(ctx context.Context, teamID, userID uint64) (bool, error) { return true, nil },
	}
	uc := NewTaskUsecase(task, team, &mockCache{})
	id, err := uc.AddComment(context.Background(), 1, 1, "hello")
	require.NoError(t, err)
	require.Equal(t, uint64(55), id)
}

func TestTaskUsecase_ListComments(t *testing.T) {
	cur := models.Task{ID: 1, TeamID: 1}
	task := &mockTaskRepo{
		getFn: func(ctx context.Context, id uint64) (models.Task, error) { return cur, nil },
		listCommentsFn: func(ctx context.Context, taskID uint64, limit, offset int) ([]models.TaskComment, error) {
			return []models.TaskComment{{ID: 1, TaskID: 1, UserID: 2, Body: "ok"}}, nil
		},
	}
	team := &mockTeamRepo{
		isMemberFn: func(ctx context.Context, teamID, userID uint64) (bool, error) { return true, nil },
	}
	uc := NewTaskUsecase(task, team, &mockCache{})
	items, err := uc.ListComments(context.Background(), 1, 1, 10, 0)
	require.NoError(t, err)
	require.Len(t, items, 1)
}
