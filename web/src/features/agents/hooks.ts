import { useQuery } from '@tanstack/react-query'
import { getAgents } from '@/features/agents/api'

export const agentsQueryKey = ['agents'] as const

export function useAgents(options?: { refetchInterval?: number }) {
  return useQuery({
    queryKey: agentsQueryKey,
    queryFn: getAgents,
    refetchInterval: options?.refetchInterval,
  })
}
