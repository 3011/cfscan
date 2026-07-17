import { createQueryString, request, type ItemsResponse } from '@/lib/http'
import type { ResultColoFacet, ResultFacetFilters, ResultFilters, ResultJobFacet, ResultJobFilters, ResultPage } from '@/features/results/types'

export function getResults(filters: ResultFilters = {}) {
  return request<ResultPage>(`/api/v1/results${createQueryString(filters)}`)
}

export function getResultFacets(filters: ResultFacetFilters = {}) {
  return request<ItemsResponse<ResultColoFacet>>(`/api/v1/results/facets${createQueryString(filters)}`)
}

export function getResultJobs(filters: ResultJobFilters = {}) {
  return request<ItemsResponse<ResultJobFacet>>(`/api/v1/results/jobs${createQueryString(filters)}`)
}
