import { useQuery } from '@tanstack/react-query'
import { getOverview } from '@/features/dashboard/api'

export const overviewQueryKey = ['overview'] as const

export function useOverview() {
  return useQuery({ queryKey: overviewQueryKey, queryFn: getOverview, refetchInterval: 5_000 })
}
