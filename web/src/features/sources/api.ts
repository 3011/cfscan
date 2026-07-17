import { request, type ItemsResponse } from '@/lib/http'
import type { ASNSource, ASNSyncSummary, CreateASNSource, SourceStatus, UpdateASNSource } from '@/features/sources/types'

export const sourcesApi = {
  getOfficial: () => request<SourceStatus>('/api/v1/sources/cloudflare'),
  syncOfficial: () => request<SourceStatus>('/api/v1/sources/cloudflare/sync', { method: 'POST', body: '{}' }),
  getASNs: () => request<ItemsResponse<ASNSource>>('/api/v1/sources/asns'),
  createASN: (input: CreateASNSource) => request<ASNSource>('/api/v1/sources/asns', { method: 'POST', body: JSON.stringify(input) }),
  updateASN: (asn: number, input: UpdateASNSource) => request<ASNSource>(`/api/v1/sources/asns/${asn}`, { method: 'PATCH', body: JSON.stringify(input) }),
  deleteASN: (asn: number) => request<void>(`/api/v1/sources/asns/${asn}`, { method: 'DELETE' }),
  syncASN: (asn: number) => request<ASNSource>(`/api/v1/sources/asns/${asn}/sync`, { method: 'POST', body: '{}' }),
  syncAllASNs: () => request<ASNSyncSummary>('/api/v1/sources/asns/sync', { method: 'POST', body: '{}' }),
}
