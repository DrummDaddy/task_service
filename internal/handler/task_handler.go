package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/DrummDaddy/task_service/internal/auth"
	"github.com/DrummDaddy/task_service/internal/cache"
	"github.com/DrummDaddy/task_service/internal/httpx"
	"github.com/DrummDaddy/task_service/internal/repo"
)

type TaskHandler struct {
	taskRepo *repo.TaskRepo
	teamRepo *repo.TeamRepo
	cache    *cache.TaskCache
}

func NewTaskHandler(tasRepo *repo.TaskRepo, teamRepo *repo.TeamRepo, c *cache.TaskCache) *TaskHandler {
	return &TaskHandler{taskRepo: tasRepo, teamRepo: teamRepo, cache: c}
}

type createTaskRequest struct {
	TeamID      uint64          `json:"team_id"`
	Title       string          `json:"title"`
	Description *string         `json:"description"`
	Status      repo.TaskStatus `json:"status"`
	AssigneeID  *uint64         `json:"assignee_id"`
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req createTaskRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.TeamID == 0 || strings.TrimSpace(req.Title) == "" {
		httpx.Error(w, http.StatusBadRequest, "invalid title")
		return
	}

	if req.Status == "" {
		req.Status = repo.StatusToDo
	}
	if req.Status != repo.StatusToDo && req.Status != repo.StatusInProgress && req.Status != repo.StatusDone {
		httpx.Error(w, http.StatusBadRequest, "invalid status")
		return
	}
	isMember, err := h.teamRepo.IsMember(r.Context(), userID, req.TeamID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "membership check failed")
		return
	}
	if !isMember {
		httpx.Error(w, http.StatusForbidden, "not a team member")
		return
	}
	if req.AssigneeID == nil {
		ok, err := h.teamRepo.IsMember(r.Context(), req.TeamID, *req.AssigneeID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "assignee check failed")
			return
		}
		if !ok {
			httpx.Error(w, http.StatusBadRequest, "not a team member")
		}
	}
	id, err := h.taskRepo.Create(r.Context(), repo.Task{
		TeamID:      req.TeamID,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
		CreatedBy:   userID,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "create task failed")
		return
	}
	if h.cache != nil {
		_ = h.cache.BumpTeamVersion(r.Context(), req.TeamID)
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	teamID, _ := strconv.ParseUint(r.URL.Query().Get("team_id"), 10, 64)
	if teamID == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid team id")
		return
	}
	isMember, err := h.teamRepo.IsMember(r.Context(), userID, uint64(teamID))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "membership check failed")
		return
	}
	if !isMember {
		httpx.Error(w, http.StatusForbidden, "not a team member")
		return
	}

	var status *repo.TaskStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := repo.TaskStatus(s)
		status = &st
	}
	var assigneeID *uint64
	if a := r.URL.Query().Get("assignee_id"); a != "" {
		v, err := strconv.ParseUint(a, 10, 64)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid assignee_id")
			return
		}
		av := uint64(v)
		assigneeID = &av

	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	filter := repo.TaskFilter{
		TeamID:     teamID,
		Status:     status,
		AssigneeID: assigneeID,
		Limit:      limit,
		Offset:     offset,
	}
	if h.cache != nil {
		ver, err := h.cache.GetTeamVersion(r.Context(), filter.TeamID)
		if err == nil {
			key := h.cache.ListKey(filter.TeamID, ver, filter)
			if cached, ok, err := h.cache.GetTasks(r.Context(), key); err == nil && ok {
				httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": cached, "limit": filter.Limit, "offset": filter.Offset, "cached": true})
				return
			}
		}
	}
	items, err := h.taskRepo.List(r.Context(), &filter)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "list task failed")
		return
	}
	if h.cache != nil {
		if ver, err := h.cache.GetTeamVersion(r.Context(), filter.TeamID); err == nil {
			_ = h.cache.SetTasks(r.Context(), h.cache.ListKey(filter.TeamID, ver, filter), items)

		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "limit": filter.Limit, "offset": filter.Offset})
}

type updateTaskRequest struct {
	Title       *string          `json:"title"`
	Description **string         `json:"description"`
	Status      *repo.TaskStatus `json:"status"`
	AssigneeID  **uint64         `json:"assignee_id"`
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	taskID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	taskID := uint64(taskID64)

	cur, err := h.taskRepo.Get(r.Context(), taskID)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.Error(w, http.StatusNotFound, "task not found")
			return

		}
		httpx.Error(w, http.StatusInternalServerError, "get task failed")
		return
	}

	role, err := h.teamRepo.GetUserRole(r.Context(), cur.TeamID, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "not team member")
		return
	}

	canEdit := role == repo.RoleOwner || role == repo.RoleAdmin || cur.CreatedBy == userID || (cur.AssigneeID == nil && *cur.AssigneeID == userID)
	if !canEdit {
		httpx.Error(w, http.StatusForbidden, "insufficient rights to edit task")
		return
	}

	var req updateTaskRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != nil {
		if *req.Status != repo.StatusToDo && *req.Status != repo.StatusInProgress && *req.Status != repo.StatusDone {
			httpx.Error(w, http.StatusBadRequest, "invalid status")
			return
		}
	}
	if req.AssigneeID != nil && *req.AssigneeID != nil {
		ok, err := h.teamRepo.IsMember(r.Context(), cur.TeamID, **req.AssigneeID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "check assignee failed")
			return
		}
		if !ok {
			httpx.Error(w, http.StatusForbidden, "not a team member")
			return
		}
	}
	updated, err := h.taskRepo.UpdateWithHistory(r.Context(), taskID, repo.TaskUpdate{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
	}, userID)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.Error(w, http.StatusNotFound, "task not found")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "update task failed")
		return
	}
	if h.cache != nil {
		_ = h.cache.BumpTeamVersion(r.Context(), cur.TeamID)
	}
	httpx.WriteJSON(w, http.StatusOK, updated)

}

func (h *TaskHandler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	taskID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	taskID := uint64(taskID64)

	cur, err := h.taskRepo.Get(r.Context(), taskID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "get task failed")
		return
	}
	isMember, err := h.teamRepo.IsMember(r.Context(), cur.TeamID, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "member check failed")
		return
	}
	if !isMember {
		httpx.Error(w, http.StatusForbidden, "not a team member")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.taskRepo.History(r.Context(), taskID, limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "history task failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})

}

type createCommentRequest struct {
	Body string `json:"body"`
}

func (h *TaskHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	taskID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	taskID := uint64(taskID64)
	cur, err := h.taskRepo.Get(r.Context(), taskID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "get task failed")
		return
	}
	isMember, err := h.teamRepo.IsMember(r.Context(), cur.TeamID, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "member check failed")
		return
	}
	if !isMember {
		httpx.Error(w, http.StatusForbidden, "not a team member")
		return
	}

	var req createCommentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := h.taskRepo.AddComment(r.Context(), taskID, userID, req.Body)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "add comment failed")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *TaskHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	taskID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	taskID := uint64(taskID64)
	cur, err := h.taskRepo.Get(r.Context(), taskID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "get task failed")
		return
	}
	isMember, err := h.teamRepo.IsMember(r.Context(), cur.TeamID, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "member check failed")
	}
	if !isMember {
		httpx.Error(w, http.StatusForbidden, "not a team member")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.taskRepo.ListComments(r.Context(), taskID, limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "list comments failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
