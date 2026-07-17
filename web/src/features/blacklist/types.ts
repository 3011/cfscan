export interface BlacklistEntry {
  agent_id: string
  agent_name: string
  region: string
  continent: string
  target_ip: string
  reason: string
  failure_count: number
  blocked_at: string
  retry_after: string
  updated_at: string
}
