import { request, type ItemsResponse } from '@/lib/http'
import type { Agent } from '@/features/agents/types'

export function getAgents() {
  return request<ItemsResponse<Agent>>('/api/v1/agents')
}
