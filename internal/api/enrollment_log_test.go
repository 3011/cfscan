package api

import (
	"strings"
	"testing"
)

func TestSafeLogPathRedactsPairingToken(t *testing.T) {
	pairingToken := strings.Join([]string{"8b8f3bf4", "5ea2", "49e0", "b07e", "087f69973223"}, "-")
	cases := map[string]string{
		"/api/v1/agent-enrollments/" + pairingToken:              "/api/v1/agent-enrollments/{pairingToken}",
		"/api/v1/agent-enrollments/" + pairingToken + "/approve": "/api/v1/agent-enrollments/{pairingToken}/approve",
		"/api/v1/agent-enrollments/id/" + pairingToken:           "/api/v1/agent-enrollments/id/" + pairingToken,
		"/api/v1/agents": "/api/v1/agents",
	}
	for input, expected := range cases {
		if got := safeLogPath(input); got != expected {
			t.Fatalf("safeLogPath(%q) = %q, want %q", input, got, expected)
		}
	}
}
