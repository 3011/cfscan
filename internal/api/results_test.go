package api

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestResultFilterDefaults(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/results", nil)
	before := time.Now().UTC().Add(-24*time.Hour - time.Second)
	filter, err := resultFilterFromRequest(request, true)
	if err != nil {
		t.Fatal(err)
	}
	if filter.View != "latest" || filter.Page != 1 || filter.PageSize != 50 {
		t.Fatalf("unexpected paging defaults: %+v", filter)
	}
	if filter.Sort != "latency_ms" || filter.Order != "asc" {
		t.Fatalf("unexpected latest sorting: %+v", filter)
	}
	if filter.Since == nil || filter.Since.Before(before) {
		t.Fatalf("expected a recent 24h boundary, got %v", filter.Since)
	}
}

func TestResultFilterHistoryAndExplicitOptions(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/results?view=history&time_range=all&page=3&page_size=100&sort=packet_loss&order=desc&search=1.1.1&job_id=abc", nil)
	filter, err := resultFilterFromRequest(request, true)
	if err != nil {
		t.Fatal(err)
	}
	if filter.View != "history" || filter.Page != 3 || filter.PageSize != 100 {
		t.Fatalf("unexpected pagination: %+v", filter)
	}
	if filter.Sort != "packet_loss" || filter.Order != "desc" || filter.Since != nil {
		t.Fatalf("unexpected filtering: %+v", filter)
	}
	if filter.TargetIP != "1.1.1" || filter.JobID != "abc" {
		t.Fatalf("unexpected search filters: %+v", filter)
	}
}

func TestResultFilterRejectsInvalidOptions(t *testing.T) {
	for _, path := range []string{
		"/api/v1/results?view=current",
		"/api/v1/results?page_size=201",
		"/api/v1/results?sort=unknown",
		"/api/v1/results?order=sideways",
		"/api/v1/results?time_range=forever",
	} {
		request := httptest.NewRequest("GET", path, nil)
		if _, err := resultFilterFromRequest(request, true); err == nil {
			t.Fatalf("expected %s to fail", path)
		}
	}
}
