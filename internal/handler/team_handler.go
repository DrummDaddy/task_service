package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/DrummDaddy/task_service/internal/auth"
	"github.com/DrummDaddy/task_service/internal/core/usecase"
	"github.com/DrummDaddy/task_service/internal/httpx"
	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/go-chi/chi/v5"
)

type TeamHandler struct {
	uc *usecase.TeamUseCase
}

func NewTeamHandler(uc *usecase.TeamUseCase) *TeamHandler {
	return &TeamHandler{uc: uc}
}

type createTeamRequest struct {
	Name string `json:"name"`
}

func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createTeamRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	id, err := h.uc.CreateTeam(r.Context(), userID, strings.TrimSpace(req.Name))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "create team failed")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id, "name": req.Name})
}

func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	items, err := h.uc.ListTeams(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "list teams failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

type inviteRequest struct {
	UserID uint64                `json:"user_id"`
	Role   models.TeamMemberRole `json:"role"`
}

func (h *TeamHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	teamID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || teamID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid team id")
		return
	}
	var req inviteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	emailSent, err := h.uc.Invite(r.Context(), userID, uint64(teamID64), req.UserID, req.Role)
	if err != nil {
		if err == repo.ErrConflict {
			httpx.Error(w, http.StatusConflict, "already a member or insufficient role")
			return
		}
		httpx.Error(w, http.StatusBadRequest, "invite failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "email_sent": emailSent})
}
