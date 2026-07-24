import { createQueryString, request } from '@/lib/http'
import type { IPTrend, LeagueCandidate, LeagueDashboard, TrendTimeRange } from '@/features/league/types'

export function getLeagueDashboard(agentId = '') {
  return request<LeagueDashboard>(`/api/v1/league${createQueryString({ agent_id: agentId, limit: 500 })}`)
}

export function getIPTrend(candidate: LeagueCandidate, timeRange: TrendTimeRange) {
  return request<IPTrend>(`/api/v1/results/trend${createQueryString({
    agent_id: candidate.agent_id,
    target_ip: candidate.target_ip,
    scheme: candidate.scheme,
    hostname: candidate.hostname,
    path: candidate.path,
    port: candidate.port,
    attempts: candidate.attempts,
    timeout_ms: candidate.timeout_ms,
    time_range: timeRange,
  })}`)
}
