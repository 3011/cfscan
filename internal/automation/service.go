package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/3011/cfscan/internal/cloudflare"
	"github.com/3011/cfscan/internal/model"
	"github.com/3011/cfscan/internal/scheduling"
)

type Store interface {
	GetBlacklistRecheckSettings(context.Context) (model.BlacklistRecheckSettings, error)
	ClaimDueBlacklistRecheck(context.Context, time.Time, time.Time) (*model.BlacklistRecheckSettings, error)
	RecordBlacklistRecheckRun(context.Context, *time.Time, error) error
	CreateBlacklistRechecks(context.Context, model.BlacklistRecheckSettings) (model.BlacklistRecheckResult, error)
	ListSourceSyncSchedules(context.Context) ([]model.SourceSyncSchedule, error)
	GetSourceSyncSchedule(context.Context, string) (model.SourceSyncSchedule, error)
	ClaimDueSourceSyncSchedules(context.Context, time.Time, time.Time, int) ([]model.SourceSyncSchedule, error)
	RecordSourceSyncRun(context.Context, string, *time.Time, error) error
	StartAutomationRun(context.Context, model.StartAutomationRunRequest) (model.AutomationRun, error)
	FinishAutomationRun(context.Context, string, string, json.RawMessage, error) error
}

type Service struct {
	store  Store
	syncer *cloudflare.Syncer
	logger *slog.Logger
}

func NewService(store Store, syncer *cloudflare.Syncer, logger *slog.Logger) *Service {
	return &Service{store: store, syncer: syncer, logger: logger}
}

func (s *Service) RunDue(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	count := 0
	blacklist, err := s.store.ClaimDueBlacklistRecheck(ctx, now, now.Add(5*time.Minute))
	if err != nil {
		return count, err
	}
	if blacklist != nil {
		count++
		next, nextErr := scheduling.Next(blacklist.CronExpression, blacklist.Timezone, now)
		if nextErr != nil {
			_ = s.store.RecordBlacklistRecheckRun(ctx, nil, nextErr)
		} else {
			_, runErr := s.runBlacklist(ctx, *blacklist, "scheduled")
			_ = s.store.RecordBlacklistRecheckRun(ctx, &next, runErr)
		}
	}

	sources, err := s.store.ClaimDueSourceSyncSchedules(ctx, now, now.Add(5*time.Minute), 3)
	if err != nil {
		return count, err
	}
	for _, item := range sources {
		count++
		next, nextErr := scheduling.Next(item.CronExpression, item.Timezone, now)
		if nextErr != nil {
			_ = s.store.RecordSourceSyncRun(ctx, item.Source, nil, nextErr)
			continue
		}
		_, runErr := s.runSource(ctx, item, "scheduled")
		_ = s.store.RecordSourceSyncRun(ctx, item.Source, &next, runErr)
	}
	return count, nil
}

func (s *Service) RunStartup(ctx context.Context) (int, error) {
	items, err := s.store.ListSourceSyncSchedules(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	count := 0
	for _, item := range items {
		if !item.Enabled || !item.RunOnStartup {
			continue
		}
		count++
		next, nextErr := scheduling.Next(item.CronExpression, item.Timezone, now)
		if nextErr != nil {
			_ = s.store.RecordSourceSyncRun(ctx, item.Source, nil, nextErr)
			continue
		}
		_, runErr := s.runSource(ctx, item, "startup")
		_ = s.store.RecordSourceSyncRun(ctx, item.Source, &next, runErr)
	}
	return count, nil
}

func (s *Service) RunBlacklistNow(ctx context.Context) (model.BlacklistRecheckResult, error) {
	settings, err := s.store.GetBlacklistRecheckSettings(ctx)
	if err != nil {
		return model.BlacklistRecheckResult{}, err
	}
	result, runErr := s.runBlacklist(ctx, settings, "manual")
	_ = s.store.RecordBlacklistRecheckRun(ctx, nil, runErr)
	return result, runErr
}

func (s *Service) RunSourceNow(ctx context.Context, source string) (json.RawMessage, error) {
	item, err := s.store.GetSourceSyncSchedule(ctx, source)
	if err != nil {
		return nil, err
	}
	summary, runErr := s.runSource(ctx, item, "manual")
	_ = s.store.RecordSourceSyncRun(ctx, source, nil, runErr)
	return summary, runErr
}

func (s *Service) runBlacklist(ctx context.Context, settings model.BlacklistRecheckSettings, trigger string) (model.BlacklistRecheckResult, error) {
	run, err := s.startRun(ctx, "blacklist_recheck", "default", "黑名单复查", trigger, settings)
	if err != nil {
		return model.BlacklistRecheckResult{}, err
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	result, runErr := s.store.CreateBlacklistRechecks(runCtx, settings)
	status := "completed"
	if result.Skipped {
		status = "skipped"
	}
	summary := mustJSON(result)
	if finishErr := s.store.FinishAutomationRun(ctx, run.ID, status, summary, runErr); finishErr != nil && runErr == nil {
		runErr = finishErr
	}
	if runErr != nil {
		s.logger.Warn("blacklist recheck failed", "error", runErr)
	} else {
		s.logger.Info("blacklist recheck completed", "targets", result.Targets, "jobs", result.Jobs, "skipped", result.Skipped)
	}
	return result, runErr
}

func (s *Service) runSource(ctx context.Context, item model.SourceSyncSchedule, trigger string) (json.RawMessage, error) {
	run, err := s.startRun(ctx, "source_sync", item.Source, item.Name, trigger, item)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var summary json.RawMessage
	var runErr error
	switch item.Source {
	case "official":
		var result model.SourceStatus
		result, runErr = s.syncer.Sync(runCtx)
		summary = mustJSON(result)
	case "asn":
		var result model.ASNSyncSummary
		result, runErr = s.syncer.SyncEnabledASNs(runCtx)
		summary = mustJSON(result)
	case "colo":
		var result model.ColoSyncSummary
		result, runErr = s.syncer.SyncColos(runCtx)
		summary = mustJSON(result)
	default:
		runErr = fmt.Errorf("unsupported source sync type %q", item.Source)
		summary = json.RawMessage(`{}`)
	}
	if finishErr := s.store.FinishAutomationRun(ctx, run.ID, "completed", summary, runErr); finishErr != nil && runErr == nil {
		runErr = finishErr
	}
	if runErr != nil {
		s.logger.Warn("source sync automation failed", "source", item.Source, "error", runErr)
	} else {
		s.logger.Info("source sync automation completed", "source", item.Source, "trigger", trigger)
	}
	return summary, runErr
}

func (s *Service) startRun(ctx context.Context, automationType, key, name, trigger string, config any) (model.AutomationRun, error) {
	return s.store.StartAutomationRun(ctx, model.StartAutomationRunRequest{
		AutomationType: automationType,
		AutomationKey:  key,
		Name:           name,
		Trigger:        trigger,
		ConfigSnapshot: mustJSON(config),
	})
}

func mustJSON(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return encoded
}
