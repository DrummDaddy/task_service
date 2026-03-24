package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/DrummDaddy/task_service/internal/models"
)

type TeamRepo struct {
	db *sql.DB
}

func NewTeamRepo(db *sql.DB) *TeamRepo { return &TeamRepo{db: db} }

func (r *TeamRepo) CreateTeamOwner(ctx context.Context, name string, createdBy uint64) (uint64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("tx begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `INSERT INTO teams(name, created_by) VALUES (?, ?)`, name, createdBy)
	if err != nil {
		return 0, fmt.Errorf("teams insert: %w", err)
	}
	teamID64, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("teams last insert id: %w", err)
	}
	teamID := uint64(teamID64)

	_, err = tx.ExecContext(ctx, `INSERT INTO team_members(team_id, user_id, role) VALUES (?, ?, 'owner')`, teamID, createdBy)
	if err != nil {
		return 0, fmt.Errorf("teams members insert owner: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("tx commit: %w", err)
	}
	return teamID, nil
}

func (r *TeamRepo) ListTeamsByUser(ctx context.Context, userID uint64) ([]models.Team, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT t.id, t.name, t.created_by
FROM teams t
JOIN team_members tm ON tm.team_id = t.id
WHERE tm.user_id = ?
ORDER BY t.id DESC `, userID)
	if err != nil {
		return nil, fmt.Errorf("teams list: %w", err)
	}
	defer rows.Close()
	var out []models.Team
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedBy); err != nil {
			return nil, fmt.Errorf("teams scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TeamRepo) GetUserRole(ctx context.Context, teamID uint64, userID uint64) (models.TeamMemberRole, error) {
	var role models.TeamMemberRole
	err := r.db.QueryRowContext(ctx,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
	}
	return role, nil
}

func (r *TeamRepo) IsMember(ctx context.Context, teamID uint64, userID uint64) (bool, error) {
	var x int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID, userID).Scan(&x)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("team member id member: %w", err)

	}
	return true, nil
}

func (r *TeamRepo) AddMember(ctx context.Context, teamID uint64, userID uint64, role models.TeamMemberRole) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO team_members(team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID, userID, role)
	if err != nil {
		if isMysqlError(err) {
			return ErrConflict
		}
		return fmt.Errorf("team_member insert: %w", err)

	}
	return nil
}
