package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/3011/cfscan/v2/internal/league"
	"github.com/3011/cfscan/v2/internal/model"
	storepkg "github.com/3011/cfscan/v2/internal/store"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ResolveScanAgentIDs(ctx context.Context, requested []string) ([]string, error) {
	if len(requested) == 0 {
		rows, err := s.pool.Query(ctx, `SELECT id::text FROM agents WHERE last_seen_at >= NOW() - INTERVAL '45 seconds' ORDER BY name`)
		if err != nil {
			return nil, fmt.Errorf("list online scan agents: %w", err)
		}
		defer rows.Close()
		items := make([]string, 0)
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			items = append(items, id)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("no online agents selected")
		}
		return items, nil
	}

	unique := make([]string, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, id := range requested {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text FROM agents WHERE id::text = ANY($1::text[])`, unique)
	if err != nil {
		return nil, fmt.Errorf("validate scan agents: %w", err)
	}
	defer rows.Close()
	found := make(map[string]struct{}, len(unique))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range unique {
		if _, exists := found[id]; !exists {
			return nil, fmt.Errorf("scan agent %s not found", id)
		}
	}
	return unique, nil
}

func (s *Store) PlanLeagueTargets(
	ctx context.Context,
	input model.CreateScanJobRequest,
	prefixes []model.Prefix,
	agentIDs []string,
	seed string,
	force bool,
) ([]model.AgentScanTargets, error) {
	if len(agentIDs) == 0 {
		return nil, fmt.Errorf("no scan agents selected")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin league planning: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
UPDATE prefix_league_entries SET active = FALSE, updated_at = NOW()
WHERE agent_id::text = ANY($1::text[]) AND scheme = $2 AND hostname = $3 AND path = $4
  AND port = $5 AND attempts = $6 AND timeout_ms = $7`,
		agentIDs, input.Scheme, input.Hostname, input.Path, input.Port, input.Attempts, input.TimeoutMS); err != nil {
		return nil, fmt.Errorf("deactivate stale league prefixes: %w", err)
	}

	batch := &pgx.Batch{}
	for _, agentID := range agentIDs {
		for _, prefix := range prefixes {
			if !prefix.Active || (!input.IncludeIPv6 && prefix.IPVersion == 6) {
				continue
			}
			batch.Queue(`
INSERT INTO prefix_league_entries (agent_id, prefix_cidr, scheme, hostname, path, port, attempts, timeout_ms, active)
VALUES ($1::uuid, $2::cidr, $3, $4, $5, $6, $7, $8, TRUE)
ON CONFLICT (agent_id, prefix_cidr, scheme, hostname, path, port, attempts, timeout_ms)
DO UPDATE SET active = TRUE, updated_at = NOW()`,
				agentID, prefix.CIDR, input.Scheme, input.Hostname, input.Path, input.Port, input.Attempts, input.TimeoutMS)
		}
	}
	results := tx.SendBatch(ctx, batch)
	if err := results.Close(); err != nil {
		return nil, fmt.Errorf("upsert league prefixes: %w", err)
	}

	entries, err := loadLeagueEntries(ctx, tx, input, agentIDs)
	if err != nil {
		return nil, err
	}
	thresholds := league.Thresholds{MaxLatencyMS: input.MaxLatencyMS, MaxPacketLoss: input.MaxPacketLoss}
	for index := range entries {
		entries[index] = league.Evaluate(entries[index], thresholds)
	}
	if err := saveLeagueEntries(ctx, tx, entries); err != nil {
		return nil, err
	}
	candidateItems, err := loadLeagueCandidates(ctx, tx, input, agentIDs, 2)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit league planning: %w", err)
	}

	entriesByAgent := make(map[string][]model.PrefixLeagueEntry, len(agentIDs))
	candidatesByAgent := make(map[string]map[string][]model.LeagueCandidate, len(agentIDs))
	for _, entry := range entries {
		entriesByAgent[entry.AgentID] = append(entriesByAgent[entry.AgentID], entry)
	}
	for _, candidate := range candidateItems {
		byPrefix := candidatesByAgent[candidate.AgentID]
		if byPrefix == nil {
			byPrefix = make(map[string][]model.LeagueCandidate)
			candidatesByAgent[candidate.AgentID] = byPrefix
		}
		byPrefix[candidate.PrefixCIDR] = append(byPrefix[candidate.PrefixCIDR], candidate)
	}

	now := time.Now().UTC()
	planned := make([]model.AgentScanTargets, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		items := league.PlanAgent(entriesByAgent[agentID], candidatesByAgent[agentID], league.PlanOptions{
			Now: now, Seed: seed + "|" + agentID, Budget: input.TargetCount, Force: force,
		})
		planned = append(planned, model.AgentScanTargets{AgentID: agentID, Targets: items})
	}
	return planned, nil
}

func loadLeagueEntries(ctx context.Context, tx pgx.Tx, input model.CreateScanJobRequest, agentIDs []string) ([]model.PrefixLeagueEntry, error) {
	rows, err := tx.Query(ctx, `
WITH stats_7d AS (
    SELECT r.agent_id, r.target_prefix,
           COUNT(*)::int AS sample_count,
           COUNT(DISTINCT r.target_ip)::int AS distinct_ip_count,
           COALESCE(AVG(CASE WHEN r.available THEN 100.0 ELSE 0.0 END), 0)::float8 AS availability_rate,
           COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.latency_ms) FILTER (WHERE r.available), 0)::float8 AS latency_p95_ms,
           COALESCE(AVG(r.packet_loss), 0)::float8 AS packet_loss_avg,
           MAX(r.scanned_at) AS last_result_at
    FROM scan_results r
    JOIN scan_jobs j ON j.id = r.job_id
    WHERE r.target_prefix IS NOT NULL AND r.agent_id::text = ANY($1::text[])
      AND r.scanned_at >= NOW() - INTERVAL '7 days'
      AND j.scheme = $2 AND j.hostname = $3 AND j.path = $4 AND j.port = $5
      AND j.attempts = $6 AND j.timeout_ms = $7
    GROUP BY r.agent_id, r.target_prefix
), stats_24h AS (
    SELECT r.agent_id, r.target_prefix,
           COUNT(*)::int AS sample_count,
           COALESCE(AVG(CASE WHEN r.available THEN 100.0 ELSE 0.0 END), 0)::float8 AS availability_rate,
           COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.latency_ms) FILTER (WHERE r.available), 0)::float8 AS latency_p95_ms,
           COALESCE(AVG(r.packet_loss), 0)::float8 AS packet_loss_avg
    FROM scan_results r
    JOIN scan_jobs j ON j.id = r.job_id
    WHERE r.target_prefix IS NOT NULL AND r.agent_id::text = ANY($1::text[])
      AND r.scanned_at >= NOW() - INTERVAL '24 hours'
      AND j.scheme = $2 AND j.hostname = $3 AND j.path = $4 AND j.port = $5
      AND j.attempts = $6 AND j.timeout_ms = $7
    GROUP BY r.agent_id, r.target_prefix
)
SELECT e.agent_id::text, e.prefix_cidr::text, e.scheme, e.hostname, e.path, e.port, e.attempts, e.timeout_ms,
       e.tier, e.active,
       COALESCE(s7.sample_count, 0), COALESCE(s7.distinct_ip_count, 0),
       COALESCE(s7.availability_rate, 0), COALESCE(s7.latency_p95_ms, 0), COALESCE(s7.packet_loss_avg, 0),
       COALESCE(s24.sample_count, 0), COALESCE(s24.availability_rate, 0),
       COALESCE(s24.latency_p95_ms, 0), COALESCE(s24.packet_loss_avg, 0),
       e.bad_streak, s7.last_result_at, e.last_scheduled_at, e.last_evaluated_at, e.updated_at
FROM prefix_league_entries e
LEFT JOIN stats_7d s7 ON s7.agent_id = e.agent_id AND s7.target_prefix = e.prefix_cidr
LEFT JOIN stats_24h s24 ON s24.agent_id = e.agent_id AND s24.target_prefix = e.prefix_cidr
WHERE e.active AND e.agent_id::text = ANY($1::text[])
  AND e.scheme = $2 AND e.hostname = $3 AND e.path = $4 AND e.port = $5
  AND e.attempts = $6 AND e.timeout_ms = $7`,
		agentIDs, input.Scheme, input.Hostname, input.Path, input.Port, input.Attempts, input.TimeoutMS)
	if err != nil {
		return nil, fmt.Errorf("load league entries: %w", err)
	}
	defer rows.Close()
	items := make([]model.PrefixLeagueEntry, 0)
	for rows.Next() {
		var item model.PrefixLeagueEntry
		if err := rows.Scan(
			&item.AgentID, &item.PrefixCIDR, &item.Scheme, &item.Hostname, &item.Path, &item.Port, &item.Attempts, &item.TimeoutMS,
			&item.Tier, &item.Active, &item.SampleCount, &item.DistinctIPCount, &item.AvailabilityRate,
			&item.LatencyP95MS, &item.PacketLossAvg, &item.RecentSampleCount, &item.RecentAvailabilityRate,
			&item.RecentLatencyP95MS, &item.RecentPacketLossAvg, &item.BadStreak, &item.LastResultAt,
			&item.LastScheduledAt, &item.LastEvaluatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func saveLeagueEntries(ctx context.Context, tx pgx.Tx, entries []model.PrefixLeagueEntry) error {
	batch := &pgx.Batch{}
	for _, item := range entries {
		batch.Queue(`
UPDATE prefix_league_entries SET tier = $9, active = $10, sample_count = $11, distinct_ip_count = $12,
    availability_rate = $13, latency_p95_ms = $14, packet_loss_avg = $15,
    recent_sample_count = $16, recent_availability_rate = $17, recent_latency_p95_ms = $18,
    recent_packet_loss_avg = $19, bad_streak = $20, last_result_at = $21,
    last_evaluated_at = $22, updated_at = NOW()
WHERE agent_id = $1::uuid AND prefix_cidr = $2::cidr AND scheme = $3 AND hostname = $4
  AND path = $5 AND port = $6 AND attempts = $7 AND timeout_ms = $8`,
			item.AgentID, item.PrefixCIDR, item.Scheme, item.Hostname, item.Path, item.Port, item.Attempts, item.TimeoutMS,
			item.Tier, item.Active, item.SampleCount, item.DistinctIPCount, item.AvailabilityRate,
			item.LatencyP95MS, item.PacketLossAvg, item.RecentSampleCount, item.RecentAvailabilityRate,
			item.RecentLatencyP95MS, item.RecentPacketLossAvg, item.BadStreak, item.LastResultAt, item.LastEvaluatedAt)
	}
	results := tx.SendBatch(ctx, batch)
	if err := results.Close(); err != nil {
		return fmt.Errorf("save league entries: %w", err)
	}
	return nil
}

func loadLeagueCandidates(ctx context.Context, tx pgx.Tx, input model.CreateScanJobRequest, agentIDs []string, perPrefix int) ([]model.LeagueCandidate, error) {
	if perPrefix <= 0 {
		perPrefix = 2
	}
	rows, err := tx.Query(ctx, `
WITH ip_stats AS (
    SELECT r.agent_id, r.target_prefix, r.target_ip,
           COUNT(*)::int AS sample_count,
           COALESCE(AVG(CASE WHEN r.available THEN 100.0 ELSE 0.0 END), 0)::float8 AS availability_rate,
           COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.latency_ms) FILTER (WHERE r.available), 0)::float8 AS latency_p95_ms,
           COALESCE(AVG(r.packet_loss), 0)::float8 AS packet_loss_avg,
           MAX(r.scanned_at) AS last_scanned_at,
           (array_agg(r.colo ORDER BY r.scanned_at DESC))[1] AS colo
    FROM scan_results r
    JOIN scan_jobs j ON j.id = r.job_id
    WHERE r.target_prefix IS NOT NULL AND r.agent_id::text = ANY($1::text[])
      AND r.scanned_at >= NOW() - INTERVAL '7 days'
      AND j.scheme = $2 AND j.hostname = $3 AND j.path = $4 AND j.port = $5
      AND j.attempts = $6 AND j.timeout_ms = $7
    GROUP BY r.agent_id, r.target_prefix, r.target_ip
    HAVING COUNT(*) >= 1
), ranked AS (
    SELECT s.*, ROW_NUMBER() OVER (
        PARTITION BY s.agent_id, s.target_prefix
        ORDER BY s.availability_rate DESC, s.latency_p95_ms ASC, s.packet_loss_avg ASC, s.last_scanned_at DESC
    ) AS candidate_rank
    FROM ip_stats s
)
SELECT r.agent_id::text, r.target_prefix::text, host(r.target_ip), COALESCE(r.colo, ''),
       r.sample_count, r.availability_rate, r.latency_p95_ms, r.packet_loss_avg, r.last_scanned_at
FROM ranked r WHERE r.candidate_rank <= $8
ORDER BY r.agent_id, r.target_prefix, r.candidate_rank`,
		agentIDs, input.Scheme, input.Hostname, input.Path, input.Port, input.Attempts, input.TimeoutMS, perPrefix)
	if err != nil {
		return nil, fmt.Errorf("load league candidates: %w", err)
	}
	defer rows.Close()
	items := make([]model.LeagueCandidate, 0)
	for rows.Next() {
		var item model.LeagueCandidate
		if err := rows.Scan(&item.AgentID, &item.PrefixCIDR, &item.TargetIP, &item.Colo,
			&item.SampleCount, &item.AvailabilityRate, &item.LatencyP95MS, &item.PacketLossAvg,
			&item.LastScannedAt); err != nil {
			return nil, err
		}
		item.Scheme, item.Hostname, item.Path = input.Scheme, input.Hostname, input.Path
		item.Port, item.Attempts, item.TimeoutMS = input.Port, input.Attempts, input.TimeoutMS
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetLeagueDashboard(ctx context.Context, agentID string, limit int) (model.LeagueDashboard, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	filterArgs := []any{}
	condition := "e.active"
	if agentID != "" {
		filterArgs = append(filterArgs, agentID)
		condition += fmt.Sprintf(" AND e.agent_id = $%d::uuid", len(filterArgs))
	}
	var summary model.LeagueSummary
	if err := s.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT COUNT(*) FILTER (WHERE e.tier = 'observation')::int,
       COUNT(*) FILTER (WHERE e.tier = 'challenger')::int,
       COUNT(*) FILTER (WHERE e.tier = 'champion')::int
FROM prefix_league_entries e WHERE %s`, condition), filterArgs...).Scan(
		&summary.ObservationPrefixes, &summary.ChallengerPrefixes, &summary.ChampionPrefixes,
	); err != nil {
		return model.LeagueDashboard{}, fmt.Errorf("summarize league prefixes: %w", err)
	}
	args := append(append([]any{}, filterArgs...), limit)
	limitPosition := len(args)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
SELECT e.agent_id::text, a.name, a.region, a.continent, e.prefix_cidr::text,
       e.scheme, e.hostname, e.path, e.port, e.attempts, e.timeout_ms, e.tier, e.active,
       e.sample_count, e.distinct_ip_count, e.availability_rate, e.latency_p95_ms, e.packet_loss_avg,
       e.recent_sample_count, e.recent_availability_rate, e.recent_latency_p95_ms,
       e.recent_packet_loss_avg, e.bad_streak, e.last_result_at, e.last_scheduled_at,
       e.last_evaluated_at, e.updated_at
FROM prefix_league_entries e JOIN agents a ON a.id = e.agent_id
WHERE %s
ORDER BY CASE e.tier WHEN 'champion' THEN 0 WHEN 'challenger' THEN 1 ELSE 2 END,
         e.availability_rate DESC, e.latency_p95_ms ASC, e.prefix_cidr
LIMIT $%d`, condition, limitPosition), args...)
	if err != nil {
		return model.LeagueDashboard{}, fmt.Errorf("list league prefixes: %w", err)
	}
	prefixes := make([]model.PrefixLeagueEntry, 0)
	for rows.Next() {
		var item model.PrefixLeagueEntry
		if err := rows.Scan(
			&item.AgentID, &item.AgentName, &item.Region, &item.Continent, &item.PrefixCIDR,
			&item.Scheme, &item.Hostname, &item.Path, &item.Port, &item.Attempts, &item.TimeoutMS,
			&item.Tier, &item.Active, &item.SampleCount, &item.DistinctIPCount, &item.AvailabilityRate,
			&item.LatencyP95MS, &item.PacketLossAvg, &item.RecentSampleCount, &item.RecentAvailabilityRate,
			&item.RecentLatencyP95MS, &item.RecentPacketLossAvg, &item.BadStreak, &item.LastResultAt,
			&item.LastScheduledAt, &item.LastEvaluatedAt, &item.UpdatedAt,
		); err != nil {
			rows.Close()
			return model.LeagueDashboard{}, err
		}
		prefixes = append(prefixes, item)
	}
	rows.Close()

	candidateCondition := "e.active AND e.tier IN ('champion', 'challenger')"
	candidateArgs := []any{}
	if agentID != "" {
		candidateArgs = append(candidateArgs, agentID)
		candidateCondition += fmt.Sprintf(" AND e.agent_id = $%d::uuid", len(candidateArgs))
	}
	candidateArgs = append(candidateArgs, limit)
	candidateLimitPosition := len(candidateArgs)
	candidateRows, err := s.pool.Query(ctx, fmt.Sprintf(`
WITH ip_stats AS (
    SELECT e.agent_id, e.prefix_cidr, e.tier, e.scheme, e.hostname, e.path, e.port, e.attempts, e.timeout_ms, r.target_ip,
           COUNT(*)::int AS sample_count,
           COALESCE(AVG(CASE WHEN r.available THEN 100.0 ELSE 0.0 END), 0)::float8 AS availability_rate,
           COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY r.latency_ms) FILTER (WHERE r.available), 0)::float8 AS latency_p95_ms,
           COALESCE(AVG(r.packet_loss), 0)::float8 AS packet_loss_avg,
           MAX(r.scanned_at) AS last_scanned_at,
           (array_agg(r.colo ORDER BY r.scanned_at DESC))[1] AS colo
    FROM prefix_league_entries e
    JOIN scan_results r ON r.agent_id = e.agent_id AND r.target_prefix = e.prefix_cidr
    JOIN scan_jobs j ON j.id = r.job_id AND j.scheme = e.scheme AND j.hostname = e.hostname
        AND j.path = e.path AND j.port = e.port AND j.attempts = e.attempts AND j.timeout_ms = e.timeout_ms
    WHERE %s AND r.scanned_at >= NOW() - INTERVAL '7 days'
    GROUP BY e.agent_id, e.prefix_cidr, e.tier, e.scheme, e.hostname, e.path, e.port, e.attempts, e.timeout_ms, r.target_ip
    HAVING COUNT(*) >= 2
), ranked AS (
    SELECT s.*, ROW_NUMBER() OVER (
        PARTITION BY s.agent_id, s.prefix_cidr, s.scheme, s.hostname, s.path, s.port, s.attempts, s.timeout_ms
        ORDER BY s.availability_rate DESC, s.latency_p95_ms ASC, s.packet_loss_avg ASC, s.last_scanned_at DESC
    ) AS candidate_rank
    FROM ip_stats s
), selected AS (
    SELECT * FROM ranked WHERE candidate_rank <= 2
)
SELECT s.agent_id::text, a.name, a.region, a.continent, s.prefix_cidr::text, s.tier,
       s.scheme, s.hostname, s.path, s.port, s.attempts, s.timeout_ms, host(s.target_ip), COALESCE(s.colo, ''), s.sample_count, s.availability_rate,
       s.latency_p95_ms, s.packet_loss_avg, s.last_scanned_at, COUNT(*) OVER()::int
FROM selected s JOIN agents a ON a.id = s.agent_id
ORDER BY CASE s.tier WHEN 'champion' THEN 0 ELSE 1 END,
         s.availability_rate DESC, s.latency_p95_ms ASC
LIMIT $%d`, candidateCondition, candidateLimitPosition), candidateArgs...)
	if err != nil {
		return model.LeagueDashboard{}, fmt.Errorf("list league candidates: %w", err)
	}
	candidates := make([]model.LeagueCandidate, 0)
	candidateTotal := 0
	for candidateRows.Next() {
		var item model.LeagueCandidate
		if err := candidateRows.Scan(&item.AgentID, &item.AgentName, &item.Region, &item.Continent,
			&item.PrefixCIDR, &item.Tier, &item.Scheme, &item.Hostname, &item.Path, &item.Port, &item.Attempts, &item.TimeoutMS,
			&item.TargetIP, &item.Colo, &item.SampleCount,
			&item.AvailabilityRate, &item.LatencyP95MS, &item.PacketLossAvg, &item.LastScannedAt,
			&candidateTotal); err != nil {
			candidateRows.Close()
			return model.LeagueDashboard{}, err
		}
		candidates = append(candidates, item)
	}
	candidateRows.Close()

	summary.CandidateIPs = candidateTotal
	return model.LeagueDashboard{Summary: summary, Prefixes: prefixes, Candidates: candidates}, nil
}

func (s *Store) GetIPTrend(ctx context.Context, filter model.IPTrendFilter) (model.IPTrend, error) {
	var trend model.IPTrend
	trend.AgentID = filter.AgentID
	trend.TargetIP = filter.TargetIP
	if err := s.pool.QueryRow(ctx, `SELECT name FROM agents WHERE id = $1::uuid`, filter.AgentID).Scan(&trend.AgentName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.IPTrend{}, storepkg.ErrNotFound
		}
		return model.IPTrend{}, fmt.Errorf("load trend agent: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
SELECT r.scanned_at, r.available, r.latency_ms, r.packet_loss, r.tcp_connect_ms,
       r.tls_handshake_ms, r.ttfb_ms, r.colo
FROM scan_results r
JOIN scan_jobs j ON j.id = r.job_id
WHERE r.agent_id = $1::uuid AND r.target_ip = $2::inet AND r.scanned_at >= $3
  AND j.scheme = $4 AND j.hostname = $5 AND j.path = $6 AND j.port = $7
  AND j.attempts = $8 AND j.timeout_ms = $9
ORDER BY r.scanned_at ASC
LIMIT 10000`, filter.AgentID, filter.TargetIP, filter.Since, filter.Scheme, filter.Hostname,
		filter.Path, filter.Port, filter.Attempts, filter.TimeoutMS)
	if err != nil {
		return model.IPTrend{}, fmt.Errorf("load IP trend: %w", err)
	}
	defer rows.Close()
	trend.Points = make([]model.IPTrendPoint, 0)
	latencies := make([]float64, 0)
	availableCount := 0
	packetLossTotal := 0.0
	for rows.Next() {
		var point model.IPTrendPoint
		if err := rows.Scan(&point.ScannedAt, &point.Available, &point.LatencyMS, &point.PacketLoss,
			&point.TCPConnectMS, &point.TLSHandshakeMS, &point.TTFBMS, &point.Colo); err != nil {
			return model.IPTrend{}, err
		}
		trend.Points = append(trend.Points, point)
		packetLossTotal += point.PacketLoss
		if point.Available {
			availableCount++
			latencies = append(latencies, point.LatencyMS)
		}
		if point.Colo != "" {
			trend.Summary.LatestColo = point.Colo
		}
	}
	if err := rows.Err(); err != nil {
		return model.IPTrend{}, err
	}
	trend.Summary.SampleCount = len(trend.Points)
	if len(trend.Points) > 0 {
		trend.Summary.AvailabilityRate = float64(availableCount) / float64(len(trend.Points)) * 100
		trend.Summary.PacketLossAvg = packetLossTotal / float64(len(trend.Points))
	}
	if len(latencies) > 0 {
		sort.Float64s(latencies)
		trend.Summary.LatencyP50MS = percentile(latencies, 0.5)
		trend.Summary.LatencyP95MS = percentile(latencies, 0.95)
	}
	return trend, nil
}

func percentile(sorted []float64, quantile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := float64(len(sorted)-1) * quantile
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	weight := position - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
