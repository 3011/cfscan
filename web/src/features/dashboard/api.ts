import { request } from '@/lib/http'
import type { Overview } from '@/features/dashboard/types'

export function getOverview() {
  return request<Overview>('/api/v1/overview')
}
