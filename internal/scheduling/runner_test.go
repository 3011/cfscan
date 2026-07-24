package scheduling

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
	"github.com/3011/cfscan/v2/internal/scans"
)

type runnerStoreStub struct {
	items          []model.ScanSchedule
	finishedStatus string
	finishedError  error
	recordedError  error
}

func (s *runnerStoreStub) ClaimDueScanSchedules(context.Context, time.Time, time.Time, int) ([]model.ScanSchedule, error) {
	return s.items, nil
}
func (s *runnerStoreStub) RecordScanScheduleRun(_ context.Context, _ string, _ *time.Time, _ *string, err error) error {
	s.recordedError = err
	return nil
}
func (s *runnerStoreStub) StartAutomationRun(context.Context, model.StartAutomationRunRequest) (model.AutomationRun, error) {
	return model.AutomationRun{ID: "run"}, nil
}
func (s *runnerStoreStub) FinishAutomationRun(_ context.Context, _ string, status string, _ json.RawMessage, err error) error {
	s.finishedStatus = status
	s.finishedError = err
	return nil
}

type jobCreatorStub struct{ err error }

func (s jobCreatorStub) Create(context.Context, model.CreateScanJobRequest, string) (model.ScanJob, error) {
	return model.ScanJob{}, s.err
}

func TestRunnerSkipsLeagueScheduleWhenNoPrefixIsDue(t *testing.T) {
	store := &runnerStoreStub{items: []model.ScanSchedule{{
		ID: "schedule", Name: "league", CronExpression: "@hourly", Timezone: "UTC",
	}}}
	runner := NewRunner(store, jobCreatorStub{err: scans.ErrNoLeagueTargetsDue}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	count, err := runner.RunDue(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("got %d claimed schedules", count)
	}
	if store.finishedStatus != "skipped" || store.finishedError != nil || store.recordedError != nil {
		t.Fatalf("unexpected completion: status=%q finish=%v record=%v", store.finishedStatus, store.finishedError, store.recordedError)
	}
}
