import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { getResultFacets, getResultJobs, getResults } from '@/features/results/api'
import type { ResultFacetFilters, ResultFilters, ResultJobFilters } from '@/features/results/types'

export const resultsQueryKey = ['scan-results'] as const

export function useScanResults(filters: ResultFilters, refetchInterval = 10_000) {
  return useQuery({
    queryKey: [...resultsQueryKey, filters],
    queryFn: () => getResults(filters),
    refetchInterval,
    placeholderData: keepPreviousData,
  })
}

export function useResultFacets(filters: ResultFacetFilters, refetchInterval = 30_000) {
  return useQuery({
    queryKey: ['scan-result-facets', filters],
    queryFn: () => getResultFacets(filters),
    refetchInterval,
    placeholderData: keepPreviousData,
  })
}

export function useResultJobs(filters: ResultJobFilters, refetchInterval = 30_000) {
  return useQuery({
    queryKey: ['scan-result-jobs', filters],
    queryFn: () => getResultJobs(filters),
    refetchInterval,
    placeholderData: keepPreviousData,
  })
}
