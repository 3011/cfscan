package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/3011/cfscan/v2/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var initialSchema string

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	store := &Store{pool: pool}
	if err := store.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, initialSchema); err != nil {
		return fmt.Errorf("apply initial schema: %w", err)
	}
	return nil
}

func (s *Store) Heartbeat(ctx context.Context, agentID string) error {
	commandTag, err := s.pool.Exec(ctx, `
UPDATE agents SET status = 'online', last_seen_at = NOW(), updated_at = NOW() WHERE id = $1`, agentID)
	if err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

func (s *Store) ListAgents(ctx context.Context) ([]model.Agent, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id::text, name, region, continent, concurrency,
       CASE WHEN last_seen_at >= NOW() - INTERVAL '45 seconds' THEN 'online' ELSE 'offline' END,
       os, architecture, version, last_seen_at, created_at
FROM agents ORDER BY continent, region, name`)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()
	items := make([]model.Agent, 0)
	for rows.Next() {
		var item model.Agent
		if err := rows.Scan(&item.ID, &item.Name, &item.Region, &item.Continent, &item.Concurrency, &item.Status, &item.OS, &item.Architecture, &item.Version, &item.LastSeenAt, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Overview(ctx context.Context) (model.Overview, error) {
	var result model.Overview
	err := s.pool.QueryRow(ctx, `
WITH latest AS (
    SELECT DISTINCT ON (agent_id, target_ip) agent_id, target_ip, available, latency_ms
    FROM scan_results
    ORDER BY agent_id, target_ip, scanned_at DESC
)
SELECT
    (SELECT COUNT(*) FROM agents),
    (SELECT COUNT(*) FROM agents WHERE last_seen_at >= NOW() - INTERVAL '45 seconds'),
    (SELECT COUNT(DISTINCT cidr) FROM (
        SELECT cidr FROM cloudflare_prefixes WHERE active
        UNION ALL
        SELECT p.cidr FROM asn_prefixes p JOIN asn_sources a ON a.asn = p.asn
        WHERE p.active AND a.enabled
    ) prefixes),
    (SELECT COUNT(*) FROM ip_targets WHERE enabled),
    (SELECT COUNT(*) FROM latest WHERE available),
    (SELECT COUNT(*) FROM blacklist_entries WHERE retry_after > NOW()),
    (SELECT COUNT(*) FROM scan_jobs WHERE status IN ('pending', 'running')),
    (SELECT COUNT(*) FROM scan_jobs WHERE status = 'completed'),
    COALESCE((SELECT AVG(latency_ms) FROM latest WHERE available), 0),
    (SELECT COUNT(*) FROM scan_results WHERE scanned_at >= NOW() - INTERVAL '24 hours')`).Scan(
		&result.AgentsTotal, &result.AgentsOnline, &result.PrefixesTotal, &result.IPsTotal,
		&result.IPsAvailable, &result.IPsBlacklisted, &result.RunningJobs, &result.CompletedJobs,
		&result.AverageLatencyMS, &result.ResultsLast24Hour,
	)
	if err != nil {
		return model.Overview{}, fmt.Errorf("load overview: %w", err)
	}
	return result, nil
}

func (s *Store) ReplaceCloudflarePrefixes(ctx context.Context, prefixes []model.Prefix) (model.SourceStatus, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.SourceStatus{}, fmt.Errorf("begin prefix sync: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE cloudflare_prefixes SET active = FALSE WHERE source = 'cloudflare_official'`); err != nil {
		return model.SourceStatus{}, fmt.Errorf("deactivate prefixes: %w", err)
	}
	v4, v6 := 0, 0
	for _, prefix := range prefixes {
		if _, err := tx.Exec(ctx, `
INSERT INTO cloudflare_prefixes (cidr, ip_version, source, active, first_seen_at, last_seen_at)
VALUES ($1::cidr, $2, 'cloudflare_official', TRUE, NOW(), NOW())
ON CONFLICT (cidr) DO UPDATE SET ip_version = EXCLUDED.ip_version, source = EXCLUDED.source,
    active = TRUE, last_seen_at = NOW()`, prefix.CIDR, prefix.IPVersion); err != nil {
			return model.SourceStatus{}, fmt.Errorf("upsert prefix %s: %w", prefix.CIDR, err)
		}
		if prefix.IPVersion == 4 {
			v4++
		} else {
			v6++
		}
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
INSERT INTO source_sync_state (source, status, prefix_count, ipv4_count, ipv6_count, last_synced_at, last_error, updated_at)
VALUES ('cloudflare_official', 'ok', $1, $2, $3, $4, '', NOW())
ON CONFLICT (source) DO UPDATE SET status = 'ok', prefix_count = EXCLUDED.prefix_count,
    ipv4_count = EXCLUDED.ipv4_count, ipv6_count = EXCLUDED.ipv6_count,
    last_synced_at = EXCLUDED.last_synced_at, last_error = '', updated_at = NOW()`, len(prefixes), v4, v6, now); err != nil {
		return model.SourceStatus{}, fmt.Errorf("record prefix sync: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return model.SourceStatus{}, fmt.Errorf("commit prefix sync: %w", err)
	}
	return model.SourceStatus{Source: "cloudflare_official", Status: "ok", PrefixCount: len(prefixes), IPv4Count: v4, IPv6Count: v6, LastSyncedAt: &now}, nil
}

func (s *Store) RecordSourceError(ctx context.Context, source string, syncErr error) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO source_sync_state (source, status, last_error, updated_at)
VALUES ($1, 'error', $2, NOW())
ON CONFLICT (source) DO UPDATE SET status = 'error', last_error = EXCLUDED.last_error, updated_at = NOW()`, source, truncate(syncErr.Error(), 2000))
	return err
}

func (s *Store) SourceStatus(ctx context.Context, source string) (model.SourceStatus, error) {
	var result model.SourceStatus
	err := s.pool.QueryRow(ctx, `
SELECT source, status, prefix_count, ipv4_count, ipv6_count, last_synced_at, last_error
FROM source_sync_state WHERE source = $1`, source).Scan(
		&result.Source, &result.Status, &result.PrefixCount, &result.IPv4Count, &result.IPv6Count, &result.LastSyncedAt, &result.LastError,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.SourceStatus{Source: source, Status: "never"}, nil
	}
	if err != nil {
		return model.SourceStatus{}, fmt.Errorf("load source status: %w", err)
	}
	return result, nil
}

func (s *Store) ListASNSources(ctx context.Context, enabledOnly bool) ([]model.ASNSource, error) {
	query := `
SELECT asn, name, organization, enabled, managed, sync_status, prefix_count,
       ipv4_count, ipv6_count, last_synced_at, last_error, created_at, updated_at
FROM asn_sources`
	if enabledOnly {
		query += ` WHERE enabled`
	}
	query += ` ORDER BY managed DESC, asn`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list ASN sources: %w", err)
	}
	defer rows.Close()
	items := make([]model.ASNSource, 0)
	for rows.Next() {
		var item model.ASNSource
		if err := scanASNSource(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateASNSource(ctx context.Context, input model.CreateASNSourceRequest) (model.ASNSource, error) {
	if input.ASN <= 0 || input.ASN > 4294967295 {
		return model.ASNSource{}, fmt.Errorf("ASN must be between 1 and 4294967295")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = fmt.Sprintf("AS%d", input.ASN)
	}
	organization := strings.TrimSpace(input.Organization)
	if organization == "" {
		organization = "Cloudflare, Inc."
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	var item model.ASNSource
	err := scanASNSource(s.pool.QueryRow(ctx, `
INSERT INTO asn_sources (asn, name, organization, enabled, managed)
VALUES ($1, $2, $3, $4, FALSE)
RETURNING asn, name, organization, enabled, managed, sync_status, prefix_count,
          ipv4_count, ipv6_count, last_synced_at, last_error, created_at, updated_at`,
		input.ASN, name, organization, enabled), &item)
	if err != nil {
		return model.ASNSource{}, fmt.Errorf("create ASN source: %w", err)
	}
	return item, nil
}

func (s *Store) UpdateASNSource(ctx context.Context, asn int64, input model.UpdateASNSourceRequest) (model.ASNSource, error) {
	var name, organization, enabled any
	if input.Name != nil {
		value := strings.TrimSpace(*input.Name)
		if value == "" {
			return model.ASNSource{}, fmt.Errorf("name cannot be empty")
		}
		name = value
	}
	if input.Organization != nil {
		value := strings.TrimSpace(*input.Organization)
		if value == "" {
			return model.ASNSource{}, fmt.Errorf("organization cannot be empty")
		}
		organization = value
	}
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	var item model.ASNSource
	err := scanASNSource(s.pool.QueryRow(ctx, `
UPDATE asn_sources SET
    name = COALESCE($2::text, name),
    organization = COALESCE($3::text, organization),
    enabled = COALESCE($4::boolean, enabled),
    updated_at = NOW()
WHERE asn = $1
RETURNING asn, name, organization, enabled, managed, sync_status, prefix_count,
          ipv4_count, ipv6_count, last_synced_at, last_error, created_at, updated_at`,
		asn, name, organization, enabled), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ASNSource{}, fmt.Errorf("ASN source not found")
	}
	if err != nil {
		return model.ASNSource{}, fmt.Errorf("update ASN source: %w", err)
	}
	return item, nil
}

func (s *Store) DeleteASNSource(ctx context.Context, asn int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM asn_sources WHERE asn = $1 AND NOT managed`, asn)
	if err != nil {
		return fmt.Errorf("delete ASN source: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var managed bool
		err := s.pool.QueryRow(ctx, `SELECT managed FROM asn_sources WHERE asn = $1`, asn).Scan(&managed)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("ASN source not found")
		}
		if err != nil {
			return err
		}
		if managed {
			return fmt.Errorf("managed Cloudflare ASN cannot be deleted; disable it instead")
		}
	}
	return nil
}

func (s *Store) ReplaceASNPrefixes(ctx context.Context, asn int64, prefixes []model.Prefix) (model.ASNSource, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.ASNSource{}, fmt.Errorf("begin AS%d prefix sync: %w", asn, err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE asn_prefixes SET active = FALSE WHERE asn = $1`, asn); err != nil {
		return model.ASNSource{}, fmt.Errorf("deactivate AS%d prefixes: %w", asn, err)
	}
	v4, v6 := 0, 0
	batch := &pgx.Batch{}
	for _, prefix := range prefixes {
		batch.Queue(`
INSERT INTO asn_prefixes (asn, cidr, ip_version, active, first_seen_at, last_seen_at)
VALUES ($1, $2::cidr, $3, TRUE, NOW(), NOW())
ON CONFLICT (asn, cidr) DO UPDATE SET ip_version = EXCLUDED.ip_version,
    active = TRUE, last_seen_at = NOW()`, asn, prefix.CIDR, prefix.IPVersion)
		if prefix.IPVersion == 4 {
			v4++
		} else {
			v6++
		}
	}
	results := tx.SendBatch(ctx, batch)
	for range prefixes {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return model.ASNSource{}, fmt.Errorf("upsert AS%d prefix: %w", asn, err)
		}
	}
	if err := results.Close(); err != nil {
		return model.ASNSource{}, fmt.Errorf("finish AS%d prefix batch: %w", asn, err)
	}
	now := time.Now().UTC()
	var item model.ASNSource
	err = scanASNSource(tx.QueryRow(ctx, `
UPDATE asn_sources SET sync_status = 'ok', prefix_count = $2, ipv4_count = $3,
    ipv6_count = $4, last_synced_at = $5, last_error = '', updated_at = NOW()
WHERE asn = $1
RETURNING asn, name, organization, enabled, managed, sync_status, prefix_count,
          ipv4_count, ipv6_count, last_synced_at, last_error, created_at, updated_at`,
		asn, len(prefixes), v4, v6, now), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ASNSource{}, fmt.Errorf("ASN source AS%d not found", asn)
	}
	if err != nil {
		return model.ASNSource{}, fmt.Errorf("record AS%d sync: %w", asn, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return model.ASNSource{}, fmt.Errorf("commit AS%d prefix sync: %w", asn, err)
	}
	return item, nil
}

func (s *Store) RecordASNError(ctx context.Context, asn int64, syncErr error) error {
	_, err := s.pool.Exec(ctx, `
UPDATE asn_sources SET sync_status = 'error', last_error = $2, updated_at = NOW()
WHERE asn = $1`, asn, truncate(syncErr.Error(), 2000))
	return err
}

func (s *Store) ListActivePrefixes(ctx context.Context, includeIPv6 bool) ([]model.Prefix, error) {
	query := `
WITH all_prefixes AS (
    SELECT cidr, ip_version, source, active, last_seen_at, 0 AS priority
    FROM cloudflare_prefixes WHERE active
    UNION ALL
    SELECT p.cidr, p.ip_version, 'asn:' || p.asn::text, p.active, p.last_seen_at, 1 AS priority
    FROM asn_prefixes p JOIN asn_sources a ON a.asn = p.asn
    WHERE p.active AND a.enabled
), deduplicated AS (
    SELECT DISTINCT ON (cidr) cidr, ip_version, source, active, last_seen_at
    FROM all_prefixes
    ORDER BY cidr, priority, last_seen_at DESC
)
SELECT cidr::text, ip_version, source, active, last_seen_at
FROM deduplicated WHERE TRUE`
	if !includeIPv6 {
		query += ` AND ip_version = 4`
	}
	query += ` ORDER BY ip_version, cidr`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list active prefixes: %w", err)
	}
	defer rows.Close()
	items := make([]model.Prefix, 0)
	for rows.Next() {
		var item model.Prefix
		if err := rows.Scan(&item.CIDR, &item.IPVersion, &item.Source, &item.Active, &item.LastSeen); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ReplaceColoLocations(ctx context.Context, locations []model.ColoLocation) (model.ColoSyncSummary, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.ColoSyncSummary{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE incoming_colos (code TEXT PRIMARY KEY, city TEXT, country TEXT, continent TEXT, status TEXT) ON COMMIT DROP`); err != nil {
		return model.ColoSyncSummary{}, err
	}
	for _, item := range locations {
		if _, err := tx.Exec(ctx, `INSERT INTO incoming_colos (code, city, country, continent, status) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (code) DO UPDATE SET city=EXCLUDED.city,country=EXCLUDED.country,continent=EXCLUDED.continent,status=EXCLUDED.status`, item.Code, item.City, item.Country, item.Continent, item.Status); err != nil {
			return model.ColoSyncSummary{}, fmt.Errorf("stage colo %s: %w", item.Code, err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM colo_locations WHERE code NOT IN (SELECT code FROM incoming_colos)`); err != nil {
		return model.ColoSyncSummary{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO colo_locations (code, city, country, continent, status, updated_at) SELECT code,city,country,continent,status,NOW() FROM incoming_colos ON CONFLICT (code) DO UPDATE SET city=EXCLUDED.city,country=EXCLUDED.country,continent=EXCLUDED.continent,status=EXCLUDED.status,updated_at=NOW()`); err != nil {
		return model.ColoSyncSummary{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.ColoSyncSummary{}, err
	}
	return model.ColoSyncSummary{Locations: len(locations), SyncedAt: time.Now().UTC()}, nil
}

func (s *Store) ListColoLocations(ctx context.Context) ([]model.ColoLocation, error) {
	rows, err := s.pool.Query(ctx, `SELECT code, city, country, continent, status, updated_at FROM colo_locations ORDER BY continent, country, city, code`)
	if err != nil {
		return nil, fmt.Errorf("list colo locations: %w", err)
	}
	defer rows.Close()
	items := make([]model.ColoLocation, 0)
	for rows.Next() {
		var item model.ColoLocation
		if err := rows.Scan(&item.Code, &item.City, &item.Country, &item.Continent, &item.Status, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanASNSource(row rowScanner, item *model.ASNSource) error {
	return row.Scan(&item.ASN, &item.Name, &item.Organization, &item.Enabled, &item.Managed,
		&item.Status, &item.PrefixCount, &item.IPv4Count, &item.IPv6Count,
		&item.LastSyncedAt, &item.LastError, &item.CreatedAt, &item.UpdatedAt)
}

func (s *Store) CreateScanJob(ctx context.Context, input model.CreateScanJobRequest, agentTargets []model.AgentScanTargets) (model.ScanJob, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.ScanJob{}, fmt.Errorf("begin create job: %w", err)
	}
	defer tx.Rollback(ctx)
	if len(agentTargets) == 0 {
		return model.ScanJob{}, fmt.Errorf("no scan agents selected")
	}

	var job model.ScanJob
	row := tx.QueryRow(ctx, `
INSERT INTO scan_jobs (name, kind, status, sampling_mode, scheme, hostname, path, port, attempts, timeout_ms,
    max_latency_ms, max_packet_loss, blacklist_minutes)
VALUES ($1, $2, 'pending', $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id::text, name, kind, status, sampling_mode, scheme, hostname, path, port, attempts, timeout_ms,
    max_latency_ms, max_packet_loss, blacklist_minutes, total_targets, completed_targets,
    success_targets, failed_targets, created_at, started_at, finished_at`,
		input.Name, input.Kind, input.SamplingMode, input.Scheme, input.Hostname, input.Path, input.Port, input.Attempts, input.TimeoutMS,
		input.MaxLatencyMS, input.MaxPacketLoss, input.BlacklistMinutes)
	if err := (jobRow{row}).ScanJob(&job); err != nil {
		return model.ScanJob{}, fmt.Errorf("insert scan job: %w", err)
	}

	type scheduledPrefix struct {
		agentID string
		prefix  string
	}
	inserted := 0
	scheduledPrefixes := make(map[string]scheduledPrefix)
	for _, agentSet := range agentTargets {
		if agentSet.AgentID == "" {
			return model.ScanJob{}, fmt.Errorf("scan target set is missing agent_id")
		}
		for _, target := range agentSet.Targets {
			if target.TargetIP == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `
INSERT INTO ip_targets (ip, source, enabled, updated_at)
VALUES ($1::inet, 'cloudflare_union', TRUE, NOW())
ON CONFLICT (ip) DO UPDATE SET enabled = TRUE, updated_at = NOW()`, target.TargetIP); err != nil {
				return model.ScanJob{}, fmt.Errorf("upsert target %s: %w", target.TargetIP, err)
			}
			tag, err := tx.Exec(ctx, `
INSERT INTO scan_tasks (job_id, preferred_agent_id, target_ip, target_prefix)
SELECT $1::uuid, $2::uuid, $3::inet, NULLIF($4::text, '')::cidr
WHERE $5 OR NOT EXISTS (
    SELECT 1 FROM blacklist_entries b
    WHERE b.agent_id = $2::uuid AND b.target_ip = $3::inet AND b.retry_after > NOW()
)
ON CONFLICT DO NOTHING`, job.ID, agentSet.AgentID, target.TargetIP, target.PrefixCIDR, input.IncludeBlocked)
			if err != nil {
				return model.ScanJob{}, fmt.Errorf("insert scan task: %w", err)
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			inserted++
			if input.SamplingMode == "league" && target.PrefixCIDR != "" {
				scheduledPrefixes[agentSet.AgentID+"|"+target.PrefixCIDR] = scheduledPrefix{agentID: agentSet.AgentID, prefix: target.PrefixCIDR}
			}
		}
	}
	if inserted == 0 {
		return model.ScanJob{}, fmt.Errorf("no tasks created; all selected targets may be blacklisted")
	}
	if _, err := tx.Exec(ctx, `UPDATE scan_jobs SET total_targets = $2 WHERE id = $1`, job.ID, inserted); err != nil {
		return model.ScanJob{}, err
	}
	for _, selected := range scheduledPrefixes {
		if _, err := tx.Exec(ctx, `
UPDATE prefix_league_entries SET last_scheduled_at = NOW(), updated_at = NOW()
WHERE agent_id = $1::uuid AND prefix_cidr = $2::cidr AND scheme = $3 AND hostname = $4
  AND path = $5 AND port = $6 AND attempts = $7 AND timeout_ms = $8`,
			selected.agentID, selected.prefix, input.Scheme, input.Hostname, input.Path,
			input.Port, input.Attempts, input.TimeoutMS); err != nil {
			return model.ScanJob{}, fmt.Errorf("mark league prefix scheduled: %w", err)
		}
	}
	job.TotalTargets = inserted
	if err := tx.Commit(ctx); err != nil {
		return model.ScanJob{}, fmt.Errorf("commit create job: %w", err)
	}
	return job, nil
}

func (s *Store) ListScanJobs(ctx context.Context, limit int) ([]model.ScanJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, scanJobSelect+` ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()
	items := make([]model.ScanJob, 0)
	for rows.Next() {
		var item model.ScanJob
		if err := scanJobRow(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetScanJob(ctx context.Context, id string) (model.ScanJob, error) {
	var item model.ScanJob
	if err := scanJobRow(s.pool.QueryRow(ctx, scanJobSelect+` WHERE id = $1`, id), &item); err != nil {
		return model.ScanJob{}, fmt.Errorf("get job: %w", err)
	}
	return item, nil
}

func (s *Store) StopScanJob(ctx context.Context, id string) (model.ScanJob, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.ScanJob{}, err
	}
	defer tx.Rollback(ctx)
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM scan_jobs WHERE id = $1 FOR UPDATE`, id).Scan(&status); err != nil {
		return model.ScanJob{}, fmt.Errorf("load job for stop: %w", err)
	}
	if status != "pending" && status != "running" {
		return model.ScanJob{}, fmt.Errorf("job cannot be stopped from status %s", status)
	}
	if _, err := tx.Exec(ctx, `UPDATE scan_tasks SET status = 'cancelled', lease_until = NULL WHERE job_id = $1 AND status IN ('pending','leased')`, id); err != nil {
		return model.ScanJob{}, fmt.Errorf("cancel remaining tasks: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE scan_jobs SET status = 'stopped', finished_at = NOW() WHERE id = $1`, id); err != nil {
		return model.ScanJob{}, fmt.Errorf("stop job: %w", err)
	}
	var item model.ScanJob
	if err := scanJobRow(tx.QueryRow(ctx, scanJobSelect+` WHERE id = $1`, id), &item); err != nil {
		return model.ScanJob{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.ScanJob{}, err
	}
	return item, nil
}

func (s *Store) ListScanSchedules(ctx context.Context) ([]model.ScanSchedule, error) {
	rows, err := s.pool.Query(ctx, scanScheduleSelect+` ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list scan schedules: %w", err)
	}
	defer rows.Close()
	items := make([]model.ScanSchedule, 0)
	for rows.Next() {
		var item model.ScanSchedule
		if err := scanScheduleRow(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetScanSchedule(ctx context.Context, id string) (model.ScanSchedule, error) {
	var item model.ScanSchedule
	if err := scanScheduleRow(s.pool.QueryRow(ctx, scanScheduleSelect+` WHERE id = $1`, id), &item); err != nil {
		return model.ScanSchedule{}, fmt.Errorf("get scan schedule: %w", err)
	}
	return item, nil
}

func (s *Store) CreateScanSchedule(ctx context.Context, input model.UpsertScanScheduleRequest, nextRun time.Time) (model.ScanSchedule, error) {
	agentIDs, err := json.Marshal(input.AgentIDs)
	if err != nil {
		return model.ScanSchedule{}, fmt.Errorf("encode schedule agents: %w", err)
	}
	var item model.ScanSchedule
	err = scanScheduleRow(s.pool.QueryRow(ctx, `
INSERT INTO scan_schedules (name, enabled, cron_expression, timezone, agent_ids, sampling_mode, target_count,
    scheme, hostname, path, port, attempts, timeout_ms, max_latency_ms, max_packet_loss,
    blacklist_minutes, include_ipv6, include_blocked, next_run_at)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
RETURNING id::text, name, enabled, cron_expression, timezone, agent_ids, sampling_mode, target_count,
    scheme, hostname, path, port, attempts, timeout_ms, max_latency_ms, max_packet_loss,
    blacklist_minutes, include_ipv6, include_blocked, next_run_at, last_run_at,
    last_job_id::text, last_error, created_at, updated_at`,
		input.Name, input.Enabled, input.CronExpression, input.Timezone, agentIDs, input.SamplingMode, input.TargetCount,
		input.Scheme, input.Hostname, input.Path, input.Port, input.Attempts, input.TimeoutMS,
		input.MaxLatencyMS, input.MaxPacketLoss, input.BlacklistMinutes, input.IncludeIPv6,
		input.IncludeBlocked, nextRun), &item)
	if err != nil {
		return model.ScanSchedule{}, fmt.Errorf("create scan schedule: %w", err)
	}
	return item, nil
}

func (s *Store) UpdateScanSchedule(ctx context.Context, id string, input model.UpsertScanScheduleRequest, nextRun time.Time) (model.ScanSchedule, error) {
	agentIDs, err := json.Marshal(input.AgentIDs)
	if err != nil {
		return model.ScanSchedule{}, fmt.Errorf("encode schedule agents: %w", err)
	}
	var item model.ScanSchedule
	err = scanScheduleRow(s.pool.QueryRow(ctx, `
UPDATE scan_schedules SET name = $2, enabled = $3, cron_expression = $4, timezone = $5,
    agent_ids = $6::jsonb, sampling_mode = $7, target_count = $8, scheme = $9, hostname = $10, path = $11,
    port = $12, attempts = $13, timeout_ms = $14, max_latency_ms = $15,
    max_packet_loss = $16, blacklist_minutes = $17, include_ipv6 = $18,
    include_blocked = $19, next_run_at = $20, last_error = '', updated_at = NOW()
WHERE id = $1
RETURNING id::text, name, enabled, cron_expression, timezone, agent_ids, sampling_mode, target_count,
    scheme, hostname, path, port, attempts, timeout_ms, max_latency_ms, max_packet_loss,
    blacklist_minutes, include_ipv6, include_blocked, next_run_at, last_run_at,
    last_job_id::text, last_error, created_at, updated_at`,
		id, input.Name, input.Enabled, input.CronExpression, input.Timezone, agentIDs,
		input.SamplingMode, input.TargetCount, input.Scheme, input.Hostname, input.Path, input.Port, input.Attempts,
		input.TimeoutMS, input.MaxLatencyMS, input.MaxPacketLoss, input.BlacklistMinutes,
		input.IncludeIPv6, input.IncludeBlocked, nextRun), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ScanSchedule{}, fmt.Errorf("scan schedule not found")
	}
	if err != nil {
		return model.ScanSchedule{}, fmt.Errorf("update scan schedule: %w", err)
	}
	return item, nil
}

func (s *Store) DeleteScanSchedule(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM scan_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete scan schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("scan schedule not found")
	}
	return nil
}

func (s *Store) ClaimDueScanSchedules(ctx context.Context, now, leaseUntil time.Time, limit int) ([]model.ScanSchedule, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
WITH picked AS (
    SELECT s.id
    FROM scan_schedules s
    WHERE s.enabled AND s.next_run_at <= $1
      AND NOT EXISTS (
          SELECT 1 FROM scan_jobs j
          WHERE j.id = s.last_job_id AND j.status IN ('pending', 'running')
      )
    ORDER BY s.next_run_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
UPDATE scan_schedules s SET next_run_at = $3, updated_at = NOW()
FROM picked WHERE s.id = picked.id
RETURNING s.id::text, s.name, s.enabled, s.cron_expression, s.timezone, s.agent_ids,
    s.sampling_mode, s.target_count, s.scheme, s.hostname, s.path, s.port, s.attempts, s.timeout_ms,
    s.max_latency_ms, s.max_packet_loss, s.blacklist_minutes, s.include_ipv6,
    s.include_blocked, s.next_run_at, s.last_run_at, s.last_job_id::text,
    s.last_error, s.created_at, s.updated_at`, now, limit, leaseUntil)
	if err != nil {
		return nil, fmt.Errorf("claim due scan schedules: %w", err)
	}
	defer rows.Close()
	items := make([]model.ScanSchedule, 0)
	for rows.Next() {
		var item model.ScanSchedule
		if err := scanScheduleRow(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RecordScanScheduleRun(ctx context.Context, id string, nextRun *time.Time, jobID *string, runErr error) error {
	lastError := ""
	if runErr != nil {
		lastError = truncate(runErr.Error(), 2000)
	}
	_, err := s.pool.Exec(ctx, `
UPDATE scan_schedules SET next_run_at = COALESCE($2::timestamptz, next_run_at),
    last_run_at = NOW(), last_job_id = $3::uuid, last_error = $4, updated_at = NOW()
WHERE id = $1`, id, nextRun, jobID, lastError)
	if err != nil {
		return fmt.Errorf("record scan schedule run: %w", err)
	}
	return nil
}

func (s *Store) ClaimTasks(ctx context.Context, agentID string, limit int) (*model.TaskBatch, error) {
	if limit <= 0 {
		limit = 32
	}
	if limit > 512 {
		limit = 512
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE scan_tasks SET status = 'pending', lease_until = NULL WHERE preferred_agent_id = $1 AND status = 'leased' AND lease_until < NOW()`, agentID); err != nil {
		return nil, err
	}
	var batch model.TaskBatch
	err = tx.QueryRow(ctx, `
SELECT j.id::text, j.name, j.scheme, j.hostname, j.path, j.port, j.attempts, j.timeout_ms
FROM scan_jobs j
WHERE j.status IN ('pending', 'running')
  AND EXISTS (SELECT 1 FROM scan_tasks t WHERE t.job_id = j.id AND t.preferred_agent_id = $1 AND t.status = 'pending')
ORDER BY j.created_at
LIMIT 1
FOR UPDATE SKIP LOCKED`, agentID).Scan(&batch.JobID, &batch.JobName, &batch.Scheme, &batch.Hostname, &batch.Path, &batch.Port, &batch.Attempts, &batch.TimeoutMS)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select claimable job: %w", err)
	}

	rows, err := tx.Query(ctx, `
WITH picked AS (
    SELECT id FROM scan_tasks
    WHERE job_id = $1 AND preferred_agent_id = $2 AND status = 'pending'
    ORDER BY id LIMIT $3 FOR UPDATE SKIP LOCKED
)
UPDATE scan_tasks t SET status = 'leased', lease_until = NOW() + INTERVAL '3 minutes'
FROM picked WHERE t.id = picked.id
RETURNING t.id, host(t.target_ip)`, batch.JobID, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("claim tasks: %w", err)
	}
	for rows.Next() {
		var task model.ScanTask
		if err := rows.Scan(&task.ID, &task.TargetIP); err != nil {
			rows.Close()
			return nil, err
		}
		batch.Tasks = append(batch.Tasks, task)
	}
	rows.Close()
	if len(batch.Tasks) == 0 {
		return nil, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE scan_jobs SET status = 'running', started_at = COALESCE(started_at, NOW()) WHERE id = $1`, batch.JobID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *Store) SubmitResults(ctx context.Context, batch model.ResultBatch) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var maxLatency, maxPacketLoss float64
	var blacklistMinutes int
	var jobStatus string
	if err := tx.QueryRow(ctx, `SELECT status, max_latency_ms, max_packet_loss, blacklist_minutes FROM scan_jobs WHERE id = $1 FOR UPDATE`, batch.JobID).Scan(&jobStatus, &maxLatency, &maxPacketLoss, &blacklistMinutes); err != nil {
		return fmt.Errorf("load job thresholds: %w", err)
	}
	if jobStatus == "stopped" {
		return tx.Commit(ctx)
	}
	for _, result := range batch.Results {
		var taskTarget, taskPrefix string
		err := tx.QueryRow(ctx, `
UPDATE scan_tasks SET status = 'completed', lease_until = NULL, completed_at = NOW()
WHERE id = $1 AND job_id = $2 AND preferred_agent_id = $3 AND status = 'leased'
RETURNING host(target_ip), COALESCE(target_prefix::text, '')`, result.TaskID, batch.JobID, batch.AgentID).Scan(&taskTarget, &taskPrefix)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("task %d does not belong to job or agent", result.TaskID)
		}
		if err != nil {
			return fmt.Errorf("complete task %d: %w", result.TaskID, err)
		}
		if result.TargetIP != taskTarget {
			return fmt.Errorf("task %d target mismatch", result.TaskID)
		}
		_, err = tx.Exec(ctx, `
INSERT INTO scan_results (job_id, task_id, agent_id, target_ip, target_prefix, available, latency_ms, packet_loss,
    tcp_connect_ms, tls_handshake_ms, ttfb_ms, total_ms, http_status, http_version, tls_version,
    colo, cf_ray, error_code, error_message, successful_tries, attempts, scanned_at)
VALUES ($1, $2, $3, $4::inet, NULLIF($5::text, '')::cidr, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, NOW())
ON CONFLICT (task_id) DO UPDATE SET target_prefix = EXCLUDED.target_prefix, available = EXCLUDED.available, latency_ms = EXCLUDED.latency_ms,
    packet_loss = EXCLUDED.packet_loss, tcp_connect_ms = EXCLUDED.tcp_connect_ms,
    tls_handshake_ms = EXCLUDED.tls_handshake_ms, ttfb_ms = EXCLUDED.ttfb_ms,
    total_ms = EXCLUDED.total_ms, http_status = EXCLUDED.http_status,
    http_version = EXCLUDED.http_version, tls_version = EXCLUDED.tls_version,
    colo = EXCLUDED.colo, cf_ray = EXCLUDED.cf_ray, error_code = EXCLUDED.error_code,
    error_message = EXCLUDED.error_message, successful_tries = EXCLUDED.successful_tries,
    attempts = EXCLUDED.attempts, scanned_at = NOW()`,
			batch.JobID, result.TaskID, batch.AgentID, taskTarget, taskPrefix, result.Available,
			result.LatencyMS, result.PacketLoss, result.TCPConnectMS, result.TLSHandshakeMS,
			result.TTFBMS, result.TotalMS, result.HTTPStatus, result.HTTPVersion, result.TLSVersion,
			result.Colo, result.CFRay, result.ErrorCode, truncate(result.ErrorMessage, 1000),
			result.SuccessfulTries, result.Attempts,
		)
		if err != nil {
			return fmt.Errorf("insert result for task %d: %w", result.TaskID, err)
		}

		reason := ""
		switch {
		case !result.Available:
			reason = result.ErrorCode
			if reason == "" {
				reason = "UNAVAILABLE"
			}
		case result.PacketLoss > maxPacketLoss:
			reason = "HIGH_PACKET_LOSS"
		case result.LatencyMS > maxLatency:
			reason = "HIGH_LATENCY"
		}
		if reason != "" {
			_, err = tx.Exec(ctx, `
INSERT INTO blacklist_entries (agent_id, target_ip, reason, failure_count, blocked_at, retry_after, updated_at)
VALUES ($1, $2::inet, $3, 1, NOW(), NOW() + ($4 * INTERVAL '1 minute'), NOW())
ON CONFLICT (agent_id, target_ip) DO UPDATE SET reason = EXCLUDED.reason,
    failure_count = blacklist_entries.failure_count + 1, blocked_at = NOW(),
    retry_after = NOW() + ($4 * INTERVAL '1 minute'), updated_at = NOW()`, batch.AgentID, taskTarget, reason, blacklistMinutes)
		} else {
			_, err = tx.Exec(ctx, `DELETE FROM blacklist_entries WHERE agent_id = $1 AND target_ip = $2::inet`, batch.AgentID, taskTarget)
		}
		if err != nil {
			return fmt.Errorf("update blacklist: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
UPDATE scan_jobs j SET
    completed_targets = stats.completed,
    success_targets = stats.success,
    failed_targets = stats.failed,
    status = CASE WHEN stats.completed >= j.total_targets THEN 'completed' ELSE 'running' END,
    finished_at = CASE WHEN stats.completed >= j.total_targets THEN NOW() ELSE NULL END
FROM (
    SELECT t.job_id, COUNT(*) FILTER (WHERE t.status = 'completed')::int AS completed,
           COUNT(*) FILTER (WHERE t.status = 'completed' AND r.available AND r.packet_loss <= j2.max_packet_loss AND r.latency_ms <= j2.max_latency_ms)::int AS success,
           COUNT(*) FILTER (WHERE t.status = 'completed' AND (NOT r.available OR r.packet_loss > j2.max_packet_loss OR r.latency_ms > j2.max_latency_ms))::int AS failed
    FROM scan_tasks t
    JOIN scan_jobs j2 ON j2.id = t.job_id
    LEFT JOIN scan_results r ON r.task_id = t.id
    WHERE t.job_id = $1
    GROUP BY t.job_id
) stats WHERE j.id = stats.job_id`, batch.JobID)
	if err != nil {
		return fmt.Errorf("update job progress: %w", err)
	}
	return tx.Commit(ctx)
}

func resultDataset(filter model.ResultFilter, includeGeo, includeJobFilter bool) (string, []any) {
	candidateConditions := []string{"1=1"}
	finalConditions := []string{"1=1"}
	args := make([]any, 0, 12)
	addCandidate := func(clause string, value any) {
		args = append(args, value)
		candidateConditions = append(candidateConditions, fmt.Sprintf(clause, len(args)))
	}
	addFinal := func(clause string, value any) {
		args = append(args, value)
		finalConditions = append(finalConditions, fmt.Sprintf(clause, len(args)))
	}
	if filter.AgentID != "" {
		addCandidate("r.agent_id = $%d::uuid", filter.AgentID)
	}
	if includeJobFilter && filter.JobID != "" {
		addCandidate("r.job_id = $%d::uuid", filter.JobID)
	}
	if filter.Region != "" {
		addCandidate("a.region = $%d", filter.Region)
	}
	if filter.Continent != "" {
		addCandidate("a.continent = $%d", filter.Continent)
	}
	if filter.TargetIP != "" {
		addCandidate("host(r.target_ip) ILIKE '%%' || $%d || '%%'", filter.TargetIP)
	}
	if filter.Since != nil {
		addCandidate("r.scanned_at >= $%d", *filter.Since)
	}
	if filter.Available != nil {
		addFinal("q.available = $%d", *filter.Available)
	}
	if includeGeo {
		if filter.Colo != "" {
			addFinal("q.colo = $%d", strings.ToUpper(filter.Colo))
		}
		if filter.ColoCity != "" {
			addFinal("q.colo_city = $%d", filter.ColoCity)
		}
		if filter.ColoCountry != "" {
			addFinal("q.colo_country = $%d", filter.ColoCountry)
		}
		if filter.ColoContinent != "" {
			addFinal("q.colo_continent = $%d", filter.ColoContinent)
		}
	}

	rankExpression := "1::bigint"
	selectedWhere := "TRUE"
	if filter.View == "latest" {
		rankExpression = `ROW_NUMBER() OVER (
            PARTITION BY r.agent_id, r.target_ip, j.scheme, j.hostname, j.path, j.port, j.attempts, j.timeout_ms
            ORDER BY r.scanned_at DESC, r.id DESC
        )`
		selectedWhere = "result_rank = 1"
	}

	query := `
WITH result_candidates AS (
    SELECT r.id, r.job_id::text AS job_id, j.name AS job_name, j.kind AS job_kind,
           j.created_at AS job_created_at, r.agent_id::text AS agent_id, a.name AS agent_name,
           a.region, a.continent, host(r.target_ip) AS target_ip, r.available,
           r.latency_ms, r.packet_loss, r.tcp_connect_ms, r.tls_handshake_ms,
           r.ttfb_ms, r.total_ms, r.http_status, r.http_version, r.tls_version,
           r.colo, COALESCE(cl.city, '') AS colo_city,
           COALESCE(cl.country, '') AS colo_country,
           COALESCE(cl.continent, '') AS colo_continent,
           r.cf_ray, r.error_code, r.scanned_at,
           ` + rankExpression + ` AS result_rank
    FROM scan_results r
    JOIN agents a ON a.id = r.agent_id
    JOIN scan_jobs j ON j.id = r.job_id
    LEFT JOIN colo_locations cl ON cl.code = r.colo
    WHERE ` + strings.Join(candidateConditions, " AND ") + `
), result_selected AS (
    SELECT * FROM result_candidates WHERE ` + selectedWhere + `
), result_filtered AS (
    SELECT * FROM result_selected q WHERE ` + strings.Join(finalConditions, " AND ") + `
)`
	return query, args
}

func resultOrder(filter model.ResultFilter) string {
	order := "ASC"
	if strings.EqualFold(filter.Order, "desc") {
		order = "DESC"
	}
	switch filter.Sort {
	case "target_ip":
		return "target_ip " + order + ", scanned_at DESC, id DESC"
	case "agent_name":
		return "agent_name " + order + ", scanned_at DESC, id DESC"
	case "colo":
		return "colo " + order + ", scanned_at DESC, id DESC"
	case "available":
		return "available " + order + ", scanned_at DESC, id DESC"
	case "packet_loss":
		return "packet_loss " + order + ", scanned_at DESC, id DESC"
	case "http_status":
		return "http_status " + order + ", scanned_at DESC, id DESC"
	case "scanned_at":
		return "scanned_at " + order + ", id DESC"
	default:
		// Failed rows always remain behind usable results when sorting latency.
		return "available DESC, latency_ms " + order + ", scanned_at DESC, id DESC"
	}
}

func (s *Store) ListResults(ctx context.Context, filter model.ResultFilter) (model.ResultPage, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 200 {
		filter.PageSize = 50
	}
	if filter.View != "history" {
		filter.View = "latest"
	}
	countFilter := filter
	countFilter.Available = nil
	countBase, countArgs := resultDataset(countFilter, true, true)
	var counts model.ResultStatusCounts
	if err := s.pool.QueryRow(ctx, countBase+` SELECT COUNT(*), COUNT(*) FILTER (WHERE available), COUNT(*) FILTER (WHERE NOT available) FROM result_filtered`, countArgs...).Scan(&counts.All, &counts.Available, &counts.Failed); err != nil {
		return model.ResultPage{}, fmt.Errorf("count results: %w", err)
	}
	total := counts.All
	if filter.Available != nil {
		if *filter.Available {
			total = counts.Available
		} else {
			total = counts.Failed
		}
	}

	base, args := resultDataset(filter, true, true)
	page := model.ResultPage{
		Items:    make([]model.ScanResult, 0),
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Counts:   counts,
	}
	if total > 0 {
		page.TotalPages = int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))
	}

	pageArgs := append(append([]any{}, args...), filter.PageSize, (filter.Page-1)*filter.PageSize)
	limitPosition, offsetPosition := len(args)+1, len(args)+2
	query := base + fmt.Sprintf(`
SELECT id, job_id, job_name, agent_id, agent_name, region, continent, target_ip,
       available, latency_ms, packet_loss, tcp_connect_ms, tls_handshake_ms,
       ttfb_ms, total_ms, http_status, http_version, tls_version, colo,
       colo_city, colo_country, colo_continent, cf_ray, error_code, scanned_at
FROM result_filtered
ORDER BY %s
LIMIT $%d OFFSET $%d`, resultOrder(filter), limitPosition, offsetPosition)
	rows, err := s.pool.Query(ctx, query, pageArgs...)
	if err != nil {
		return model.ResultPage{}, fmt.Errorf("list results: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item model.ScanResult
		if err := rows.Scan(&item.ID, &item.JobID, &item.JobName, &item.AgentID, &item.AgentName,
			&item.Region, &item.Continent, &item.TargetIP, &item.Available, &item.LatencyMS,
			&item.PacketLoss, &item.TCPConnectMS, &item.TLSHandshakeMS, &item.TTFBMS,
			&item.TotalMS, &item.HTTPStatus, &item.HTTPVersion, &item.TLSVersion,
			&item.Colo, &item.ColoCity, &item.ColoCountry, &item.ColoContinent,
			&item.CFRay, &item.ErrorCode, &item.ScannedAt); err != nil {
			return model.ResultPage{}, err
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return model.ResultPage{}, err
	}
	return page, nil
}

func (s *Store) ListResultColoFacets(ctx context.Context, filter model.ResultFilter) ([]model.ResultColoFacet, error) {
	if filter.View != "history" {
		filter.View = "latest"
	}
	base, args := resultDataset(filter, false, true)
	query := base + `
SELECT colo, colo_city, colo_country, colo_continent, COUNT(*)::int
FROM result_filtered
WHERE colo <> '' AND colo_city <> '' AND colo_country <> '' AND colo_continent <> ''
GROUP BY colo, colo_city, colo_country, colo_continent
HAVING COUNT(*) > 0
ORDER BY colo_continent, colo_country, colo_city, colo`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list result colo facets: %w", err)
	}
	defer rows.Close()
	items := make([]model.ResultColoFacet, 0, 128)
	for rows.Next() {
		var item model.ResultColoFacet
		if err := rows.Scan(&item.Code, &item.City, &item.Country, &item.Continent, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListResultJobFacets(ctx context.Context, filter model.ResultFilter) ([]model.ResultJobFacet, error) {
	if filter.View != "history" {
		filter.View = "latest"
	}
	base, args := resultDataset(filter, true, false)
	query := base + `
SELECT job_id, MAX(job_name), MAX(job_kind), COUNT(*)::int, MAX(job_created_at)
FROM result_filtered
GROUP BY job_id
ORDER BY MAX(job_created_at) DESC`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list result job facets: %w", err)
	}
	defer rows.Close()
	items := make([]model.ResultJobFacet, 0, 64)
	for rows.Next() {
		var item model.ResultJobFacet
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Count, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListBlacklist(ctx context.Context, limit int) ([]model.BlacklistEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
SELECT b.agent_id::text, a.name, a.region, a.continent, host(b.target_ip), b.reason,
       b.failure_count, b.blocked_at, b.retry_after, b.updated_at
FROM blacklist_entries b JOIN agents a ON a.id = b.agent_id
ORDER BY b.retry_after, b.failure_count DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list blacklist: %w", err)
	}
	defer rows.Close()
	items := make([]model.BlacklistEntry, 0)
	for rows.Next() {
		var item model.BlacklistEntry
		if err := rows.Scan(&item.AgentID, &item.AgentName, &item.Region, &item.Continent,
			&item.TargetIP, &item.Reason, &item.FailureCount, &item.BlockedAt,
			&item.RetryAfter, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetBlacklistRecheckSettings(ctx context.Context) (model.BlacklistRecheckSettings, error) {
	var item model.BlacklistRecheckSettings
	err := s.pool.QueryRow(ctx, `
SELECT enabled, cron_expression, timezone, due_only, fraction, max_targets, skip_if_running,
       attempts, timeout_ms, max_latency_ms, max_packet_loss, retry_minutes, next_run_at,
       last_run_at, last_error, updated_at,
       (SELECT COUNT(*)::int FROM blacklist_entries b WHERE NOT br.due_only OR b.retry_after <= NOW())
FROM blacklist_recheck_settings br WHERE key = 'default'`).Scan(
		&item.Enabled, &item.CronExpression, &item.Timezone, &item.DueOnly, &item.Fraction,
		&item.MaxTargets, &item.SkipIfRunning, &item.Attempts, &item.TimeoutMS,
		&item.MaxLatencyMS, &item.MaxPacketLoss, &item.RetryMinutes, &item.NextRunAt,
		&item.LastRunAt, &item.LastError, &item.UpdatedAt, &item.EligibleTargets,
	)
	if err != nil {
		return model.BlacklistRecheckSettings{}, fmt.Errorf("get blacklist recheck settings: %w", err)
	}
	return item, nil
}

func (s *Store) UpdateBlacklistRecheckSettings(ctx context.Context, input model.UpdateBlacklistRecheckSettingsRequest, nextRun time.Time) (model.BlacklistRecheckSettings, error) {
	_, err := s.pool.Exec(ctx, `
UPDATE blacklist_recheck_settings SET enabled = $1, cron_expression = $2, timezone = $3,
    due_only = $4, fraction = $5, max_targets = $6, skip_if_running = $7,
    attempts = $8, timeout_ms = $9, max_latency_ms = $10, max_packet_loss = $11,
    retry_minutes = $12, next_run_at = $13, last_error = '', updated_at = NOW()
WHERE key = 'default'`, input.Enabled, input.CronExpression, input.Timezone, input.DueOnly,
		input.Fraction, input.MaxTargets, input.SkipIfRunning, input.Attempts, input.TimeoutMS,
		input.MaxLatencyMS, input.MaxPacketLoss, input.RetryMinutes, nextRun)
	if err != nil {
		return model.BlacklistRecheckSettings{}, fmt.Errorf("update blacklist recheck settings: %w", err)
	}
	return s.GetBlacklistRecheckSettings(ctx)
}

func (s *Store) ClaimDueBlacklistRecheck(ctx context.Context, now, leaseUntil time.Time) (*model.BlacklistRecheckSettings, error) {
	tag, err := s.pool.Exec(ctx, `
UPDATE blacklist_recheck_settings SET next_run_at = $2, updated_at = NOW()
WHERE key = 'default' AND enabled AND next_run_at <= $1`, now, leaseUntil)
	if err != nil {
		return nil, fmt.Errorf("claim blacklist recheck: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	item, err := s.GetBlacklistRecheckSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) RecordBlacklistRecheckRun(ctx context.Context, nextRun *time.Time, runErr error) error {
	lastError := ""
	if runErr != nil {
		lastError = truncate(runErr.Error(), 2000)
	}
	_, err := s.pool.Exec(ctx, `
UPDATE blacklist_recheck_settings SET next_run_at = COALESCE($1::timestamptz, next_run_at),
    last_run_at = NOW(), last_error = $2, updated_at = NOW() WHERE key = 'default'`, nextRun, lastError)
	if err != nil {
		return fmt.Errorf("record blacklist recheck run: %w", err)
	}
	return nil
}

func (s *Store) CreateBlacklistRechecks(ctx context.Context, settings model.BlacklistRecheckSettings) (model.BlacklistRecheckResult, error) {
	result := model.BlacklistRecheckResult{JobIDs: []string{}}
	if settings.Fraction <= 0 || settings.Fraction > 1 {
		settings.Fraction = 0.5
	}
	if settings.MaxTargets <= 0 || settings.MaxTargets > 5000 {
		settings.MaxTargets = 500
	}
	if settings.Attempts < 1 || settings.Attempts > 10 {
		settings.Attempts = 3
	}
	if settings.TimeoutMS < 500 || settings.TimeoutMS > 30000 {
		settings.TimeoutMS = 5000
	}
	if settings.MaxLatencyMS <= 0 {
		settings.MaxLatencyMS = 1000
	}
	if settings.MaxPacketLoss < 0 || settings.MaxPacketLoss > 100 {
		settings.MaxPacketLoss = 50
	}
	if settings.RetryMinutes < 1 || settings.RetryMinutes > 10080 {
		settings.RetryMinutes = 120
	}
	if settings.SkipIfRunning {
		var active int
		if err := s.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM scan_jobs WHERE kind = 'blacklist_recheck' AND status IN ('pending', 'running')`).Scan(&active); err != nil {
			return result, fmt.Errorf("check active blacklist rechecks: %w", err)
		}
		if active > 0 {
			result.Skipped = true
			result.Reason = "上一轮黑名单复查仍在等待或运行"
			return result, nil
		}
	}

	condition := ""
	if settings.DueOnly {
		condition = "WHERE b.retry_after <= NOW()"
	}
	candidateLimit := int(math.Ceil(float64(settings.MaxTargets) / settings.Fraction))
	if candidateLimit < settings.MaxTargets {
		candidateLimit = settings.MaxTargets
	}
	if candidateLimit > 20000 {
		candidateLimit = 20000
	}
	rows, err := s.pool.Query(ctx, `
SELECT b.agent_id::text, a.name, host(b.target_ip), b.failure_count
FROM blacklist_entries b JOIN agents a ON a.id = b.agent_id `+condition+`
ORDER BY b.agent_id, b.retry_after, b.failure_count DESC LIMIT $1`, candidateLimit)
	if err != nil {
		return result, fmt.Errorf("select blacklist rechecks: %w", err)
	}
	type candidate struct {
		agentID, agentName, ip string
		failures               int
	}
	groups := map[string][]candidate{}
	order := make([]string, 0)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.agentID, &item.agentName, &item.ip, &item.failures); err != nil {
			rows.Close()
			return result, err
		}
		if _, exists := groups[item.agentID]; !exists {
			order = append(order, item.agentID)
		}
		groups[item.agentID] = append(groups[item.agentID], item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, err
	}

	remaining := settings.MaxTargets
	for _, agentID := range order {
		if remaining <= 0 {
			break
		}
		candidates := groups[agentID]
		count := int(math.Ceil(float64(len(candidates)) * settings.Fraction))
		if count < 1 && len(candidates) > 0 {
			count = 1
		}
		if count > len(candidates) {
			count = len(candidates)
		}
		if count > remaining {
			count = remaining
		}
		if count == 0 {
			continue
		}
		selected := candidates[:count]
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return result, err
		}
		var jobID string
		name := "黑名单复查"
		if selected[0].agentName != "" {
			name += " · " + selected[0].agentName
		}
		err = tx.QueryRow(ctx, `
INSERT INTO scan_jobs (name, kind, status, sampling_mode, scheme, hostname, path, port, attempts, timeout_ms,
    max_latency_ms, max_packet_loss, blacklist_minutes, total_targets)
VALUES ($1, 'blacklist_recheck', 'pending', 'count', 'https', 'cloudflare.com', '/cdn-cgi/trace', 443,
        $2, $3, $4, $5, $6, $7)
RETURNING id::text`, name, settings.Attempts, settings.TimeoutMS, settings.MaxLatencyMS,
			settings.MaxPacketLoss, settings.RetryMinutes, len(selected)).Scan(&jobID)
		if err != nil {
			tx.Rollback(ctx)
			return result, err
		}
		for _, item := range selected {
			if _, err := tx.Exec(ctx, `INSERT INTO scan_tasks (job_id, preferred_agent_id, target_ip) VALUES ($1, $2, $3::inet) ON CONFLICT DO NOTHING`, jobID, agentID, item.ip); err != nil {
				tx.Rollback(ctx)
				return result, err
			}
			if _, err := tx.Exec(ctx, `UPDATE blacklist_entries SET retry_after = NOW() + ($3 * INTERVAL '1 minute'), updated_at = NOW() WHERE agent_id = $1 AND target_ip = $2::inet`, agentID, item.ip, settings.RetryMinutes); err != nil {
				tx.Rollback(ctx)
				return result, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return result, err
		}
		result.Jobs++
		result.Targets += len(selected)
		result.JobIDs = append(result.JobIDs, jobID)
		remaining -= len(selected)
	}
	if result.Targets == 0 {
		result.Skipped = true
		result.Reason = "当前没有符合复查条件的黑名单目标"
	}
	return result, nil
}

func (s *Store) ListSourceSyncSchedules(ctx context.Context) ([]model.SourceSyncSchedule, error) {
	rows, err := s.pool.Query(ctx, sourceSyncScheduleSelect+` ORDER BY source`)
	if err != nil {
		return nil, fmt.Errorf("list source sync schedules: %w", err)
	}
	defer rows.Close()
	items := make([]model.SourceSyncSchedule, 0, 2)
	for rows.Next() {
		var item model.SourceSyncSchedule
		if err := scanSourceSyncSchedule(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetSourceSyncSchedule(ctx context.Context, source string) (model.SourceSyncSchedule, error) {
	var item model.SourceSyncSchedule
	if err := scanSourceSyncSchedule(s.pool.QueryRow(ctx, sourceSyncScheduleSelect+` WHERE source = $1`, source), &item); err != nil {
		return model.SourceSyncSchedule{}, fmt.Errorf("get source sync schedule: %w", err)
	}
	return item, nil
}

func (s *Store) UpdateSourceSyncSchedule(ctx context.Context, source string, input model.UpdateSourceSyncScheduleRequest, nextRun time.Time) (model.SourceSyncSchedule, error) {
	tag, err := s.pool.Exec(ctx, `
UPDATE source_sync_schedules SET enabled = $2, cron_expression = $3, timezone = $4,
    run_on_startup = $5, next_run_at = $6, last_error = '', updated_at = NOW()
WHERE source = $1`, source, input.Enabled, input.CronExpression, input.Timezone, input.RunOnStartup, nextRun)
	if err != nil {
		return model.SourceSyncSchedule{}, fmt.Errorf("update source sync schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return model.SourceSyncSchedule{}, fmt.Errorf("source sync schedule not found")
	}
	return s.GetSourceSyncSchedule(ctx, source)
}

func (s *Store) ClaimDueSourceSyncSchedules(ctx context.Context, now, leaseUntil time.Time, limit int) ([]model.SourceSyncSchedule, error) {
	if limit <= 0 || limit > 10 {
		limit = 2
	}
	rows, err := s.pool.Query(ctx, `
WITH picked AS (
    SELECT s.source FROM source_sync_schedules s
    WHERE s.enabled AND s.next_run_at <= $1
      AND NOT EXISTS (
          SELECT 1 FROM automation_runs ar
          WHERE ar.automation_type = 'source_sync' AND ar.automation_key = s.source
            AND ar.status = 'running' AND ar.started_at > NOW() - INTERVAL '15 minutes'
      )
    ORDER BY s.next_run_at LIMIT $3 FOR UPDATE SKIP LOCKED
)
UPDATE source_sync_schedules s SET next_run_at = $2, updated_at = NOW()
FROM picked WHERE s.source = picked.source
RETURNING s.source, s.name, s.enabled, s.cron_expression, s.timezone, s.run_on_startup,
          s.next_run_at, s.last_run_at, s.last_error, s.updated_at`, now, leaseUntil, limit)
	if err != nil {
		return nil, fmt.Errorf("claim source sync schedules: %w", err)
	}
	defer rows.Close()
	items := make([]model.SourceSyncSchedule, 0, limit)
	for rows.Next() {
		var item model.SourceSyncSchedule
		if err := scanSourceSyncSchedule(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RecordSourceSyncRun(ctx context.Context, source string, nextRun *time.Time, runErr error) error {
	lastError := ""
	if runErr != nil {
		lastError = truncate(runErr.Error(), 2000)
	}
	_, err := s.pool.Exec(ctx, `
UPDATE source_sync_schedules SET next_run_at = COALESCE($2::timestamptz, next_run_at),
    last_run_at = NOW(), last_error = $3, updated_at = NOW() WHERE source = $1`, source, nextRun, lastError)
	if err != nil {
		return fmt.Errorf("record source sync run: %w", err)
	}
	return nil
}

func (s *Store) StartAutomationRun(ctx context.Context, input model.StartAutomationRunRequest) (model.AutomationRun, error) {
	if len(input.ConfigSnapshot) == 0 {
		input.ConfigSnapshot = json.RawMessage(`{}`)
	}
	var item model.AutomationRun
	var config, summary []byte
	err := s.pool.QueryRow(ctx, `
INSERT INTO automation_runs (automation_type, automation_key, name, trigger, config_snapshot)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING id::text, automation_type, automation_key, name, trigger, status,
          config_snapshot, summary, error, started_at, finished_at`, input.AutomationType,
		input.AutomationKey, input.Name, input.Trigger, input.ConfigSnapshot).Scan(
		&item.ID, &item.AutomationType, &item.AutomationKey, &item.Name, &item.Trigger,
		&item.Status, &config, &summary, &item.Error, &item.StartedAt, &item.FinishedAt)
	if err != nil {
		return model.AutomationRun{}, fmt.Errorf("start automation run: %w", err)
	}
	item.ConfigSnapshot, item.Summary = config, summary
	return item, nil
}

func (s *Store) FinishAutomationRun(ctx context.Context, id, status string, summary json.RawMessage, runErr error) error {
	if status == "" {
		status = "completed"
	}
	if len(summary) == 0 {
		summary = json.RawMessage(`{}`)
	}
	errorText := ""
	if runErr != nil {
		errorText = truncate(runErr.Error(), 4000)
		status = "failed"
	}
	_, err := s.pool.Exec(ctx, `
UPDATE automation_runs SET status = $2, summary = $3::jsonb, error = $4, finished_at = NOW()
WHERE id = $1`, id, status, summary, errorText)
	if err != nil {
		return fmt.Errorf("finish automation run: %w", err)
	}
	return nil
}

func (s *Store) ListAutomationRuns(ctx context.Context, limit int) ([]model.AutomationRun, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
SELECT id::text, automation_type, automation_key, name, trigger, status,
       config_snapshot, summary, error, started_at, finished_at
FROM automation_runs ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list automation runs: %w", err)
	}
	defer rows.Close()
	items := make([]model.AutomationRun, 0)
	for rows.Next() {
		var item model.AutomationRun
		var config, summary []byte
		if err := rows.Scan(&item.ID, &item.AutomationType, &item.AutomationKey, &item.Name,
			&item.Trigger, &item.Status, &config, &summary, &item.Error, &item.StartedAt,
			&item.FinishedAt); err != nil {
			return nil, err
		}
		item.ConfigSnapshot, item.Summary = config, summary
		items = append(items, item)
	}
	return items, rows.Err()
}

const sourceSyncScheduleSelect = `
SELECT source, name, enabled, cron_expression, timezone, run_on_startup,
       next_run_at, last_run_at, last_error, updated_at
FROM source_sync_schedules`

func scanSourceSyncSchedule(row rowScanner, item *model.SourceSyncSchedule) error {
	return row.Scan(&item.Source, &item.Name, &item.Enabled, &item.CronExpression,
		&item.Timezone, &item.RunOnStartup, &item.NextRunAt, &item.LastRunAt,
		&item.LastError, &item.UpdatedAt)
}

const scanScheduleSelect = `
SELECT id::text, name, enabled, cron_expression, timezone, agent_ids, sampling_mode, target_count,
       scheme, hostname, path, port, attempts, timeout_ms, max_latency_ms, max_packet_loss,
       blacklist_minutes, include_ipv6, include_blocked, next_run_at, last_run_at,
       last_job_id::text, last_error, created_at, updated_at
FROM scan_schedules`

func scanScheduleRow(row rowScanner, item *model.ScanSchedule) error {
	var agentIDs []byte
	if err := row.Scan(&item.ID, &item.Name, &item.Enabled, &item.CronExpression,
		&item.Timezone, &agentIDs, &item.SamplingMode, &item.TargetCount, &item.Scheme, &item.Hostname,
		&item.Path, &item.Port, &item.Attempts, &item.TimeoutMS, &item.MaxLatencyMS,
		&item.MaxPacketLoss, &item.BlacklistMinutes, &item.IncludeIPv6,
		&item.IncludeBlocked, &item.NextRunAt, &item.LastRunAt, &item.LastJobID,
		&item.LastError, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return err
	}
	if len(agentIDs) == 0 {
		item.AgentIDs = []string{}
		return nil
	}
	if err := json.Unmarshal(agentIDs, &item.AgentIDs); err != nil {
		return fmt.Errorf("decode schedule agents: %w", err)
	}
	return nil
}

const scanJobSelect = `
SELECT id::text, name, kind, status, sampling_mode, scheme, hostname, path, port, attempts, timeout_ms,
       max_latency_ms, max_packet_loss, blacklist_minutes, total_targets, completed_targets,
       success_targets, failed_targets,
       CASE WHEN total_targets > 0 THEN completed_targets::float8 / total_targets * 100 ELSE 0 END,
       created_at, started_at, finished_at
FROM scan_jobs`

type rowScanner interface{ Scan(...any) error }

func scanJobRow(row rowScanner, item *model.ScanJob) error {
	return row.Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.SamplingMode, &item.Scheme, &item.Hostname,
		&item.Path, &item.Port, &item.Attempts, &item.TimeoutMS, &item.MaxLatencyMS,
		&item.MaxPacketLoss, &item.BlacklistMinutes, &item.TotalTargets,
		&item.CompletedTargets, &item.SuccessTargets, &item.FailedTargets, &item.Progress,
		&item.CreatedAt, &item.StartedAt, &item.FinishedAt)
}

// ScanJob adds a typed helper to pgx.Row without leaking query details above.
type jobRow struct{ pgx.Row }

func (r jobRow) ScanJob(item *model.ScanJob) error {
	return r.Scan(&item.ID, &item.Name, &item.Kind, &item.Status, &item.SamplingMode, &item.Scheme, &item.Hostname,
		&item.Path, &item.Port, &item.Attempts, &item.TimeoutMS, &item.MaxLatencyMS,
		&item.MaxPacketLoss, &item.BlacklistMinutes, &item.TotalTargets,
		&item.CompletedTargets, &item.SuccessTargets, &item.FailedTargets,
		&item.CreatedAt, &item.StartedAt, &item.FinishedAt)
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
