package usecase

import (
	"context"
	"time"

	"github.com/DrummDaddy/task_service/internal/core/ports"
	"github.com/DrummDaddy/task_service/internal/models"
)

type ReportUsecase struct {
	reports ports.ReportsRepository
}

func NewReportUsecase(reports ports.ReportsRepository) *ReportUsecase {
	return &ReportUsecase{reports: reports}
}

func (uc *ReportUsecase) TeamStats(ctx context.Context, now time.Time) ([]models.TeamStatsRow, error) {
	return uc.reports.TeamStats(ctx, now)
}
func (uc *ReportUsecase) TopCreatorsLastMonth(ctx context.Context) ([]models.TopCreatorRow, error) {
	return uc.reports.TopCreatorsLastMonth(ctx)
}

func (uc *ReportUsecase) TaskWithInvalidAssignee(ctx context.Context, limit, offset int) ([]models.IntegrityTaskRow, error) {
	return uc.reports.TaskWithInvalidAssignee(ctx, limit, offset)
}
