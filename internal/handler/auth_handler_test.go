package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/DrummDaddy/task_service/internal/config"
	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthHandler_Register(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {_= db.Close()})

	mock.ExpectExec("INSERT INTO users").
		WithArgs("u@example.com", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(10, 1))

	cfg := config.Config{}
	cfg.Auth.JWTSecret = "secret"
	cfg.Auth.PasswordHashCost = 4

	h := NewAuthHandler(cfg, repo.NewUserRepo(db))

	body, _ := json.Marshal(map[string]any{"email": "u@example.com", "password": "password123"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Register(w, r)
	require.Equal(t, http.StatusCreated, w.Code)
	require.NoError(t, mock.ExpectationsWereMet())

}

func TestAuthHandler_Login(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {_= db.Close()})

	cfg := config.Config{}
	cfg.Auth.JWTSecret = "secret"
	cfg.Auth.AccessTokenTTL = 24 * time.Hour
	cfg.Auth.PasswordPepper = ""

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), 4)
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash"}).AddRow(uint64(7), "u@example.com", hash)

	mock.ExpectQuery("SELECT id, email, password_hash FROM users WHERE email = ?").WithArgs("u@example.com").WillReturnRows(rows).
		WithArgs("u@example.com").WillReturnRows(rows)

	h := NewAuthHandler(cfg, repo.NewUserRepo(db))

	body, _ := json.Marshal(map[string]any{"email":"u@example", "password": "password123"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, r)
	require.Equal(t, http.StatusCreated, w.Code
	require.Contains(t, w.Body.String(),"access_token")
	require.NoError(t, mock.ExpectationsWereMet())
}
