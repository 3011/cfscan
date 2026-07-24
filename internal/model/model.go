package model

import (
	"encoding/json"
	"time"
)

type Agent struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Region       string    `json:"region"`
	Continent    string    `json:"continent"`
	Concurrency  int       `json:"concurrency"`
	Status       string    `json:"status"`
	OS           string    `json:"os"`
	Architecture string    `json:"architecture"`
	Version      string    `json:"version"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	CreatedAt    time.Time `json:"created_at"`
}

const (
	AgentEnrollmentModeDevice        = "device"
	AgentEnrollmentModePreauthorized = "preauthorized"

	AgentEnrollmentPending  = "pending"
	AgentEnrollmentApproved = "approved"
	AgentEnrollmentClaimed  = "claimed"
	AgentEnrollmentRejected = "rejected"
	AgentEnrollmentRevoked  = "revoked"
	AgentEnrollmentExpired  = "expired"
)

type AgentEnrollment struct {
	ID                   string     `json:"id"`
	Mode                 string     `json:"mode"`
	Status               string     `json:"status"`
	RequestedName        string     `json:"requested_name"`
	OS                   string     `json:"os"`
	Architecture         string     `json:"architecture"`
	Version              string     `json:"version"`
	RequestedConcurrency int        `json:"requested_concurrency"`
	Name                 string     `json:"name,omitempty"`
	Region               string     `json:"region,omitempty"`
	Continent            string     `json:"continent,omitempty"`
	Concurrency          int        `json:"concurrency,omitempty"`
	AgentID              string     `json:"agent_id,omitempty"`
	ExpiresAt            time.Time  `json:"expires_at"`
	ApprovedAt           *time.Time `json:"approved_at,omitempty"`
	ClaimedAt            *time.Time `json:"claimed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type CreateAgentEnrollment struct {
	TokenHash            string
	Mode                 string
	Status               string
	RequestedName        string
	OS                   string
	Architecture         string
	Version              string
	RequestedConcurrency int
	Name                 string
	Region               string
	Continent            string
	Concurrency          int
	ExpiresAt            time.Time
}

type CreateDeviceEnrollmentRequest struct {
	Name         string `json:"name"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Version      string `json:"version"`
	Concurrency  int    `json:"concurrency"`
}

type CreateDeviceEnrollmentResponse struct {
	PairingToken            string `json:"pairing_token"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type ApproveAgentEnrollmentRequest struct {
	Name        string `json:"name"`
	Region      string `json:"region"`
	Continent   string `json:"continent"`
	Concurrency int    `json:"concurrency"`
}

type CreatePreauthorizedEnrollmentRequest struct {
	Name        string `json:"name"`
	Region      string `json:"region"`
	Continent   string `json:"continent"`
	Concurrency int    `json:"concurrency"`
	TTLMinutes  int    `json:"ttl_minutes"`
}

type CreatePreauthorizedEnrollmentResponse struct {
	Enrollment   AgentEnrollment `json:"enrollment"`
	PairingToken string          `json:"pairing_token"`
	ExpiresIn    int             `json:"expires_in"`
}

type ClaimAgentEnrollmentRequest struct {
	PairingToken     string `json:"pairing_token"`
	CredentialID     string `json:"credential_id"`
	CredentialSecret string `json:"credential_secret"`
	OS               string `json:"os,omitempty"`
	Architecture     string `json:"architecture,omitempty"`
	Version          string `json:"version,omitempty"`
}

type ClaimAgentEnrollmentResponse struct {
	Status      string `json:"status"`
	AgentID     string `json:"agent_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
	Interval    int    `json:"interval,omitempty"`
}

type AgentEnrollmentConfig struct {
	PublicURL    string `json:"public_url"`
	AgentImage   string `json:"agent_image"`
	AgentVersion string `json:"agent_version"`
	TTLSeconds   int    `json:"ttl_seconds"`
	PollInterval int    `json:"poll_interval"`
}

type Overview struct {
	AgentsTotal       int64   `json:"agents_total"`
	AgentsOnline      int64   `json:"agents_online"`
	PrefixesTotal     int64   `json:"prefixes_total"`
	IPsTotal          int64   `json:"ips_total"`
	IPsAvailable      int64   `json:"ips_available"`
	IPsBlacklisted    int64   `json:"ips_blacklisted"`
	RunningJobs       int64   `json:"running_jobs"`
	CompletedJobs     int64   `json:"completed_jobs"`
	AverageLatencyMS  float64 `json:"average_latency_ms"`
	ResultsLast24Hour int64   `json:"results_last_24h"`
}

type Prefix struct {
	CIDR      string    `json:"cidr"`
	IPVersion int       `json:"ip_version"`
	Source    string    `json:"source"`
	Active    bool      `json:"active"`
	LastSeen  time.Time `json:"last_seen_at"`
}

type SourceStatus struct {
	Source       string     `json:"source"`
	Status       string     `json:"status"`
	PrefixCount  int        `json:"prefix_count"`
	IPv4Count    int        `json:"ipv4_count"`
	IPv6Count    int        `json:"ipv6_count"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
}

// ASNSource describes one BGP origin included in the Cloudflare address pool.
type ASNSource struct {
	ASN          int64      `json:"asn"`
	Name         string     `json:"name"`
	Organization string     `json:"organization"`
	Enabled      bool       `json:"enabled"`
	Managed      bool       `json:"managed"`
	Status       string     `json:"status"`
	PrefixCount  int        `json:"prefix_count"`
	IPv4Count    int        `json:"ipv4_count"`
	IPv6Count    int        `json:"ipv6_count"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type CreateASNSourceRequest struct {
	ASN          int64  `json:"asn"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Enabled      *bool  `json:"enabled,omitempty"`
}

type UpdateASNSourceRequest struct {
	Name         *string `json:"name,omitempty"`
	Organization *string `json:"organization,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
}

type ASNSyncSummary struct {
	Items  []ASNSource `json:"items"`
	Synced int         `json:"synced"`
	Failed int         `json:"failed"`
}

type CreateScanJobRequest struct {
	Kind             string   `json:"-"`
	ForceLeague      bool     `json:"-"`
	Name             string   `json:"name"`
	AgentIDs         []string `json:"agent_ids"`
	SamplingMode     string   `json:"sampling_mode"`
	TargetCount      int      `json:"target_count"`
	Scheme           string   `json:"scheme"`
	Hostname         string   `json:"hostname"`
	Path             string   `json:"path"`
	Port             int      `json:"port"`
	Attempts         int      `json:"attempts"`
	TimeoutMS        int      `json:"timeout_ms"`
	MaxLatencyMS     float64  `json:"max_latency_ms"`
	MaxPacketLoss    float64  `json:"max_packet_loss"`
	BlacklistMinutes int      `json:"blacklist_minutes"`
	IncludeIPv6      bool     `json:"include_ipv6"`
	IncludeBlocked   bool     `json:"include_blocked"`
}

type ScanTarget struct {
	TargetIP   string `json:"target_ip"`
	PrefixCIDR string `json:"prefix_cidr,omitempty"`
}

type AgentScanTargets struct {
	AgentID string       `json:"agent_id"`
	Targets []ScanTarget `json:"targets"`
}

type ScanJob struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Kind             string     `json:"kind"`
	Status           string     `json:"status"`
	SamplingMode     string     `json:"sampling_mode"`
	Scheme           string     `json:"scheme"`
	Hostname         string     `json:"hostname"`
	Path             string     `json:"path"`
	Port             int        `json:"port"`
	Attempts         int        `json:"attempts"`
	TimeoutMS        int        `json:"timeout_ms"`
	MaxLatencyMS     float64    `json:"max_latency_ms"`
	MaxPacketLoss    float64    `json:"max_packet_loss"`
	BlacklistMinutes int        `json:"blacklist_minutes"`
	TotalTargets     int        `json:"total_targets"`
	CompletedTargets int        `json:"completed_targets"`
	SuccessTargets   int        `json:"success_targets"`
	FailedTargets    int        `json:"failed_targets"`
	Progress         float64    `json:"progress"`
	CreatedAt        time.Time  `json:"created_at"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
}

type TaskClaimRequest struct {
	Limit int `json:"limit"`
}

type ScanTask struct {
	ID       int64  `json:"id"`
	TargetIP string `json:"target_ip"`
}

type TaskBatch struct {
	JobID     string     `json:"job_id"`
	JobName   string     `json:"job_name"`
	Scheme    string     `json:"scheme"`
	Hostname  string     `json:"hostname"`
	Path      string     `json:"path"`
	Port      int        `json:"port"`
	Attempts  int        `json:"attempts"`
	TimeoutMS int        `json:"timeout_ms"`
	Tasks     []ScanTask `json:"tasks"`
}

type ProbeResult struct {
	TaskID          int64   `json:"task_id"`
	TargetIP        string  `json:"target_ip"`
	Available       bool    `json:"available"`
	LatencyMS       float64 `json:"latency_ms"`
	PacketLoss      float64 `json:"packet_loss"`
	TCPConnectMS    float64 `json:"tcp_connect_ms"`
	TLSHandshakeMS  float64 `json:"tls_handshake_ms"`
	TTFBMS          float64 `json:"ttfb_ms"`
	TotalMS         float64 `json:"total_ms"`
	HTTPStatus      int     `json:"http_status"`
	HTTPVersion     string  `json:"http_version"`
	TLSVersion      string  `json:"tls_version"`
	Colo            string  `json:"colo"`
	CFRay           string  `json:"cf_ray"`
	ErrorCode       string  `json:"error_code,omitempty"`
	ErrorMessage    string  `json:"error_message,omitempty"`
	SuccessfulTries int     `json:"successful_tries"`
	Attempts        int     `json:"attempts"`
}

type ResultBatch struct {
	AgentID string        `json:"-"`
	JobID   string        `json:"job_id"`
	Results []ProbeResult `json:"results"`
}

// ColoLocation maps a Cloudflare colo code to the geographic metadata published on Cloudflare Status.
type ColoLocation struct {
	Code      string    `json:"code"`
	City      string    `json:"city"`
	Country   string    `json:"country"`
	Continent string    `json:"continent"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ColoSyncSummary struct {
	Locations int       `json:"locations"`
	SyncedAt  time.Time `json:"synced_at"`
}

// ResultColoFacet is a colo location that currently has matching scan results.
type ResultColoFacet struct {
	Code      string `json:"code"`
	City      string `json:"city"`
	Country   string `json:"country"`
	Continent string `json:"continent"`
	Count     int    `json:"count"`
}

type ScanResult struct {
	ID             int64     `json:"id"`
	JobID          string    `json:"job_id"`
	JobName        string    `json:"job_name"`
	AgentID        string    `json:"agent_id"`
	AgentName      string    `json:"agent_name"`
	Region         string    `json:"region"`
	Continent      string    `json:"continent"`
	TargetIP       string    `json:"target_ip"`
	Available      bool      `json:"available"`
	LatencyMS      float64   `json:"latency_ms"`
	PacketLoss     float64   `json:"packet_loss"`
	TCPConnectMS   float64   `json:"tcp_connect_ms"`
	TLSHandshakeMS float64   `json:"tls_handshake_ms"`
	TTFBMS         float64   `json:"ttfb_ms"`
	TotalMS        float64   `json:"total_ms"`
	HTTPStatus     int       `json:"http_status"`
	HTTPVersion    string    `json:"http_version"`
	TLSVersion     string    `json:"tls_version"`
	Colo           string    `json:"colo"`
	ColoCity       string    `json:"colo_city"`
	ColoCountry    string    `json:"colo_country"`
	ColoContinent  string    `json:"colo_continent"`
	CFRay          string    `json:"cf_ray"`
	ErrorCode      string    `json:"error_code,omitempty"`
	ScannedAt      time.Time `json:"scanned_at"`
}

type ResultFilter struct {
	View          string
	AgentID       string
	JobID         string
	Region        string
	Continent     string
	TargetIP      string
	Colo          string
	ColoCity      string
	ColoCountry   string
	ColoContinent string
	Available     *bool
	Since         *time.Time
	Page          int
	PageSize      int
	Sort          string
	Order         string
}

type ResultStatusCounts struct {
	All       int64 `json:"all"`
	Available int64 `json:"available"`
	Failed    int64 `json:"failed"`
}

type ResultPage struct {
	Items      []ScanResult       `json:"items"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
	Counts     ResultStatusCounts `json:"counts"`
}

type ResultJobFacet struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
}

type PrefixLeagueEntry struct {
	AgentID                string     `json:"agent_id"`
	AgentName              string     `json:"agent_name"`
	Region                 string     `json:"region"`
	Continent              string     `json:"continent"`
	PrefixCIDR             string     `json:"prefix_cidr"`
	Scheme                 string     `json:"scheme"`
	Hostname               string     `json:"hostname"`
	Path                   string     `json:"path"`
	Port                   int        `json:"port"`
	Attempts               int        `json:"attempts"`
	TimeoutMS              int        `json:"timeout_ms"`
	Tier                   string     `json:"tier"`
	Active                 bool       `json:"active"`
	SampleCount            int        `json:"sample_count"`
	DistinctIPCount        int        `json:"distinct_ip_count"`
	AvailabilityRate       float64    `json:"availability_rate"`
	LatencyP95MS           float64    `json:"latency_p95_ms"`
	PacketLossAvg          float64    `json:"packet_loss_avg"`
	RecentSampleCount      int        `json:"recent_sample_count"`
	RecentAvailabilityRate float64    `json:"recent_availability_rate"`
	RecentLatencyP95MS     float64    `json:"recent_latency_p95_ms"`
	RecentPacketLossAvg    float64    `json:"recent_packet_loss_avg"`
	BadStreak              int        `json:"bad_streak"`
	LastResultAt           *time.Time `json:"last_result_at,omitempty"`
	LastScheduledAt        *time.Time `json:"last_scheduled_at,omitempty"`
	LastEvaluatedAt        *time.Time `json:"last_evaluated_at,omitempty"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type LeagueCandidate struct {
	AgentID          string    `json:"agent_id"`
	AgentName        string    `json:"agent_name"`
	Region           string    `json:"region"`
	Continent        string    `json:"continent"`
	PrefixCIDR       string    `json:"prefix_cidr"`
	Tier             string    `json:"tier"`
	Scheme           string    `json:"scheme"`
	Hostname         string    `json:"hostname"`
	Path             string    `json:"path"`
	Port             int       `json:"port"`
	Attempts         int       `json:"attempts"`
	TimeoutMS        int       `json:"timeout_ms"`
	TargetIP         string    `json:"target_ip"`
	Colo             string    `json:"colo"`
	SampleCount      int       `json:"sample_count"`
	AvailabilityRate float64   `json:"availability_rate"`
	LatencyP95MS     float64   `json:"latency_p95_ms"`
	PacketLossAvg    float64   `json:"packet_loss_avg"`
	LastScannedAt    time.Time `json:"last_scanned_at"`
}

type IPTrendFilter struct {
	AgentID   string
	TargetIP  string
	Scheme    string
	Hostname  string
	Path      string
	Port      int
	Attempts  int
	TimeoutMS int
	Since     time.Time
}

type IPTrendSummary struct {
	SampleCount      int     `json:"sample_count"`
	AvailabilityRate float64 `json:"availability_rate"`
	LatencyP50MS     float64 `json:"latency_p50_ms"`
	LatencyP95MS     float64 `json:"latency_p95_ms"`
	PacketLossAvg    float64 `json:"packet_loss_avg"`
	LatestColo       string  `json:"latest_colo"`
}

type IPTrendPoint struct {
	ScannedAt      time.Time `json:"scanned_at"`
	Available      bool      `json:"available"`
	LatencyMS      float64   `json:"latency_ms"`
	PacketLoss     float64   `json:"packet_loss"`
	TCPConnectMS   float64   `json:"tcp_connect_ms"`
	TLSHandshakeMS float64   `json:"tls_handshake_ms"`
	TTFBMS         float64   `json:"ttfb_ms"`
	Colo           string    `json:"colo"`
}

type IPTrend struct {
	AgentID   string         `json:"agent_id"`
	AgentName string         `json:"agent_name"`
	TargetIP  string         `json:"target_ip"`
	Summary   IPTrendSummary `json:"summary"`
	Points    []IPTrendPoint `json:"points"`
}

type LeagueSummary struct {
	ObservationPrefixes int `json:"observation_prefixes"`
	ChallengerPrefixes  int `json:"challenger_prefixes"`
	ChampionPrefixes    int `json:"champion_prefixes"`
	CandidateIPs        int `json:"candidate_ips"`
}

type LeagueDashboardFilter struct {
	AgentID           string
	PrefixPage        int
	PrefixPageSize    int
	CandidatePage     int
	CandidatePageSize int
}

type PrefixLeaguePage struct {
	Items      []PrefixLeagueEntry `json:"items"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

type LeagueCandidatePage struct {
	Items      []LeagueCandidate `json:"items"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

type LeagueDashboard struct {
	Summary    LeagueSummary       `json:"summary"`
	Prefixes   PrefixLeaguePage    `json:"prefixes"`
	Candidates LeagueCandidatePage `json:"candidates"`
}

type BlacklistEntry struct {
	AgentID      string    `json:"agent_id"`
	AgentName    string    `json:"agent_name"`
	Region       string    `json:"region"`
	Continent    string    `json:"continent"`
	TargetIP     string    `json:"target_ip"`
	Reason       string    `json:"reason"`
	FailureCount int       `json:"failure_count"`
	BlockedAt    time.Time `json:"blocked_at"`
	RetryAfter   time.Time `json:"retry_after"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UpsertScanScheduleRequest struct {
	Name             string   `json:"name"`
	Enabled          bool     `json:"enabled"`
	CronExpression   string   `json:"cron_expression"`
	Timezone         string   `json:"timezone"`
	AgentIDs         []string `json:"agent_ids"`
	SamplingMode     string   `json:"sampling_mode"`
	TargetCount      int      `json:"target_count"`
	Scheme           string   `json:"scheme"`
	Hostname         string   `json:"hostname"`
	Path             string   `json:"path"`
	Port             int      `json:"port"`
	Attempts         int      `json:"attempts"`
	TimeoutMS        int      `json:"timeout_ms"`
	MaxLatencyMS     float64  `json:"max_latency_ms"`
	MaxPacketLoss    float64  `json:"max_packet_loss"`
	BlacklistMinutes int      `json:"blacklist_minutes"`
	IncludeIPv6      bool     `json:"include_ipv6"`
	IncludeBlocked   bool     `json:"include_blocked"`
}

func (r UpsertScanScheduleRequest) JobRequest() CreateScanJobRequest {
	return CreateScanJobRequest{
		Name: r.Name, AgentIDs: r.AgentIDs, SamplingMode: r.SamplingMode, TargetCount: r.TargetCount, Scheme: r.Scheme,
		Hostname: r.Hostname, Path: r.Path, Port: r.Port, Attempts: r.Attempts,
		TimeoutMS: r.TimeoutMS, MaxLatencyMS: r.MaxLatencyMS, MaxPacketLoss: r.MaxPacketLoss,
		BlacklistMinutes: r.BlacklistMinutes, IncludeIPv6: r.IncludeIPv6, IncludeBlocked: r.IncludeBlocked,
	}
}

func (r *UpsertScanScheduleRequest) ApplyJobRequest(job CreateScanJobRequest) {
	r.Name = job.Name
	r.AgentIDs = job.AgentIDs
	r.SamplingMode = job.SamplingMode
	r.TargetCount = job.TargetCount
	r.Scheme = job.Scheme
	r.Hostname = job.Hostname
	r.Path = job.Path
	r.Port = job.Port
	r.Attempts = job.Attempts
	r.TimeoutMS = job.TimeoutMS
	r.MaxLatencyMS = job.MaxLatencyMS
	r.MaxPacketLoss = job.MaxPacketLoss
	r.BlacklistMinutes = job.BlacklistMinutes
	r.IncludeIPv6 = job.IncludeIPv6
	r.IncludeBlocked = job.IncludeBlocked
}

type ScanSchedule struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Enabled          bool       `json:"enabled"`
	CronExpression   string     `json:"cron_expression"`
	Timezone         string     `json:"timezone"`
	AgentIDs         []string   `json:"agent_ids"`
	SamplingMode     string     `json:"sampling_mode"`
	TargetCount      int        `json:"target_count"`
	Scheme           string     `json:"scheme"`
	Hostname         string     `json:"hostname"`
	Path             string     `json:"path"`
	Port             int        `json:"port"`
	Attempts         int        `json:"attempts"`
	TimeoutMS        int        `json:"timeout_ms"`
	MaxLatencyMS     float64    `json:"max_latency_ms"`
	MaxPacketLoss    float64    `json:"max_packet_loss"`
	BlacklistMinutes int        `json:"blacklist_minutes"`
	IncludeIPv6      bool       `json:"include_ipv6"`
	IncludeBlocked   bool       `json:"include_blocked"`
	NextRunAt        time.Time  `json:"next_run_at"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	LastJobID        *string    `json:"last_job_id,omitempty"`
	LastError        string     `json:"last_error,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (s ScanSchedule) JobRequest() CreateScanJobRequest {
	return CreateScanJobRequest{
		Name: s.Name, AgentIDs: s.AgentIDs, SamplingMode: s.SamplingMode, TargetCount: s.TargetCount, Scheme: s.Scheme,
		Hostname: s.Hostname, Path: s.Path, Port: s.Port, Attempts: s.Attempts,
		TimeoutMS: s.TimeoutMS, MaxLatencyMS: s.MaxLatencyMS, MaxPacketLoss: s.MaxPacketLoss,
		BlacklistMinutes: s.BlacklistMinutes, IncludeIPv6: s.IncludeIPv6, IncludeBlocked: s.IncludeBlocked,
	}
}

// BlacklistRecheckSettings controls the only automatic blacklist recovery flow.
type BlacklistRecheckSettings struct {
	Enabled         bool       `json:"enabled"`
	CronExpression  string     `json:"cron_expression"`
	Timezone        string     `json:"timezone"`
	DueOnly         bool       `json:"due_only"`
	Fraction        float64    `json:"fraction"`
	MaxTargets      int        `json:"max_targets"`
	SkipIfRunning   bool       `json:"skip_if_running"`
	Attempts        int        `json:"attempts"`
	TimeoutMS       int        `json:"timeout_ms"`
	MaxLatencyMS    float64    `json:"max_latency_ms"`
	MaxPacketLoss   float64    `json:"max_packet_loss"`
	RetryMinutes    int        `json:"retry_minutes"`
	NextRunAt       time.Time  `json:"next_run_at"`
	LastRunAt       *time.Time `json:"last_run_at,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	EligibleTargets int        `json:"eligible_targets"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type UpdateBlacklistRecheckSettingsRequest struct {
	Enabled        bool    `json:"enabled"`
	CronExpression string  `json:"cron_expression"`
	Timezone       string  `json:"timezone"`
	DueOnly        bool    `json:"due_only"`
	Fraction       float64 `json:"fraction"`
	MaxTargets     int     `json:"max_targets"`
	SkipIfRunning  bool    `json:"skip_if_running"`
	Attempts       int     `json:"attempts"`
	TimeoutMS      int     `json:"timeout_ms"`
	MaxLatencyMS   float64 `json:"max_latency_ms"`
	MaxPacketLoss  float64 `json:"max_packet_loss"`
	RetryMinutes   int     `json:"retry_minutes"`
}

type BlacklistRecheckResult struct {
	Jobs    int      `json:"jobs"`
	Targets int      `json:"targets"`
	JobIDs  []string `json:"job_ids"`
	Skipped bool     `json:"skipped"`
	Reason  string   `json:"reason,omitempty"`
}

type SourceSyncSchedule struct {
	Source         string     `json:"source"`
	Name           string     `json:"name"`
	Enabled        bool       `json:"enabled"`
	CronExpression string     `json:"cron_expression"`
	Timezone       string     `json:"timezone"`
	RunOnStartup   bool       `json:"run_on_startup"`
	NextRunAt      time.Time  `json:"next_run_at"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type UpdateSourceSyncScheduleRequest struct {
	Enabled        bool   `json:"enabled"`
	CronExpression string `json:"cron_expression"`
	Timezone       string `json:"timezone"`
	RunOnStartup   bool   `json:"run_on_startup"`
}

type AutomationRun struct {
	ID             string          `json:"id"`
	AutomationType string          `json:"automation_type"`
	AutomationKey  string          `json:"automation_key"`
	Name           string          `json:"name"`
	Trigger        string          `json:"trigger"`
	Status         string          `json:"status"`
	ConfigSnapshot json.RawMessage `json:"config_snapshot"`
	Summary        json.RawMessage `json:"summary"`
	Error          string          `json:"error,omitempty"`
	StartedAt      time.Time       `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
}

type StartAutomationRunRequest struct {
	AutomationType string
	AutomationKey  string
	Name           string
	Trigger        string
	ConfigSnapshot json.RawMessage
}

const (
	RoleAdmin  = "admin"
	RoleViewer = "viewer"
)

type User struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Enabled     bool       `json:"enabled"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UserCredential struct {
	User
	PasswordHash string `json:"-"`
}

type CreateUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Enabled     bool   `json:"enabled"`
}

type ResetUserPasswordRequest struct {
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
