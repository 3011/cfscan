package main

import "testing"

func TestAgentVersionUsesInjectedVersion(t *testing.T) {
	previous := version
	version = "v2.0.0-test"
	t.Cleanup(func() { version = previous })
	if got := agentVersion(); got != "v2.0.0-test" {
		t.Fatalf("agentVersion() = %q", got)
	}
}
