package cache

import (
	"testing"
	"time"

	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/stretchr/testify/require"
)

func TestTasksCache(t *testing.T) {
	c := NewTaskCache(nil, 5*time.Minute)
	status := repo.StatusDone
	teamID := uint64(1)
	assignee := uint64(5)
	key := c.ListKey(1, "7", repo.TaskFilter{
		TeamID:     &teamID,
		Status:     &status,
		AssigneeID: &assignee,
		Limit:      20,
		Offset:     40,
	})
	require.Equal(t, "tasks:1:v:sdone:a5:l20;o40", key)
}
