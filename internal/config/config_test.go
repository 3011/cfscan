package config

import "testing"

func TestNewInstallDisablesLegacySharedAgentTokenByDefault(t *testing.T) {
	t.Setenv("CFSCAN_AGENT_TOKEN", "")
	if got := LoadServer().AgentToken; got != "" {
		t.Fatalf("Server legacy Agent token default = %q", got)
	}
	if got := LoadAgent().Token; got != "" {
		t.Fatalf("Agent legacy token default = %q", got)
	}
}

func TestEnrollmentReleaseDefaults(t *testing.T) {
	for _, key := range []string{"CFSCAN_AGENT_IMAGE", "CFSCAN_AGENT_VERSION", "CFSCAN_AGENT_ENROLLMENT_TTL", "CFSCAN_AGENT_ENROLLMENT_POLL_INTERVAL"} {
		t.Setenv(key, "")
	}
	cfg := LoadServer()
	if cfg.AgentImage != "ghcr.io/3011/cfscan-agent:v1.1.0" || cfg.AgentVersion != "v1.1.0" {
		t.Fatalf("unexpected Agent release defaults: image=%q version=%q", cfg.AgentImage, cfg.AgentVersion)
	}
}
