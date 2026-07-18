package config

import "testing"

func TestPublicURLUsesSingleSetting(t *testing.T) {
	t.Setenv("CFSCAN_PUBLIC_URL", "https://cfscan.example.com")
	if got := LoadServer().PublicURL; got != "https://cfscan.example.com" {
		t.Fatalf("PublicURL = %q", got)
	}
}

func TestAgentRuntimeUsesEnrollmentSettingsOnly(t *testing.T) {
	t.Setenv("CFSCAN_CENTER_URL", "https://cfscan.example.com")
	t.Setenv("CFSCAN_AGENT_CONCURRENCY", "32")
	cfg := LoadAgent()
	if cfg.CenterURL != "https://cfscan.example.com" || cfg.Concurrency != 32 {
		t.Fatalf("unexpected Agent config: %+v", cfg)
	}
}
