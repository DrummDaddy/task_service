package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type User struct {
	ID           uint64
	Email        string
	PasswordHash string
}

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(ctx context.Context, email string, passwordHash []byte) (uint64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users(email, password_hash) VALUES (?, ?)`, email, passwordHash)
	if err != nil {
		if isMysqlError(err) {
			return 0, ErrConflict
		}
		return 0, fmt.Errorf("users insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("users last insert id: %w", err)
	}
	return uint64(id), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.db.QueryRowContext(ctx, `
SELECT id, email, password_hash
FROM users WHERE email = ?`, email).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("users get: %w", err)
	}
	return u, nil
}
