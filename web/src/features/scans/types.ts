export interface ScanJob {
  id: string
  name: string
  kind: string
  status: string
  sampling_mode: 'count' | 'one_per_prefix'
  scheme: string
  hostname: string
  path: string
  port: number
  attempts: number
  timeout_ms: number
  max_latency_ms: number
  max_packet_loss: number
  blacklist_minutes: number
  total_targets: number
  completed_targets: number
  success_targets: number
  failed_targets: number
  progress: number
  created_at: string
  started_at?: string
  finished_at?: string
}

export interface CreateScanJob {
  name: string
  agent_ids: string[]
  sampling_mode: 'count' | 'one_per_prefix'
  target_count: number
  scheme: 'http' | 'https'
  hostname: string
  path: string
  port: number
  attempts: number
  timeout_ms: number
  max_latency_ms: number
  max_packet_loss: number
  blacklist_minutes: number
  include_ipv6: boolean
  include_blocked: boolean
}
