package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/DrummDaddy/task_service/internal/httpx"
	"github.com/DrummDaddy/task_service/internal/repo"
)

type ReportHandler struct {
	reports *repo.ReportsRepo
}

func NewReportHandler(reports *repo.ReportsRepo) *ReportHandler {
	return &ReportHandler{reports: reports}
}

func (h *ReportHandler) TeamStats(w http.ResponseWriter, r *http.Request) {
	rows, err := h.reports.TeamStats(r.Context(), time.Now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not get team stats")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *ReportHandler) TopCreators(w http.ResponseWriter, r *http.Request) {
	rows, err := h.reports.TopCreatorsLastMonth(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not get top creators")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *ReportHandler) IntegrityInvalidAssigness(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rows, err := h.reports.TaskWithInvalidAssignee(r.Context(), limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not get task with invalid assignees")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}
