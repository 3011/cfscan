package api

import (
	"testing"

	"github.com/3011/cfscan/v2/internal/model"
)

func TestNormalizeBlacklistRecheck(t *testing.T) {
	input := model.UpdateBlacklistRecheckSettingsRequest{
		Enabled: true, DueOnly: true, Fraction: 0.5, MaxTargets: 500, SkipIfRunning: true,
		Attempts: 3, TimeoutMS: 5000, MaxLatencyMS: 1000, MaxPacketLoss: 50, RetryMinutes: 120,
	}
	next, err := normalizeBlacklistRecheck(&input)
	if err != nil {
		t.Fatalf("normalize blacklist recheck: %v", err)
	}
	if input.CronExpression != "*/15 * * * *" || input.Timezone != "Asia/Shanghai" || next.IsZero() {
		t.Fatalf("unexpected defaults: %+v next=%v", input, next)
	}
}

func TestNormalizeBlacklistRecheckRejectsInvalidFraction(t *testing.T) {
	input := model.UpdateBlacklistRecheckSettingsRequest{
		Fraction: 1.2, MaxTargets: 500, Attempts: 3, TimeoutMS: 5000,
		MaxLatencyMS: 1000, MaxPacketLoss: 50, RetryMinutes: 120,
	}
	if _, err := normalizeBlacklistRecheck(&input); err == nil {
		t.Fatal("expected invalid fraction to fail")
	}
}

func TestNormalizeSourceSync(t *testing.T) {
	input := model.UpdateSourceSyncScheduleRequest{Enabled: true, RunOnStartup: true}
	next, err := normalizeSourceSync(&input)
	if err != nil {
		t.Fatalf("normalize source sync: %v", err)
	}
	if input.CronExpression != "0 */6 * * *" || input.Timezone != "Asia/Shanghai" || next.IsZero() {
		t.Fatalf("unexpected defaults: %+v next=%v", input, next)
	}
}
