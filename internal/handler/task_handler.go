package handler

import (
	"net/http"
	"strconv"

	"github.com/DrummDaddy/task_service/internal/core/usecase"
	"github.com/DrummDaddy/task_service/internal/models"
	"github.com/go-chi/chi/v5"

	"github.com/DrummDaddy/task_service/internal/auth"
	"github.com/DrummDaddy/task_service/internal/httpx"
)

type TaskHandler struct {
	uc *usecase.TaskUsecase
}

func NewTaskHandler(uc *usecase.TaskUsecase) *TaskHandler {
	return &TaskHandler{uc: uc}
}

type createTaskRequest struct {
	TeamID      uint64            `json:"team_id"`
	Title       string            `json:"title"`
	Description *string           `json:"description"`
	Status      models.TaskStatus `json:"status"`
	AssigneeID  *uint64           `json:"assignee_id"`
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createTaskRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	id, err := h.uc.Create(r.Context(), usecase.CreateTaskInput{
		UserID:      userID,
		TeamID:      req.TeamID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "create task failed")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	teamID, _ := strconv.ParseUint(r.URL.Query().Get("team_id"), 10, 64)
	var status *models.TaskStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := models.TaskStatus(s)
		status = &st
	}
	var assigneeID *uint64
	if a := r.URL.Query().Get("assignee_id"); a != "" {
		v, err := strconv.ParseUint(a, 10, 64)
		if err == nil {
			av := uint64(v)
			assigneeID = &av
		}
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	items, err := h.uc.List(r.Context(), usecase.ListTaskInput{
		UserID:   userID,
		TeamID:   uint64(teamID),
		Status:   status,
		Assignee: assigneeID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "list tasks failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

type updateTaskRequest struct {
	Title       *string            `json:"title"`
	Description **string           `json:"description"`
	Status      *models.TaskStatus `json:"status"`
	AssigneeID  **uint64           `json:"assignee_id"`
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var req updateTaskRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := h.uc.Update(r.Context(), usecase.UpdateTaskInput{
		UserID:      userID,
		TaskID:      uint64(taskID64),
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "update failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

type createCommentRequest struct {
	Body string `json:"body"`
}

func (h *TaskHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var req createCommentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	id, err := h.uc.AddComment(r.Context(), userID, uint64(taskID64), req.Body)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "add comment failed")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *TaskHandler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.uc.History(r.Context(), userID, uint64(taskID64), limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "history load failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *TaskHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	taskID64, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || taskID64 == 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid task id")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.uc.ListComments(r.Context(), userID, uint64(taskID64), limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "list comments failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
