package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/DrummDaddy/task_service/internal/core/usecase"
	"github.com/DrummDaddy/task_service/internal/httpx"
)

type ReportHandler struct {
	uc *usecase.ReportUsecase
}

func NewReportHandler(uc *usecase.ReportUsecase) *ReportHandler {
	return &ReportHandler{uc: uc}
}

func (h *ReportHandler) TeamStats(w http.ResponseWriter, r *http.Request) {
	rows, err := h.uc.TeamStats(r.Context(), time.Now())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "report failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *ReportHandler) TopCreators(w http.ResponseWriter, r *http.Request) {
	rows, err := h.uc.TopCreatorsLastMonth(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "report failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (h *ReportHandler) IntegrityInvalidAssignees(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rows, err := h.uc.TaskWithInvalidAssignee(r.Context(), limit, offset)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "report failed")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": rows})
}
