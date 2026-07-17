export type AgentStatus = 'online' | 'offline'

export interface Agent {
  id: string
  name: string
  region: string
  continent: string
  concurrency: number
  status: AgentStatus
  last_seen_at: string
  created_at: string
}
