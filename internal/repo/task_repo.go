package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

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

type TaskRepo struct {
	db *sql.DB
}

func NewTaskRepo(db *sql.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(ctx context.Context, t Task) (uint64, error) {
	res, err := r.db.ExecContext(ctx, `
INSERT INTO tasks(team_id, title, description, status, assignee_id, created_by)
VALUES(?, ?, ?, ?, ?, ?)
`, t.TeamID, t.Title, t.Description, t.Status, t.AssigneeID, t.CreatedBy)
	if err != nil {
		return 0, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("task last insert id: %w", err)
	}
	return uint64(id), nil

}

func (r *TaskRepo) Get(ctx context.Context, id uint64) (Task, error) {
	var t Task
	var desc sql.NullString
	var assignee sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at 
 FROM tasks 
 WHERE id = ? 
 `, id).Scan(&t.ID, &t.TeamID, &t.Title, &desc, &t.Status, &assignee, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, fmt.Errorf("get task: %w", err)

	}
	if desc.Valid {
		t.Description = &desc.String
	}
	if assignee.Valid {
		v := uint64(assignee.Int64)
		t.AssigneeID = &v
	}
	return t, nil

}
func (r *TaskRepo) List(ctx context.Context, f *TaskFilter) ([]Task, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	query := `
SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at 
FROM tasks
WHERE status = ?
`
	args := []any{f.TeamID}
	if f.Status != nil {
		query += " AND status = ?"
		args = append(args, *f.Status)
	}
	if f.AssigneeID != nil {
		query += " AND assignee_id = ?"
		args = append(args, *f.AssigneeID)
	}
	query += "ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, f.Limit, f.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		var desc sql.NullString
		var assignee sql.NullInt64
		if err := rows.Scan(&t.ID, &t.TeamID, &t.Title, &desc, &t.Status, &assignee, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("tasks scan: %w", err)
		}
		if desc.Valid {
			t.Description = &desc.String
		}
		if assignee.Valid {
			v := uint64(assignee.Int64)
			t.AssigneeID = &v
		}
		out = append(out, t)

	}
	return out, rows.Err()
}

type TaskUpdate struct {
	Title       *string
	Description **string
	Status      *TaskStatus
	AssigneeID  **uint64
}

func (r *TaskRepo) UpdateWithHistory(ctx context.Context, taskID uint64, upd TaskUpdate, changedBy uint64) (Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Task{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var cur Task
	var curDesc sql.NullString
	var curAssignee sql.NullInt64
	err = tx.QueryRowContext(ctx, `
SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at 
FROM tasks
WHERE id = ?
FOR UPDATE
`, taskID).Scan(&cur.ID, &cur.TeamID, &cur.Title, &curDesc, &cur.Status, &curAssignee, &cur.CreatedBy, &cur.CreatedAt, &cur.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, fmt.Errorf("task lock: %w", err)
	}
	if curDesc.Valid {
		cur.Description = &curDesc.String
	}
	if curAssignee.Valid {
		v := uint64(curAssignee.Int64)
		cur.AssigneeID = &v
	}

	setClauses := ""
	args := make([]any, 0, 6)
	addSet := func(clause string, val any) {
		if setClauses != "" {
			setClauses += ","
		}
		setClauses += clause
		args = append(args, val)
	}
	history := make([]TaskHistoryItem, 0, 4)
	addHistory := func(field string, oldV, newV *string) {
		history = append(history, TaskHistoryItem{
			TaskID:    taskID,
			ChangedBy: changedBy,
			FieldName: field,
			OldValue:  oldV,
			NewValue:  newV,
		})
	}
	if upd.Title != nil && *upd.Title != cur.Title {
		oldV := cur.Title
		newV := *upd.Title
		addSet("title = ?", &newV)
		addHistory("title", &oldV, &newV)
	}
	if upd.Description != nil {
		var oldV *string = cur.Description
		var newV *string = *upd.Description
		changed := false
		switch {
		case oldV != nil && newV != nil:
			changed = true
		case oldV != nil && newV == nil:
			changed = true
		case oldV != nil && newV != nil && *oldV != *newV:
			changed = true

		}
		if changed {
			addSet("description = ?", &newV)
			addHistory("description", oldV, newV)
		}
	}
	if upd.Status != nil && *upd.Status != cur.Status {
		oldS := string(*upd.Status)
		newS := string(*upd.Status)
		addSet("status = ?", *upd.Status)
		addHistory("status", &oldS, &newS)
	}
	if upd.AssigneeID != nil {
		var oldV *string
		if cur.AssigneeID != nil {
			s := fmt.Sprintf("%d", *cur.AssigneeID)
			oldV = &s
		}
		var newV *string
		if *upd.AssigneeID != nil {
			s := fmt.Sprintf("%d", **upd.AssigneeID)
			newV = &s
		}
		changed := false
		switch {
		case cur.AssigneeID == nil && *upd.AssigneeID != nil:
			changed = true
		case cur.AssigneeID != nil && *upd.AssigneeID == nil:
			changed = true
		case cur.AssigneeID != nil && *upd.AssigneeID != nil && *cur.AssigneeID != **upd.AssigneeID:
			changed = true

		}
		if changed {
			addSet("assignee_id = ?", *upd.AssigneeID)
			addHistory("assignee_id", oldV, newV)
		}
	}
	if setClauses != "" {
		args = append(args, "set_clauses = ?", setClauses)
		if _, err := tx.ExecContext(ctx, "UPDATE tasks SET"+setClauses+"WHERE id =?", args...); err != nil {
			return Task{}, fmt.Errorf("task update: %w", err)
		}
		for _, h := range history {
			_, err := tx.ExecContext(ctx, `
INSERT INTO task_history(task_id, changed_by, field_name, old_value, new_value) 
VALUES (?, ?, ?, ?, ?)
`, h.TaskID, h.ChangedBy, h.FieldName, h.OldValue, h.NewValue)
			if err != nil {
				return Task{}, fmt.Errorf("task_history insert: %w", err)
			}
		}
	}

	var out Task
	var outDesc sql.NullString
	var outAssignee sql.NullInt64
	err = tx.QueryRowContext(ctx, `
SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at 
FROM tasks 
WHERE id = ? 
`, taskID).Scan(&out.ID, &out.TeamID, &out.Title, &outDesc, &out.Status, &outAssignee, &out.CreatedBy, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return Task{}, fmt.Errorf("task selected updated: %w", err)
	}
	if outDesc.Valid {
		out.Description = &outDesc.String
	}
	if outAssignee.Valid {
		v := uint64(outAssignee.Int64)
		out.AssigneeID = &v
	}

	if err := tx.Commit(); err != nil {
		return Task{}, fmt.Errorf("commit tx: %w", err)

	}
	return out, nil

}

func (r *TaskRepo) History(ctx context.Context, taskID uint64, limit int, offset int) ([]TaskHistoryItem, error) {
	if limit < 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, task_id, changed_by, field_name, old_value, new_value, changed_at 
FROM tasks_history
WHERE task_id = ?
ORDER BY id DESC 
LIMIT ? OFFSET ?
`, taskID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("task_history list: %w", err)
	}
	defer rows.Close()
	var out []TaskHistoryItem
	for rows.Next() {
		var h TaskHistoryItem
		var oldV, newV sql.NullString
		if err := rows.Scan(&h.ID, &h.TaskID, &h.ChangedBy, &h.FieldName, &oldV, &newV, &h.ChangedAt); err != nil {
			return nil, fmt.Errorf("task_history scan: %w", err)
		}
		if oldV.Valid {
			h.OldValue = &oldV.String
		}
		if newV.Valid {
			h.NewValue = &newV.String
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *TaskRepo) AddComment(ctx context.Context, taskID, userID uint64, body string) (uint64, error) {
	res, err := r.db.ExecContext(ctx, `
INSERT INTO task_comment(task_id, user_id, body) VALUES (?, ?, ?)`, taskID, userID, body)
	if err != nil {
		return 0, fmt.Errorf("add comment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("add comment: %w", err)

	}
	return uint64(id), nil
}

func (r *TaskRepo) ListComments(ctx context.Context, taskID uint64, limit int, offset int) ([]TaskComment, error) {
	if limit < 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, task_id, user_id, body, created_at 
FROM tasks_comments
WHERE task_id = ?
ORDER BY id ASC 
LIMIT ? OFFSET ?
`, taskID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("task_comments list: %w", err)
	}
	defer rows.Close()
	var out []TaskComment
	for rows.Next() {
		var c TaskComment
		if rows.Scan(&c.ID, &c.TaskID, &c.UserID, &c.Body, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("task_comments scan: %w", err)
		}

		out = append(out, c)
	}
	return out, rows.Err()
}
