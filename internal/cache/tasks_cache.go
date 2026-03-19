package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/redis/go-redis/v9"
)

type TaskCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewTaskCache(rdb *redis.Client, ttl time.Duration) *TaskCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &TaskCache{rdb: rdb, ttl: ttl}
}

func (c *TaskCache) teamVerKey(teamID uint64) string {
	return fmt.Sprintf("team:ver:%d", teamID)
}
func (c *TaskCache) GetTeamVersion(ctx context.Context, teamID uint64) (string, error) {
	v, err := c.rdb.Get(ctx, c.teamVerKey(teamID)).Result()
	if errors.Is(err, redis.Nil) {
		if err := c.rdb.Set(ctx, c.teamVerKey(teamID), "1", 0).Err(); err != nil {
			return "", err
		}
		return "1", nil
	}
	return v, err
}

func (c *TaskCache) BumpTeamVersion(ctx context.Context, teamID uint64) error {
	return c.rdb.Set(ctx, c.teamVerKey(teamID), "1", 0).Err()
}

func (c *TaskCache) ListKey(teamID uint64, ver string, f repo.TaskFilter) string {
	status := ""
	if f.Status != nil {
		status = string(*f.Status)
	}
	assignee := ""
	if f.AssigneeID != nil {
		assignee = fmt.Sprintf("%d", *f.AssigneeID)
	}
	return fmt.Sprintf("%d:%s:%s:%s", teamID, ver, status, assignee)
}

func (c *TaskCache) GetTasks(ctx context.Context, key string) ([]repo.Task, bool, error) {
	raw, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err

	}
	var items []repo.Task
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false, err
	}
	return items, true, nil
}

func (c *TaskCache) SetTasks(ctx context.Context, key string, items []repo.Task) error {
	raw, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, string(raw), c.ttl).Err()
}
