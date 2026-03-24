package usecase

import (
	"context"
	"testing"

	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo реализует ports.UserRepository
type mockUserRepo struct {
	createFn     func(ctx context.Context, email string, hash []byte) (uint64, error)
	getByEmailFn func(ctx context.Context, email string) (models.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, email string, passwordHash []byte) (uint64, error) {
	if m.createFn != nil {
		return m.createFn(ctx, email, passwordHash)
	}
	return 0, nil
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (models.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return models.User{}, nil
}

// mockTokenIssuer реализует ports.TokenIssuer
type mockTokenIssuer struct {
	issueFn func(userID uint64) (string, error)
}

func (m *mockTokenIssuer) Issue(userID uint64) (string, error) {
	if m.issueFn != nil {
		return m.issueFn(userID)
	}
	return "", nil
}

func TestAuthUsecase_Register_Success(t *testing.T) {
	var gotEmail string
	var gotHash []byte
	users := &mockUserRepo{
		createFn: func(ctx context.Context, email string, hash []byte) (uint64, error) {
			gotEmail = email
			gotHash = hash
			return 10, nil
		},
	}
	uc := NewAuthUsecase(AuthConfig{PasswordPepper: "pep", PasswordHashCost: 4}, users, &mockTokenIssuer{})

	id, err := uc.Register(context.Background(), "User@Example.Com ", "password123")
	require.NoError(t, err)
	require.Equal(t, uint64(10), id)
	require.Equal(t, "user@example.com", gotEmail)
	require.NotEmpty(t, gotHash)
}

func TestAuthUsecase_Register_Validation(t *testing.T) {
	uc := NewAuthUsecase(AuthConfig{PasswordPepper: "", PasswordHashCost: 4}, &mockUserRepo{}, &mockTokenIssuer{})
	_, err := uc.Register(context.Background(), "", "short")
	require.Error(t, err)
}

func TestAuthUsecase_Login_Success(t *testing.T) {
	pass := "password123"
	pep := "pep"
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass+pep), 4)

	users := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (models.User, error) {
			return models.User{ID: 7, Email: "u@example.com", PasswordHash: hash}, nil
		},
	}
	tokens := &mockTokenIssuer{
		issueFn: func(userID uint64) (string, error) {
			require.Equal(t, uint64(7), userID)
			return "token-abc", nil
		},
	}
	uc := NewAuthUsecase(AuthConfig{PasswordPepper: pep, PasswordHashCost: 4}, users, tokens)

	token, err := uc.Login(context.Background(), "u@example.com", pass)
	require.NoError(t, err)
	require.Equal(t, "token-abc", token)
}

func TestAuthUsecase_Login_InvalidPassword(t *testing.T) {
	pass := "password123"
	pep := "pep"
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass+pep), 4)

	users := &mockUserRepo{
		getByEmailFn: func(ctx context.Context, email string) (models.User, error) {
			return models.User{ID: 7, Email: "u@example.com", PasswordHash: hash}, nil
		},
	}
	uc := NewAuthUsecase(AuthConfig{PasswordPepper: pep, PasswordHashCost: 4}, users, &mockTokenIssuer{})

	_, err := uc.Login(context.Background(), "u@example.com", "wrong")
	require.ErrorIs(t, err, repo.ErrNotFound)
}
