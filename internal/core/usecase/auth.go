package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/DrummDaddy/task_service/internal/core/ports"
	"github.com/DrummDaddy/task_service/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

type AuthConfig struct {
	PasswordPepper   string
	PasswordHashCost int
}

type AuthUsecase struct {
	cfg        AuthConfig
	users      ports.UserRepository
	tokenIssue ports.TokenIssuer
}

func NewAuthUsecase(cfg AuthConfig, users ports.UserRepository, issuer ports.TokenIssuer) *AuthUsecase {
	return &AuthUsecase{cfg: cfg, users: users, tokenIssue: issuer}
}

func (u *AuthUsecase) Register(ctx context.Context, email, password string) (uint64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(password) < 8 {
		return 0, fmt.Errorf("email or password is too short")
	}
	passBytes := []byte(password + u.cfg.PasswordPepper)
	hash, err := bcrypt.GenerateFromPassword(passBytes, u.cfg.PasswordHashCost)
	if err != nil {
		return 0, err
	}
	return u.users.Create(ctx, email, hash)
}

func (u *AuthUsecase) Login(ctx context.Context, email, password string) (uint64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return 0, fmt.Errorf("email or password is too short")
	}
	usr, err := u.users.GetByEmail(ctx, email)
	if err != nil {
		return 0, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(usr.PasswordHash), []byte(password+u.cfg.PasswordPepper)); err != nil {
		return 0, repo.ErrNotFound
	}
	return usr.ID, nil
}
