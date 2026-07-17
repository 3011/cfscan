package api

import "testing"

func TestEnrollmentTextValidation(t *testing.T) {
	if !validApprovedAgent("广州节点", "Guangzhou", "Asia", 64) {
		t.Fatal("expected valid multilingual Agent metadata")
	}
	if validApprovedAgent("", "Guangzhou", "Asia", 64) {
		t.Fatal("expected empty name to fail")
	}
	if validApprovedAgent(string(make([]rune, maxAgentNameLength+1)), "Guangzhou", "Asia", 64) {
		t.Fatal("expected overlong name to fail")
	}
	if validApprovedAgent("node", "Guangzhou", "Asia", 0) || validApprovedAgent("node", "Guangzhou", "Asia", maxAgentConcurrency+1) {
		t.Fatal("expected invalid concurrency to fail")
	}
}

func TestNormalizePairingToken(t *testing.T) {
	input := " 8B8F3BF4-5EA2-49E0-B07E-087F69973223 "
	expected := "8b8f3bf4-5ea2-49e0-b07e-087f69973223"
	if got := normalizePairingToken(input); got != expected {
		t.Fatalf("normalizePairingToken() = %q", got)
	}
}
