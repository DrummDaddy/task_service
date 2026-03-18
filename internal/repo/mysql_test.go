package repo

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)
func withMySQL(t *testing.T, fn func(db *sql.DB)) {
	t.Helper()
	ctx := context.Background()

	if runtime.GOOS == "windows" {
		if dh := os.Getenv("DCOKER_HOST"); dh != "" && (strings.Contains(dh, "dockerRootlessEngine") || strings.Contains(dh, "rootless")) {
			t.Skipf("skipping integration test: unsupported DOCKER_HOST=%q on windows", dh)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			t.Skipf("skipping integration test: docker/testcontainers unvailible (%v)", r)
		}
	}()

	container, err := mysql.Run(ctx,
		"mysql:8.4",
		mysql.WithDatabase("task_service"),
		mysql.WithUsername("root"),
		mysql.WithPassword("rootpass"),
		)
	if err != nil {
		t.Skipf("skipping integration test: docker/testcontainers unvailible (%v)", err)
	}
	t.Cleanup(func() {_ = container.Terminate(ctx)})
	dsn, err := container.ConnectionString(ctx, "parseTime=true&multiStatements=true")
	require.NoError(t, err)

	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, db.Ping())

	migPath := filepath.Join("..", "..", "migrations", "001_init.sql")
	raw, err := os.ReadFile(migPath)
	require.NoError(t, err)
	_, err = db.Exec(string(raw))
	require.NoError(t, err)
	fn(db)
}

func TestReportsQueries(t *testing.T){
	withMySQL(t, func(db *sql.DB) {
		ctx := context.Background()

		_, err := db.ExecContext(ctx, `INSERT INTO users(email, password_hash) VALUES 
                                            ('u1@example.com', 'x' ), 
                                            ('u2@example.com', 'x' ), 
                                            ('u3@example.com', 'x' ), 
                                            ('u4@example.com', 'x' )
                                             `)
		require.NoError(t, err)

		res, err := db.ExecContext(ctx, `INSERT INTO teams(name, created_by) VALUES('team1', 1)`)
		require.NoError(t, err)
		teamID64, _ := res.LastInsertId()
		teamID := uint(teamID64)

		_, err = db.ExecContext(ctx, `INSERT INTO team_members(team_id, user_id, role)VALUES 
                                                    (?, 1, 'owner'), 
													(?, 2, 'member'), 
													(?, 3, 'member')
`, teamID, teamID, teamID
		require.NoError(t, err)

		now := time.Now().UTC()
		lastWeek := now.Add(-2*24 * time.Hour)
		lastMonth := now.Add(-10 * 24 * time.Hour)

		_, err = db.ExecContext(ctx, `INSERT INTO tasks(team_id, title, description, status, assignee_id, created_by, created_at, updated_at) 
VALUES
    (?, 't1', NULL, 'done', 2,1,?,?), 
    (?, 't2', NULL, 'done', 2,1,?,?), 
    (?, 't3', NULL, 'done', 2,1,?,?), 
    (?, 't4', NULL, 'todo', 2,1,?,?), 
    (?, 't5', NULL, 'todo', 2,2,?,?), 
    (?, 't6', NULL, 'todo', 2,2,?,?), 
    (?, 't7', NULL, 'todo', 2,2,?,?), 
    (?, 't8', NULL, 'todo', 2,3,?,?), 
    (?, 't9', NULL, 'todo', 2,3,?,?), 
    (?, 't10', NULL, 'todo', 4,4,?,?)
    `,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	teamID, lastMonth, lastWeek,
	)
		require.NoError(t, err)
		reports := NewReportsRepo

		stats, err := reports.TeamStats(ctx, now)
		require.NoError(t, err)
		require.Len(t, stats, 1)
		require.Equal(t, "team1", stats[0].TeamName)
		require.Equal(t, 3, stats[0].MembersCount)
		require.Equal(t, 3, stats[0].DoneTaskLast7d)

		top, err := reports.TopCreatorsLastMonth(ctx)
		require.NoError(t, err)
		require.Len(t, top, 3)
		require.Equal(t, teamID, top[0].TeamID)
		require.Equal(t, uint64(1), top[0].UserID)
		require.Equal(t, 4, top[0].TaskCreated)
		require.Equal(t, 1, top[0].Rank)
		require.Equal(t, uint64(2), top[1].UserID)
		require.Equal(t, 3, top[1].TaskCreated)
		require.Equal(t,2, top[1].Rank)
		require.Equal(t, uint64(3), top[2].UserID)
		require.Equal(t,2, top[2].TaskCreated)
		require.Equal(t,3, top[2].Rank)

		invalid, err := reports.TasksWithInvalidAssignee(ctx, 50, 0)

		require.NoError(t, err)
		require.Len(t, invalid, 1)
		require.Equal(t, uint64(4), invalid[0].AssigneeID)
	})
}

func TestUpdateWithHistoryWriteAudit(t *testing.T) {
	withMySQL(t, func(db *sql.DB) {
		ctx := context.Background()
		_, err := db.ExecContext(ctx, `INSERT INTO users(email, password_hash) VALUES('u1@example.com', 'x')`)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO teams(name, created_by) VALUES('team1', 1)`)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO team_members(team_id, user_id, role)VALUES(1,1, 'owner')`)
		require.NoError(t, err)

		taskRepo := NewTaskRepo(db)
		taskID, err := taskRepo.Create(ctx, Task{
			TeamID: 1,
			Title:  "old",
			Status: StatusToDo,
			CreatedBy: 1,
		})
		require.NoError(t, err)

		newTitle := "new"
		newStatus := StatusDone
		descVal := "desc"
		descPtr := &descVal
		updated, err := taskRepo.UpdateWithHistory(ctx, taskID, TaskUpdate{
			Title: &newTitle,
			Status: &newStatus,
			Description: &descPtr,
		}, 1)
		require.NoError(t, err)
		require.Equal(t, "new", updated.Title)
		require.Equal(t, StatusDone, updated.Status)
		require.NotNil(t, updated.Description)
		require.Equal(t, "desc", *updated.Description)

		h, err := taskRepo.History(ctx, taskID, 10, 0)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(h), 3)

	})
}