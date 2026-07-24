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

func TestTrendFilterDefaultsAndExplicitConfiguration(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/results/trend?agent_id=11111111-1111-4111-8111-111111111111&target_ip=104.16.0.1&scheme=http&hostname=example.com&path=/health&port=8080&attempts=4&timeout_ms=2500&time_range=24h", nil)
	before := time.Now().UTC().Add(-24*time.Hour - time.Second)
	filter, err := trendFilterFromRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if filter.AgentID == "" || filter.TargetIP != "104.16.0.1" || filter.Scheme != "http" || filter.Hostname != "example.com" || filter.Path != "/health" {
		t.Fatalf("unexpected trend identity: %+v", filter)
	}
	if filter.Port != 8080 || filter.Attempts != 4 || filter.TimeoutMS != 2500 {
		t.Fatalf("unexpected trend probe configuration: %+v", filter)
	}
	if filter.Since.Before(before) {
		t.Fatalf("expected a recent 24h boundary, got %s", filter.Since)
	}
}

func TestTrendFilterRejectsInvalidOptions(t *testing.T) {
	validAgent := "11111111-1111-4111-8111-111111111111"
	base := "/api/v1/results/trend?agent_id=" + validAgent + "&target_ip=104.16.0.1"
	paths := []string{
		"/api/v1/results/trend?agent_id=not-a-uuid&target_ip=104.16.0.1",
		"/api/v1/results/trend?agent_id=" + validAgent + "&target_ip=invalid",
		base + "&scheme=ftp",
		base + "&hostname=https://example.com",
		base + "&path=health",
		base + "&port=70000",
		base + "&attempts=11",
		base + "&timeout_ms=100",
		base + "&time_range=all",
	}
	for _, path := range paths {
		request := httptest.NewRequest("GET", path, nil)
		if _, err := trendFilterFromRequest(request); err == nil {
			t.Fatalf("expected %s to fail", path)
		}
	}
}

func TestLeagueDashboardFilterDefaultsAndExplicitPaging(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/league", nil)
	filter, err := leagueDashboardFilterFromRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if filter.AgentID != "" || filter.PrefixPage != 1 || filter.PrefixPageSize != 50 || filter.CandidatePage != 1 || filter.CandidatePageSize != 50 {
		t.Fatalf("unexpected league defaults: %+v", filter)
	}

	request = httptest.NewRequest("GET", "/api/v1/league?agent_id=11111111-1111-4111-8111-111111111111&prefix_page=3&prefix_page_size=100&candidate_page=2&candidate_page_size=200", nil)
	filter, err = leagueDashboardFilterFromRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if filter.AgentID == "" || filter.PrefixPage != 3 || filter.PrefixPageSize != 100 || filter.CandidatePage != 2 || filter.CandidatePageSize != 200 {
		t.Fatalf("unexpected league paging: %+v", filter)
	}
}

func TestLeagueDashboardFilterRejectsInvalidPaging(t *testing.T) {
	for _, path := range []string{
		"/api/v1/league?agent_id=invalid",
		"/api/v1/league?prefix_page=0",
		"/api/v1/league?prefix_page_size=201",
		"/api/v1/league?candidate_page=invalid",
		"/api/v1/league?candidate_page_size=0",
	} {
		request := httptest.NewRequest("GET", path, nil)
		if _, err := leagueDashboardFilterFromRequest(request); err == nil {
			t.Fatalf("expected %s to fail", path)
		}
	}
}

func TestLeaguePaginationRequested(t *testing.T) {
	legacy := httptest.NewRequest("GET", "/api/v1/league?limit=500", nil)
	if leaguePaginationRequested(legacy) {
		t.Fatal("legacy limit request must retain the array response")
	}
	for _, path := range []string{
		"/api/v1/league?prefix_page=1",
		"/api/v1/league?prefix_page_size=50",
		"/api/v1/league?candidate_page=1",
		"/api/v1/league?candidate_page_size=50",
	} {
		if !leaguePaginationRequested(httptest.NewRequest("GET", path, nil)) {
			t.Fatalf("expected pagination for %s", path)
		}
	}
}
