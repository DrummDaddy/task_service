package handler

import (
	"net/http"
	"strings"

	"github.com/DrummDaddy/task_service/internal/core/usecase"
	"github.com/DrummDaddy/task_service/internal/httpx"
	"github.com/DrummDaddy/task_service/internal/repo"
)

type AuthHandler struct {
	uc *usecase.AuthUsecase
}

func NewAuthHandler(uc *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{uc: uc}
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

	id, err := h.uc.Register(r.Context(), req.Email, req.Password)
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

	token, err := h.uc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid token")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
	})
}
