import { request, type ItemsResponse } from '@/lib/http'
import type { BlacklistEntry } from '@/features/blacklist/types'
import type { BlacklistRecheckResult } from '@/features/settings/types'

export const blacklistApi = {
  getEntries: (limit = 500) => request<ItemsResponse<BlacklistEntry>>(`/api/v1/blacklist?limit=${limit}`),
  recheck: () => request<BlacklistRecheckResult>('/api/v1/blacklist/recheck', { method: 'POST', body: '{}' }),
}
