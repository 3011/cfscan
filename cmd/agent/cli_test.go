package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRequiresIndependentIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "identity.json")
	t.Setenv("CFSCAN_AGENT_IDENTITY_FILE", path)
	err := runSavedIdentity(context.Background(), slog.Default())
	if err == nil || !strings.Contains(err.Error(), "run connect or join first") {
		t.Fatalf("unexpected error: %v", err)
	}
}
