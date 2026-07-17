CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    region TEXT NOT NULL,
    continent TEXT NOT NULL,
    concurrency INTEGER NOT NULL CHECK (concurrency > 0),
    status TEXT NOT NULL DEFAULT 'online',
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cloudflare_prefixes (
    id BIGSERIAL PRIMARY KEY,
    cidr CIDR NOT NULL UNIQUE,
    ip_version SMALLINT NOT NULL CHECK (ip_version IN (4, 6)),
    source TEXT NOT NULL DEFAULT 'cloudflare_official',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cloudflare_prefixes_active ON cloudflare_prefixes (active, ip_version);

CREATE TABLE IF NOT EXISTS source_sync_state (
    source TEXT PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'never',
    prefix_count INTEGER NOT NULL DEFAULT 0,
    ipv4_count INTEGER NOT NULL DEFAULT 0,
    ipv6_count INTEGER NOT NULL DEFAULT 0,
    last_synced_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);



CREATE TABLE IF NOT EXISTS asn_sources (
    asn BIGINT PRIMARY KEY CHECK (asn > 0 AND asn <= 4294967295),
    name TEXT NOT NULL,
    organization TEXT NOT NULL DEFAULT 'Cloudflare, Inc.',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    managed BOOLEAN NOT NULL DEFAULT FALSE,
    sync_status TEXT NOT NULL DEFAULT 'never',
    prefix_count INTEGER NOT NULL DEFAULT 0,
    ipv4_count INTEGER NOT NULL DEFAULT 0,
    ipv6_count INTEGER NOT NULL DEFAULT 0,
    last_synced_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS asn_prefixes (
    asn BIGINT NOT NULL REFERENCES asn_sources(asn) ON DELETE CASCADE,
    cidr CIDR NOT NULL,
    ip_version SMALLINT NOT NULL CHECK (ip_version IN (4, 6)),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (asn, cidr)
);
CREATE INDEX IF NOT EXISTS idx_asn_prefixes_active ON asn_prefixes (active, ip_version, asn);

INSERT INTO asn_sources (asn, name, organization, enabled, managed)
VALUES
    (13335, 'CLOUDFLARENET', 'Cloudflare, Inc.', TRUE, TRUE),
    (209242, 'CLOUDFLARESPECTRUM', 'Cloudflare, Inc.', TRUE, TRUE),
    (14789, 'CLOUDFLARENET', 'Cloudflare, Inc.', TRUE, TRUE),
    (394536, 'CLOUDFLARENET-SFO', 'Cloudflare, Inc.', TRUE, TRUE),
    (395747, 'CLOUDFLARENET-SFO05', 'Cloudflare, Inc.', TRUE, TRUE),
    (400095, 'CLOUDFLARENET', 'Cloudflare, Inc.', TRUE, TRUE)
ON CONFLICT (asn) DO UPDATE SET managed = TRUE;

CREATE TABLE IF NOT EXISTS ip_targets (
    id BIGSERIAL PRIMARY KEY,
    ip INET NOT NULL UNIQUE,
    source TEXT NOT NULL DEFAULT 'cloudflare_official',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS scan_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'normal',
    status TEXT NOT NULL DEFAULT 'pending',
    sampling_mode TEXT NOT NULL DEFAULT 'count',
    scheme TEXT NOT NULL DEFAULT 'https',
    hostname TEXT NOT NULL DEFAULT 'cloudflare.com',
    path TEXT NOT NULL DEFAULT '/cdn-cgi/trace',
    port INTEGER NOT NULL DEFAULT 443,
    attempts INTEGER NOT NULL DEFAULT 3,
    timeout_ms INTEGER NOT NULL DEFAULT 5000,
    max_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 1000,
    max_packet_loss DOUBLE PRECISION NOT NULL DEFAULT 50,
    blacklist_minutes INTEGER NOT NULL DEFAULT 60,
    total_targets INTEGER NOT NULL DEFAULT 0,
    completed_targets INTEGER NOT NULL DEFAULT 0,
    success_targets INTEGER NOT NULL DEFAULT 0,
    failed_targets INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'normal';
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS sampling_mode TEXT NOT NULL DEFAULT 'count';
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS scheme TEXT NOT NULL DEFAULT 'https';
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS hostname TEXT NOT NULL DEFAULT 'cloudflare.com';
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS path TEXT NOT NULL DEFAULT '/cdn-cgi/trace';
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS port INTEGER NOT NULL DEFAULT 443;
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 3;
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS timeout_ms INTEGER NOT NULL DEFAULT 5000;
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS max_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 1000;
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS max_packet_loss DOUBLE PRECISION NOT NULL DEFAULT 50;
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS blacklist_minutes INTEGER NOT NULL DEFAULT 60;
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS success_targets INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scan_jobs ADD COLUMN IF NOT EXISTS failed_targets INTEGER NOT NULL DEFAULT 0;


CREATE TABLE IF NOT EXISTS scan_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cron_expression TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    agent_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    sampling_mode TEXT NOT NULL DEFAULT 'count',
    target_count INTEGER NOT NULL DEFAULT 128,
    scheme TEXT NOT NULL DEFAULT 'https',
    hostname TEXT NOT NULL DEFAULT 'cloudflare.com',
    path TEXT NOT NULL DEFAULT '/cdn-cgi/trace',
    port INTEGER NOT NULL DEFAULT 443,
    attempts INTEGER NOT NULL DEFAULT 3,
    timeout_ms INTEGER NOT NULL DEFAULT 5000,
    max_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 1000,
    max_packet_loss DOUBLE PRECISION NOT NULL DEFAULT 50,
    blacklist_minutes INTEGER NOT NULL DEFAULT 60,
    include_ipv6 BOOLEAN NOT NULL DEFAULT FALSE,
    include_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ,
    last_job_id UUID REFERENCES scan_jobs(id) ON DELETE SET NULL,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE scan_schedules ADD COLUMN IF NOT EXISTS sampling_mode TEXT NOT NULL DEFAULT 'count';
CREATE INDEX IF NOT EXISTS idx_scan_schedules_due ON scan_schedules (enabled, next_run_at);

CREATE TABLE IF NOT EXISTS scan_tasks (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    preferred_agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    target_ip INET NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    lease_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE (job_id, preferred_agent_id, target_ip)
);
CREATE INDEX IF NOT EXISTS idx_scan_tasks_claim ON scan_tasks (preferred_agent_id, status, id);
CREATE INDEX IF NOT EXISTS idx_scan_tasks_job ON scan_tasks (job_id, status);

CREATE TABLE IF NOT EXISTS scan_results (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID REFERENCES scan_jobs(id) ON DELETE CASCADE,
    task_id BIGINT REFERENCES scan_tasks(id) ON DELETE SET NULL,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    target_ip INET NOT NULL,
    available BOOLEAN NOT NULL,
    latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    packet_loss DOUBLE PRECISION NOT NULL DEFAULT 100,
    tcp_connect_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    tls_handshake_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    ttfb_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    http_status INTEGER NOT NULL DEFAULT 0,
    http_version TEXT NOT NULL DEFAULT '',
    tls_version TEXT NOT NULL DEFAULT '',
    colo TEXT NOT NULL DEFAULT '',
    cf_ray TEXT NOT NULL DEFAULT '',
    error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    successful_tries INTEGER NOT NULL DEFAULT 0,
    attempts INTEGER NOT NULL DEFAULT 1,
    scanned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS task_id BIGINT REFERENCES scan_tasks(id) ON DELETE SET NULL;
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS tcp_connect_ms DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS tls_handshake_ms DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS ttfb_ms DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS total_ms DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS http_status INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS http_version TEXT NOT NULL DEFAULT '';
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS tls_version TEXT NOT NULL DEFAULT '';
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS cf_ray TEXT NOT NULL DEFAULT '';
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS error_message TEXT NOT NULL DEFAULT '';
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS successful_tries INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scan_results ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 1;
ALTER TABLE scan_results ALTER COLUMN latency_ms SET DEFAULT 0;
ALTER TABLE scan_results ALTER COLUMN packet_loss SET DEFAULT 100;
ALTER TABLE scan_results ALTER COLUMN colo SET DEFAULT '';
ALTER TABLE scan_results ALTER COLUMN error_code SET DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_scan_results_agent_time ON scan_results (agent_id, scanned_at DESC);
CREATE INDEX IF NOT EXISTS idx_scan_results_target_time ON scan_results (target_ip, scanned_at DESC);
CREATE INDEX IF NOT EXISTS idx_scan_results_latency ON scan_results (available, latency_ms, scanned_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_scan_results_task_unique ON scan_results (task_id);

CREATE TABLE IF NOT EXISTS blacklist_entries (
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    target_ip INET NOT NULL,
    reason TEXT NOT NULL,
    failure_count INTEGER NOT NULL DEFAULT 1,
    blocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    retry_after TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (agent_id, target_ip)
);
CREATE INDEX IF NOT EXISTS idx_blacklist_retry ON blacklist_entries (retry_after, agent_id);


CREATE TABLE IF NOT EXISTS blacklist_recheck_settings (
    key TEXT PRIMARY KEY DEFAULT 'default' CHECK (key = 'default'),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cron_expression TEXT NOT NULL DEFAULT '*/15 * * * *',
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    due_only BOOLEAN NOT NULL DEFAULT TRUE,
    fraction DOUBLE PRECISION NOT NULL DEFAULT 0.5 CHECK (fraction > 0 AND fraction <= 1),
    max_targets INTEGER NOT NULL DEFAULT 500 CHECK (max_targets > 0 AND max_targets <= 5000),
    skip_if_running BOOLEAN NOT NULL DEFAULT TRUE,
    attempts INTEGER NOT NULL DEFAULT 3,
    timeout_ms INTEGER NOT NULL DEFAULT 5000,
    max_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 1000,
    max_packet_loss DOUBLE PRECISION NOT NULL DEFAULT 50,
    retry_minutes INTEGER NOT NULL DEFAULT 120,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '15 minutes'),
    last_run_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO blacklist_recheck_settings (key) VALUES ('default') ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS source_sync_schedules (
    source TEXT PRIMARY KEY CHECK (source IN ('official', 'asn')),
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cron_expression TEXT NOT NULL DEFAULT '0 */6 * * *',
    timezone TEXT NOT NULL DEFAULT 'Asia/Shanghai',
    run_on_startup BOOLEAN NOT NULL DEFAULT TRUE,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_run_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO source_sync_schedules (source, name)
VALUES ('official', 'Cloudflare 官方地址段同步'), ('asn', 'Cloudflare ASN 前缀同步')
ON CONFLICT (source) DO NOTHING;

CREATE TABLE IF NOT EXISTS automation_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    automation_type TEXT NOT NULL,
    automation_key TEXT NOT NULL,
    name TEXT NOT NULL,
    trigger TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    config_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_automation_runs_started ON automation_runs (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_automation_runs_type ON automation_runs (automation_type, automation_key, started_at DESC);

UPDATE automation_runs SET status = 'failed', error = CASE WHEN error = '' THEN '中心服务在执行期间中断' ELSE error END,
    finished_at = COALESCE(finished_at, NOW())
WHERE status = 'running' AND started_at < NOW() - INTERVAL '15 minutes';


-- Agent enrollment and per-Agent credentials. Runtime migrations use internal/store/postgres/schema.sql.
ALTER TABLE agents ADD COLUMN IF NOT EXISTS os TEXT NOT NULL DEFAULT '';
ALTER TABLE agents ADD COLUMN IF NOT EXISTS architecture TEXT NOT NULL DEFAULT '';
ALTER TABLE agents ADD COLUMN IF NOT EXISTS version TEXT NOT NULL DEFAULT '';
ALTER TABLE agents ADD COLUMN IF NOT EXISTS auth_mode TEXT NOT NULL DEFAULT 'legacy';

CREATE TABLE IF NOT EXISTS agent_enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL UNIQUE,
    mode TEXT NOT NULL CHECK (mode IN ('device', 'preauthorized')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'claimed', 'rejected', 'revoked')),
    requested_name TEXT NOT NULL DEFAULT '',
    os TEXT NOT NULL DEFAULT '',
    architecture TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    requested_concurrency INTEGER NOT NULL DEFAULT 1 CHECK (requested_concurrency > 0),
    name TEXT NOT NULL DEFAULT '',
    region TEXT NOT NULL DEFAULT '',
    continent TEXT NOT NULL DEFAULT '',
    concurrency INTEGER NOT NULL DEFAULT 1 CHECK (concurrency > 0),
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    approved_at TIMESTAMPTZ,
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_agent_enrollments_active ON agent_enrollments (status, expires_at DESC);

CREATE TABLE IF NOT EXISTS agent_credentials (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    secret_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_agent_credentials_agent ON agent_credentials (agent_id, revoked_at);
