package cloudflare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/3011/cfscan/v2/internal/model"
)

func TestFetchASNPrefixes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/AS13335" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","data":{"prefixes":[{"prefix":"104.16.0.0/13"},{"prefix":"104.16.0.0/13"},{"prefix":"2606:4700::/32"}]}}`))
	}))
	defer server.Close()

	syncer := &Syncer{client: server.Client(), asnBaseURL: server.URL + "/AS"}
	items, err := syncer.fetchASN(context.Background(), 13335)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 deduplicated prefixes, got %d", len(items))
	}
	if items[0].Source != "asn:13335" || items[0].IPVersion != 4 {
		t.Fatalf("unexpected first prefix: %+v", items[0])
	}
	if items[1].IPVersion != 6 {
		t.Fatalf("unexpected IPv6 prefix: %+v", items[1])
	}
}

func TestParsePrefixMasksHostBits(t *testing.T) {
	item, err := parsePrefix("104.16.1.2/13")
	if err != nil {
		t.Fatal(err)
	}
	if item.CIDR != "104.16.0.0/13" {
		t.Fatalf("unexpected CIDR %q", item.CIDR)
	}
}

type captureColoStore struct {
	locations []model.ColoLocation
}

func (s *captureColoStore) ReplaceCloudflarePrefixes(context.Context, []model.Prefix) (model.SourceStatus, error) {
	return model.SourceStatus{}, nil
}
func (s *captureColoStore) RecordSourceError(context.Context, string, error) error { return nil }
func (s *captureColoStore) ListASNSources(context.Context, bool) ([]model.ASNSource, error) {
	return nil, nil
}
func (s *captureColoStore) ReplaceASNPrefixes(context.Context, int64, []model.Prefix) (model.ASNSource, error) {
	return model.ASNSource{}, nil
}
func (s *captureColoStore) RecordASNError(context.Context, int64, error) error { return nil }
func (s *captureColoStore) ReplaceColoLocations(_ context.Context, items []model.ColoLocation) (model.ColoSyncSummary, error) {
	s.locations = items
	return model.ColoSyncSummary{Locations: len(items)}, nil
}

func TestSyncColosParsesGroupedLocationsAndSingleNameRegions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"components":[
			{"id":"asia","name":"Asia","group":true,"components":["hkg","sin"]},
			{"id":"hkg","name":"Hong Kong - (HKG)","status":"operational","group_id":"asia","group":false},
			{"id":"sin","name":"Singapore, Singapore - (SIN)","status":"operational","group_id":"asia","group":false}
		]}`))
	}))
	defer server.Close()

	store := &captureColoStore{}
	syncer := &Syncer{store: store, client: server.Client(), coloURL: server.URL}
	summary, err := syncer.SyncColos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.Locations != 2 || len(store.locations) != 2 {
		t.Fatalf("expected 2 locations, got summary=%d items=%d", summary.Locations, len(store.locations))
	}
	if got := store.locations[0]; got.Code != "HKG" || got.City != "Hong Kong" || got.Country != "Hong Kong" || got.Continent != "Asia" {
		t.Fatalf("unexpected HKG location: %+v", got)
	}
	if got := store.locations[1]; got.Code != "SIN" || got.City != "Singapore" || got.Country != "Singapore" {
		t.Fatalf("unexpected SIN location: %+v", got)
	}
}
