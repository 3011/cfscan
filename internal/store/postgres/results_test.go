package postgres

import (
	"strings"
	"testing"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
)

func TestLatestResultDatasetFiltersStatusAfterRanking(t *testing.T) {
	since := time.Now().Add(-24 * time.Hour)
	available := true
	query, args := resultDataset(model.ResultFilter{
		View: "latest", AgentID: "agent", JobID: "job", TargetIP: "1.1",
		Available: &available, Colo: "hkg", Since: &since,
	}, true, true)
	if !strings.Contains(query, "ROW_NUMBER() OVER") || !strings.Contains(query, "result_rank = 1") {
		t.Fatalf("latest query does not rank results: %s", query)
	}
	rankAt := strings.Index(query, "result_rank = 1")
	statusAt := strings.Index(query, "q.available")
	if rankAt < 0 || statusAt < rankAt {
		t.Fatalf("availability must be applied after latest-result ranking: %s", query)
	}
	if !strings.Contains(query, "r.job_id") || !strings.Contains(query, "host(r.target_ip) ILIKE") {
		t.Fatalf("candidate filters missing: %s", query)
	}
	if len(args) != 6 {
		t.Fatalf("unexpected argument count %d: %#v", len(args), args)
	}
}

func TestHistoryResultDatasetDoesNotDeduplicate(t *testing.T) {
	query, _ := resultDataset(model.ResultFilter{View: "history"}, true, true)
	if strings.Contains(query, "ROW_NUMBER() OVER") || !strings.Contains(query, "1::bigint AS result_rank") {
		t.Fatalf("history query should preserve every row: %s", query)
	}
}

func TestResultOrderAllowlist(t *testing.T) {
	if got := resultOrder(model.ResultFilter{Sort: "scanned_at", Order: "desc"}); got != "scanned_at DESC, id DESC" {
		t.Fatalf("unexpected scanned order %q", got)
	}
	if got := resultOrder(model.ResultFilter{Sort: "unknown", Order: "desc"}); !strings.HasPrefix(got, "available DESC, latency_ms DESC") {
		t.Fatalf("unexpected default order %q", got)
	}
}
