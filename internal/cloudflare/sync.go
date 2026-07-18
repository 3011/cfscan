package cloudflare

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
)

const SourceOfficial = "cloudflare_official"

type PrefixStore interface {
	ReplaceCloudflarePrefixes(context.Context, []model.Prefix) (model.SourceStatus, error)
	RecordSourceError(context.Context, string, error) error
	ListASNSources(context.Context, bool) ([]model.ASNSource, error)
	ReplaceASNPrefixes(context.Context, int64, []model.Prefix) (model.ASNSource, error)
	RecordASNError(context.Context, int64, error) error
	ReplaceColoLocations(context.Context, []model.ColoLocation) (model.ColoSyncSummary, error)
}

type Syncer struct {
	store      PrefixStore
	client     *http.Client
	v4URL      string
	v6URL      string
	asnBaseURL string
	coloURL    string
}

func NewSyncer(store PrefixStore) *Syncer {
	return &Syncer{
		store:      store,
		client:     &http.Client{Timeout: 30 * time.Second},
		v4URL:      "https://www.cloudflare.com/ips-v4",
		v6URL:      "https://www.cloudflare.com/ips-v6",
		asnBaseURL: "https://stat.ripe.net/data/announced-prefixes/data.json?resource=AS",
		coloURL:    "https://www.cloudflarestatus.com/api/v2/components.json",
	}
}

// Sync refreshes Cloudflare's explicitly published reverse-proxy address ranges.
func (s *Syncer) Sync(ctx context.Context) (model.SourceStatus, error) {
	v4, err := s.fetchOfficial(ctx, s.v4URL, 4)
	if err != nil {
		_ = s.store.RecordSourceError(ctx, SourceOfficial, err)
		return model.SourceStatus{}, err
	}
	v6, err := s.fetchOfficial(ctx, s.v6URL, 6)
	if err != nil {
		_ = s.store.RecordSourceError(ctx, SourceOfficial, err)
		return model.SourceStatus{}, err
	}
	return s.store.ReplaceCloudflarePrefixes(ctx, append(v4, v6...))
}

// SyncASN refreshes all currently announced prefixes for one managed origin ASN.
func (s *Syncer) SyncASN(ctx context.Context, asn int64) (model.ASNSource, error) {
	prefixes, err := s.fetchASN(ctx, asn)
	if err != nil {
		_ = s.store.RecordASNError(ctx, asn, err)
		return model.ASNSource{}, err
	}
	return s.store.ReplaceASNPrefixes(ctx, asn, prefixes)
}

// SyncEnabledASNs refreshes every enabled ASN and retains per-ASN failures.
func (s *Syncer) SyncEnabledASNs(ctx context.Context) (model.ASNSyncSummary, error) {
	sources, err := s.store.ListASNSources(ctx, true)
	if err != nil {
		return model.ASNSyncSummary{}, err
	}
	summary := model.ASNSyncSummary{Items: make([]model.ASNSource, 0, len(sources))}
	for _, source := range sources {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		item, syncErr := s.SyncASN(ctx, source.ASN)
		if syncErr != nil {
			source.Status = "error"
			source.LastError = syncErr.Error()
			summary.Items = append(summary.Items, source)
			summary.Failed++
			continue
		}
		summary.Items = append(summary.Items, item)
		summary.Synced++
	}
	return summary, nil
}

type statusPageResponse struct {
	Components []struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		Status   string   `json:"status"`
		GroupID  string   `json:"group_id"`
		Group    bool     `json:"group"`
		Children []string `json:"components"`
	} `json:"components"`
}

var coloSuffixPattern = regexp.MustCompile(`\s+-\s+\(([A-Z0-9]{3})\)$`)

// SyncColos refreshes Cloudflare colo geography from Cloudflare's public status components.
func (s *Syncer) SyncColos(ctx context.Context) (model.ColoSyncSummary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.coloURL, nil)
	if err != nil {
		return model.ColoSyncSummary{}, fmt.Errorf("create Cloudflare colo request: %w", err)
	}
	req.Header.Set("User-Agent", "cfscan-center/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return model.ColoSyncSummary{}, fmt.Errorf("fetch Cloudflare colo locations: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return model.ColoSyncSummary{}, fmt.Errorf("fetch Cloudflare colo locations: status %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	var payload statusPageResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&payload); err != nil {
		return model.ColoSyncSummary{}, fmt.Errorf("decode Cloudflare colo locations: %w", err)
	}
	continents := map[string]string{}
	allowed := map[string]bool{
		"Africa": true, "Asia": true, "Europe": true, "Latin America & the Caribbean": true,
		"Middle East": true, "North America": true, "Oceania": true,
	}
	for _, component := range payload.Components {
		if component.Group && allowed[component.Name] {
			continents[component.ID] = component.Name
		}
	}
	locations := make([]model.ColoLocation, 0, 350)
	seen := map[string]struct{}{}
	for _, component := range payload.Components {
		continent, ok := continents[component.GroupID]
		if !ok {
			continue
		}
		match := coloSuffixPattern.FindStringSubmatch(component.Name)
		if len(match) != 2 {
			continue
		}
		code := match[1]
		base := strings.TrimSpace(coloSuffixPattern.ReplaceAllString(component.Name, ""))
		parts := strings.Split(base, ",")
		city := strings.TrimSpace(parts[0])
		country := city
		if len(parts) >= 2 {
			country = strings.TrimSpace(parts[len(parts)-1])
		}
		if city == "" || country == "" {
			continue
		}
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		locations = append(locations, model.ColoLocation{Code: code, City: city, Country: country, Continent: continent, Status: component.Status})
	}
	if len(locations) == 0 {
		return model.ColoSyncSummary{}, fmt.Errorf("Cloudflare status returned no colo locations")
	}
	return s.store.ReplaceColoLocations(ctx, locations)
}

func (s *Syncer) fetchOfficial(ctx context.Context, url string, version int) ([]model.Prefix, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create Cloudflare prefix request: %w", err)
	}
	req.Header.Set("User-Agent", "cfscan-center/1.0")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("fetch %s: status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(message)))
	}

	items := make([]model.Prefix, 0, 32)
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for scanner.Scan() {
		value := strings.TrimSpace(scanner.Text())
		if value == "" {
			continue
		}
		prefix, err := parsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("parse Cloudflare prefix %q: %w", value, err)
		}
		if prefix.IPVersion != version {
			return nil, fmt.Errorf("unexpected IP version for %s", value)
		}
		prefix.Source = SourceOfficial
		items = append(items, prefix)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read Cloudflare prefixes: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("Cloudflare returned no IPv%d prefixes", version)
	}
	return items, nil
}

type ripeResponse struct {
	Status string `json:"status"`
	Data   struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	} `json:"data"`
}

func (s *Syncer) fetchASN(ctx context.Context, asn int64) ([]model.Prefix, error) {
	if asn <= 0 || asn > 4294967295 {
		return nil, fmt.Errorf("invalid ASN %d", asn)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%d", s.asnBaseURL, asn), nil)
	if err != nil {
		return nil, fmt.Errorf("create ASN prefix request: %w", err)
	}
	req.Header.Set("User-Agent", "cfscan-center/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch AS%d prefixes: %w", asn, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("fetch AS%d prefixes: status %d: %s", asn, resp.StatusCode, strings.TrimSpace(string(message)))
	}
	var payload ripeResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode AS%d prefixes: %w", asn, err)
	}
	if payload.Status != "ok" {
		return nil, fmt.Errorf("AS%d prefix source returned status %q", asn, payload.Status)
	}
	seen := make(map[string]struct{}, len(payload.Data.Prefixes))
	items := make([]model.Prefix, 0, len(payload.Data.Prefixes))
	source := fmt.Sprintf("asn:%d", asn)
	for _, raw := range payload.Data.Prefixes {
		prefix, err := parsePrefix(raw.Prefix)
		if err != nil {
			return nil, fmt.Errorf("parse AS%d prefix %q: %w", asn, raw.Prefix, err)
		}
		if _, ok := seen[prefix.CIDR]; ok {
			continue
		}
		seen[prefix.CIDR] = struct{}{}
		prefix.Source = source
		items = append(items, prefix)
	}
	return items, nil
}

func parsePrefix(value string) (model.Prefix, error) {
	prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
	if err != nil {
		return model.Prefix{}, err
	}
	version := 6
	if prefix.Addr().Is4() {
		version = 4
	}
	return model.Prefix{CIDR: prefix.Masked().String(), IPVersion: version, Active: true}, nil
}
