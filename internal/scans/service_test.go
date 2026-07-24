package scans

import (
	"testing"

	"github.com/3011/cfscan/v2/internal/model"
)

func TestApplyDefaultsAndValidate(t *testing.T) {
	input := model.CreateScanJobRequest{Name: "test"}
	ApplyDefaults(&input)
	if input.Scheme != "https" || input.Port != 443 || input.TargetCount != 128 {
		t.Fatalf("unexpected defaults: %+v", input)
	}
	if err := Validate(input); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsBadPath(t *testing.T) {
	input := model.CreateScanJobRequest{Name: "test", TargetCount: 1, Scheme: "https", Hostname: "cloudflare.com", Path: "trace", Port: 443, Attempts: 1, TimeoutMS: 1000, MaxLatencyMS: 1000, MaxPacketLoss: 50, BlacklistMinutes: 60}
	if err := Validate(input); err == nil {
		t.Fatal("expected path validation error")
	}
}

func TestValidateSamplingModes(t *testing.T) {
	input := model.CreateScanJobRequest{
		Name: "test", SamplingMode: SamplingModeOnePerPrefix, TargetCount: 0,
		Scheme: "https", Hostname: "cloudflare.com", Path: "/cdn-cgi/trace", Port: 443,
		Attempts: 1, TimeoutMS: 1000, MaxLatencyMS: 1000, MaxPacketLoss: 50, BlacklistMinutes: 60,
	}
	if err := Validate(input); err != nil {
		t.Fatalf("one-per-prefix should not require target_count: %v", err)
	}
	input.SamplingMode = SamplingModeLeague
	input.TargetCount = 256
	if err := Validate(input); err != nil {
		t.Fatalf("league should accept a bounded per-agent budget: %v", err)
	}
	input.TargetCount = 0
	if err := Validate(input); err == nil {
		t.Fatal("league should require a positive per-agent budget")
	}
	input.SamplingMode = "unknown"
	if err := Validate(input); err == nil {
		t.Fatal("expected unknown sampling mode to fail")
	}
}
