import { keepPreviousData, useQuery } from '@tanstack/react-query'
import type { LeagueCandidate, TrendTimeRange } from '@/features/league/types'
import { getIPTrend, getLeagueDashboard } from '@/features/league/api'

export function useLeagueDashboard(agentId: string) {
  return useQuery({
    queryKey: ['prefix-league', agentId],
    queryFn: () => getLeagueDashboard(agentId),
    refetchInterval: 30_000,
    placeholderData: keepPreviousData,
  })
}

export function useIPTrend(candidate: LeagueCandidate, timeRange: TrendTimeRange) {
  return useQuery({
    queryKey: ['ip-trend', candidate, timeRange],
    queryFn: () => getIPTrend(candidate, timeRange),
    refetchInterval: 30_000,
  })
}
