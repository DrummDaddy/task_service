package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/DrummDaddy/task_service/internal/auth"
	"github.com/DrummDaddy/task_service/internal/email"
	"github.com/DrummDaddy/task_service/internal/httpx"
	"github.com/DrummDaddy/task_service/internal/repo"
	"github.com/go-chi/chi/v5"
)

type TeamHandler struct {
	teamRepo *repo.TeamRepo
	email    *email.Client
}

func NewTeamHandler(teamRepo *repo.TeamRepo, emailClient *email.Client) *TeamHandler {
	return &TeamHandler{teamRepo: teamRepo, email: emailClient}
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
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "invalid name")
		return
	}
	teamID, err := h.teamRepo.CreateTeamOwner(r.Context(), req.Name, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "create team owner failed")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": teamID, "name": req.Name})
}

func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	teams, err := h.teamRepo.ListTeamsByUser(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "list teams failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, teams)
}

type inviteRequest struct {
	UserID uint64              `json:"user_id"`
	Role   repo.TeamMemberRole `json:"role"`
}

func (h *TeamHandler) Invite(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	teamIDStr := chi.URLParam(r, "id")
	teamID64, err := strconv.ParseUint(teamIDStr, 10, 64)
	if err != nil || teamID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid team id")
		return
	}
	teamID := uint64(teamID64)

	role, err := h.teamRepo.GetUserRole(r.Context(), teamID, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "get user role failed")
		return
	}
	if role != repo.RoleOwner && role != repo.RoleAdmin {
		httpx.Error(w, http.StatusUnauthorized, "insufficient permission")
		return
	}

	var req inviteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.UserID == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid userID")
		return
	}
	if req.Role == "" {
		req.Role = repo.RoleMember
	}

	if req.Role != repo.RoleAdmin && req.Role != repo.RoleMember {
		httpx.Error(w, http.StatusBadRequest, "invalid role")
	}

	if err := h.teamRepo.AddMember(r.Context(), teamID, req.UserID, req.Role); err != nil {
		if err == repo.ErrConflict {
			httpx.Error(w, http.StatusConflict, "already invited")
		}
		httpx.Error(w, http.StatusInternalServerError, "add member failed")
		return
	}

	emailSent := true
	if h.email != nil {
		if err := h.email.SendInvite(r.Context(), email.InvitePayload{TeamID: teamID, UserID: req.UserID, InviteBy: userID}); err != nil {
			emailSent = false
		}

	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"status": "ok", "emailSent": emailSent})

}
