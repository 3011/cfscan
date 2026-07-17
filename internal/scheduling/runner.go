package scheduling

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/3011/cfscan/internal/model"
)

type RunnerStore interface {
	ClaimDueScanSchedules(context.Context, time.Time, time.Time, int) ([]model.ScanSchedule, error)
	RecordScanScheduleRun(context.Context, string, *time.Time, *string, error) error
	StartAutomationRun(context.Context, model.StartAutomationRunRequest) (model.AutomationRun, error)
	FinishAutomationRun(context.Context, string, string, json.RawMessage, error) error
}

type JobCreator interface {
	Create(context.Context, model.CreateScanJobRequest, string) (model.ScanJob, error)
}

type Runner struct {
	store   RunnerStore
	creator JobCreator
	logger  *slog.Logger
}

func NewRunner(store RunnerStore, creator JobCreator, logger *slog.Logger) *Runner {
	return &Runner{store: store, creator: creator, logger: logger}
}

func (r *Runner) RunDue(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	items, err := r.store.ClaimDueScanSchedules(ctx, now, now.Add(5*time.Minute), 20)
	if err != nil {
		return 0, err
	}
	for _, item := range items {
		config, _ := json.Marshal(item)
		automationRun, startErr := r.store.StartAutomationRun(ctx, model.StartAutomationRunRequest{
			AutomationType: "scan_schedule", AutomationKey: item.ID, Name: item.Name,
			Trigger: "scheduled", ConfigSnapshot: config,
		})
		if startErr != nil {
			r.logger.Error("start scan schedule audit run", "schedule_id", item.ID, "error", startErr)
			continue
		}
		nextRun, nextErr := Next(item.CronExpression, item.Timezone, now)
		if nextErr != nil {
			_ = r.store.RecordScanScheduleRun(ctx, item.ID, nil, nil, nextErr)
			_ = r.store.FinishAutomationRun(ctx, automationRun.ID, "failed", json.RawMessage(`{}`), nextErr)
			r.logger.Error("scheduled scan has invalid schedule", "schedule_id", item.ID, "error", nextErr)
			continue
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		job, runErr := r.creator.Create(runCtx, item.JobRequest(), "scheduled")
		cancel()
		var jobID *string
		summary := json.RawMessage(`{}`)
		if runErr == nil {
			jobID = &job.ID
			summary, _ = json.Marshal(map[string]any{"job_id": job.ID, "tasks": job.TotalTargets})
			r.logger.Info("scheduled scan job created", "schedule_id", item.ID, "job_id", job.ID, "next_run_at", nextRun)
		} else {
			r.logger.Warn("scheduled scan job failed", "schedule_id", item.ID, "error", runErr, "next_run_at", nextRun)
		}
		_ = r.store.FinishAutomationRun(ctx, automationRun.ID, "completed", summary, runErr)
		if err := r.store.RecordScanScheduleRun(ctx, item.ID, &nextRun, jobID, runErr); err != nil {
			r.logger.Error("record scheduled scan result", "schedule_id", item.ID, "error", err)
		}
	}
	return len(items), nil
}
