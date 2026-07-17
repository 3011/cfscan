import type { CreateScanJob } from '@/features/scans/types'

export interface UpsertScanSchedule extends CreateScanJob {
  enabled: boolean
  cron_expression: string
  timezone: string
}

export interface ScanSchedule extends UpsertScanSchedule {
  id: string
  next_run_at: string
  last_run_at?: string
  last_job_id?: string
  last_error?: string
  created_at: string
  updated_at: string
}

export interface BlacklistRecheckSettings {
  enabled: boolean
  cron_expression: string
  timezone: string
  due_only: boolean
  fraction: number
  max_targets: number
  skip_if_running: boolean
  attempts: number
  timeout_ms: number
  max_latency_ms: number
  max_packet_loss: number
  retry_minutes: number
  next_run_at: string
  last_run_at?: string
  last_error?: string
  eligible_targets: number
  updated_at: string
}

export type UpdateBlacklistRecheckSettings = Omit<BlacklistRecheckSettings, 'next_run_at' | 'last_run_at' | 'last_error' | 'eligible_targets' | 'updated_at'>

export interface BlacklistRecheckResult {
  jobs: number
  targets: number
  job_ids: string[]
  skipped: boolean
  reason?: string
}

export interface SourceSyncSchedule {
  source: 'official' | 'asn' | 'colo'
  name: string
  enabled: boolean
  cron_expression: string
  timezone: string
  run_on_startup: boolean
  next_run_at: string
  last_run_at?: string
  last_error?: string
  updated_at: string
}

export interface UpdateSourceSyncSchedule {
  enabled: boolean
  cron_expression: string
  timezone: string
  run_on_startup: boolean
}

export interface AutomationRun {
  id: string
  automation_type: 'scan_schedule' | 'blacklist_recheck' | 'source_sync'
  automation_key: string
  name: string
  trigger: 'scheduled' | 'startup' | 'manual'
  status: 'running' | 'completed' | 'failed' | 'skipped'
  config_snapshot: Record<string, unknown>
  summary: Record<string, unknown>
  error?: string
  started_at: string
  finished_at?: string
}
