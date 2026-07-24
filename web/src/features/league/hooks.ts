import { keepPreviousData, useQuery } from '@tanstack/react-query'
import type { LeagueCandidate, LeagueDashboardFilters, TrendTimeRange } from '@/features/league/types'
import { getIPTrend, getLeagueDashboard } from '@/features/league/api'

export function useLeagueDashboard(filters: LeagueDashboardFilters) {
  return useQuery({
    queryKey: ['prefix-league', filters],
    queryFn: () => getLeagueDashboard(filters),
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
