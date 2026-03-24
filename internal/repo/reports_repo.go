package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/DrummDaddy/task_service/internal/models"
)

type ReportsRepo struct {
	db *sql.DB
}

func NewReportsRepo(db *sql.DB) *ReportsRepo { return &ReportsRepo{db: db} }

func (r *ReportsRepo) TeamStats(ctx context.Context, now time.Time) ([]models.TeamStatsRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT
  t.id AS team_id,
  t.name AS team_name,
  COUNT(DISTINCT tm.user_id) AS members_count,
  COALESCE(SUM(CASE
    WHEN tk.status = 'done' AND tk.updated_at >= (UTC_TIMESTAMP(3) - INTERVAL 7 DAY)
    THEN 1 ELSE 0 END), 0) AS done_tasks_last_7d
FROM teams t
LEFT JOIN team_members tm ON tm.team_id = t.id
LEFT JOIN tasks tk ON tk.team_id = t.id
GROUP BY t.id, t.name
ORDER BY t.id DESC`)
	_ = now
	if err != nil {
		return nil, fmt.Errorf("team stats: %w", err)
	}
	defer rows.Close()
	var out []models.TeamStatsRow
	for rows.Next() {
		var r models.TeamStatsRow
		if err := rows.Scan(&r.TeamID, &r.TeamName, &r.MembersCount, &r.DoneTaskLast7d); err != nil {
			return nil, fmt.Errorf("team stats: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (r *ReportsRepo) TopCreatorsLastMonth(ctx context.Context) ([]models.TopCreatorRow, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH per_user AS (
  SELECT
    t.team_id,
    t.created_by AS user_id,
    COUNT(*) AS tasks_created
  FROM tasks t
  WHERE t.created_at >= (UTC_TIMESTAMP(3) - INTERVAL 1 MONTH)
  GROUP BY t.team_id, t.created_by
),
ranked AS (
  SELECT
    team_id,
    user_id,
    tasks_created,
    ROW_NUMBER() OVER (PARTITION BY team_id ORDER BY tasks_created DESC, user_id ASC) AS rn
  FROM per_user
)
SELECT team_id, user_id, tasks_created, rn
FROM ranked
WHERE rn <= 3
ORDER BY team_id DESC, rn ASC
    `)
	if err != nil {
		return nil, fmt.Errorf("top creators: %w", err)
	}
	defer rows.Close()
	var out []models.TopCreatorRow
	for rows.Next() {
		var r models.TopCreatorRow
		if err := rows.Scan(&r.TeamID, &r.UserID, &r.TaskCreated, &r.Rank); err != nil {
			return nil, fmt.Errorf("top creators: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (r *ReportsRepo) TaskWithInvalidAssignee(ctx context.Context, limit, offset int) ([]models.IntegrityTaskRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset <= 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT tk.id, tk.team_id, tk.assignee_id
FROM tasks tk
LEFT JOIN team_members tm
  ON tm.team_id = tk.team_id AND tm.user_id = tk.assignee_id
WHERE tk.assignee_id IS NOT NULL
  AND tm.user_id IS NULL
ORDER BY tk.id DESC
LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("tasks: %w", err)
	}
	defer rows.Close()
	var out []models.IntegrityTaskRow
	for rows.Next() {
		var rec models.IntegrityTaskRow
		if err := rows.Scan(&rec.TaskID, &rec.TeamID, &rec.AssigneeID); err != nil {
			return nil, fmt.Errorf("tasks: %w", err)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
