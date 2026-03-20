package handler

import (
	"net/http"
	"strings"

	"github.com/DrummDaddy/task_service/internal/auth"
	"github.com/DrummDaddy/task_service/internal/config"
	"github.com/DrummDaddy/task_service/internal/httpx"
	"github.com/DrummDaddy/task_service/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	cfg      config.Config
	userRepo *repo.UserRepo
}

func NewAuthHandler(cfg config.Config, userRepo *repo.UserRepo) *AuthHandler {
	return &AuthHandler{cfg: cfg, userRepo: userRepo}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if er := httpx.DecodeJSON(r, &req); er != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json format")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		httpx.Error(w, http.StatusBadRequest, "email or password is too short, min 8 chars")
		return
	}

	passBytes := []byte(req.Password + h.cfg.Auth.PasswordPepper)
	hash, err := bcrypt.GenerateFromPassword(passBytes, h.cfg.Auth.PasswordHashCost)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	id, err := h.userRepo.Create(r.Context(), req.Email, hash)
	if err != nil {
		if err == repo.ErrConflict {
			httpx.Error(w, http.StatusConflict, "user already exists")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id, "email": req.Email})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if er := httpx.DecodeJSON(r, &req); er != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json format")
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "email or password required")
		return
	}

	u, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(req.Password+h.cfg.Auth.PasswordPepper)); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid password")
		return
	}
	token, err := auth.IssueAccessToken([]byte(h.cfg.Auth.JWTSecret), u.ID, h.cfg.Auth.AccessTokenTTL)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
	})
}
