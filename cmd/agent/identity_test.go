package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveIdentityUsesProtectedAtomicFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "identity.json")
	input := agentIdentity{ServerURL: "https://agent.example.com", AgentID: "agent", Token: "secret", Name: "node", Concurrency: 64}
	if err := saveIdentity(path, input); err != nil {
		t.Fatal(err)
	}
	item, err := loadIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	if item != input {
		t.Fatalf("unexpected identity: %+v", item)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("identity mode = %o", info.Mode().Perm())
	}
}

func TestValidateEnrollmentServer(t *testing.T) {
	for _, value := range []string{"https://agent.example.com", "http://localhost:8080", "http://127.0.0.1:8080", "http://[::1]:8080"} {
		if err := validateEnrollmentServer(value, false); err != nil {
			t.Fatalf("expected %s to pass: %v", value, err)
		}
	}
	if err := validateEnrollmentServer("http://example.com", false); err == nil {
		t.Fatal("expected remote cleartext URL to fail")
	}
	if err := validateEnrollmentServer("http://example.com", true); err != nil {
		t.Fatal(err)
	}
}
