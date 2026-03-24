package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/DrummDaddy/task_service/internal/core/ports"
	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/stretchr/testify/require"
)

type mockReportsRepo struct {
	teamStatsFn       func(ctx context.Context, now time.Time) ([]models.TeamStatsRow, error)
	topCreatorsFn     func(ctx context.Context) ([]models.TopCreatorRow, error)
	invalidAssigneeFn func(ctx context.Context, limit, offset int) ([]models.IntegrityTaskRow, error)
}

func (m *mockReportsRepo) TeamStats(ctx context.Context, now time.Time) ([]models.TeamStatsRow, error) {
	return m.teamStatsFn(ctx, now)
}
func (m *mockReportsRepo) TopCreatorsLastMonth(ctx context.Context) ([]models.TopCreatorRow, error) {
	return m.topCreatorsFn(ctx)
}
func (m *mockReportsRepo) TaskWithInvalidAssignee(ctx context.Context, limit, offset int) ([]models.IntegrityTaskRow, error) {
	return m.invalidAssigneeFn(ctx, limit, offset)
}

func TestReportUsecase_Forwarding(t *testing.T) {
	repo := &mockReportsRepo{
		teamStatsFn: func(ctx context.Context, now time.Time) ([]models.TeamStatsRow, error) {
			return []models.TeamStatsRow{{TeamID: 1, TeamName: "team", MembersCount: 3, DoneTaskLast7d: 2}}, nil
		},
		topCreatorsFn: func(ctx context.Context) ([]models.TopCreatorRow, error) {
			return []models.TopCreatorRow{{TeamID: 1, UserID: 2, TaskCreated: 5, Rank: 1}}, nil
		},
		invalidAssigneeFn: func(ctx context.Context, limit, offset int) ([]models.IntegrityTaskRow, error) {
			return []models.IntegrityTaskRow{{TaskID: 9, TeamID: 1, AssigneeID: 7}}, nil
		},
	}
	uc := NewReportUsecase(repo)

	stats, err := uc.TeamStats(context.Background(), time.Now())
	require.NoError(t, err)
	require.Len(t, stats, 1)

	top, err := uc.TopCreatorsLastMonth(context.Background())
	require.NoError(t, err)
	require.Len(t, top, 1)

	bad, err := uc.TaskWithInvalidAssignee(context.Background(), 50, 0)
	require.NoError(t, err)
	require.Len(t, bad, 1)
	_ = ports.TokenIssuer(nil) // avoid unused import warning if any
}
