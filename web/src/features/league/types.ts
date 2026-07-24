export type PrefixTier = 'observation' | 'challenger' | 'champion'

export interface PrefixLeagueEntry {
  agent_id: string
  agent_name: string
  region: string
  continent: string
  prefix_cidr: string
  scheme: string
  hostname: string
  path: string
  port: number
  attempts: number
  timeout_ms: number
  tier: PrefixTier
  active: boolean
  sample_count: number
  distinct_ip_count: number
  availability_rate: number
  latency_p95_ms: number
  packet_loss_avg: number
  recent_sample_count: number
  recent_availability_rate: number
  recent_latency_p95_ms: number
  recent_packet_loss_avg: number
  bad_streak: number
  last_result_at?: string
  last_scheduled_at?: string
  last_evaluated_at?: string
  updated_at: string
}

export interface LeagueCandidate {
  agent_id: string
  agent_name: string
  region: string
  continent: string
  prefix_cidr: string
  tier: PrefixTier
  scheme: string
  hostname: string
  path: string
  port: number
  attempts: number
  timeout_ms: number
  target_ip: string
  colo: string
  sample_count: number
  availability_rate: number
  latency_p95_ms: number
  packet_loss_avg: number
  last_scanned_at: string
}

export interface LeagueDashboard {
  summary: {
    observation_prefixes: number
    challenger_prefixes: number
    champion_prefixes: number
    candidate_ips: number
  }
  prefixes: PrefixLeagueEntry[]
  candidates: LeagueCandidate[]
}

export type TrendTimeRange = '24h' | '7d' | '30d'

export interface IPTrendPoint {
  scanned_at: string
  available: boolean
  latency_ms: number
  packet_loss: number
  tcp_connect_ms: number
  tls_handshake_ms: number
  ttfb_ms: number
  colo: string
}

export interface IPTrend {
  agent_id: string
  agent_name: string
  target_ip: string
  summary: {
    sample_count: number
    availability_rate: number
    latency_p50_ms: number
    latency_p95_ms: number
    packet_loss_avg: number
    latest_colo: string
  }
  points: IPTrendPoint[]
}
