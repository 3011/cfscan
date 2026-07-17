import { request, type ItemsResponse } from '@/lib/http'
import type { ScanJob } from '@/features/scans/types'
import type { AutomationRun, BlacklistRecheckResult, BlacklistRecheckSettings, ScanSchedule, SourceSyncSchedule, UpdateBlacklistRecheckSettings, UpdateSourceSyncSchedule, UpsertScanSchedule } from '@/features/settings/types'

export const settingsApi = {
  getSchedules: () => request<ItemsResponse<ScanSchedule>>('/api/v1/schedules'),
  createSchedule: (input: UpsertScanSchedule) => request<ScanSchedule>('/api/v1/schedules', { method: 'POST', body: JSON.stringify(input) }),
  updateSchedule: (id: string, input: UpsertScanSchedule) => request<ScanSchedule>(`/api/v1/schedules/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteSchedule: (id: string) => request<void>(`/api/v1/schedules/${id}`, { method: 'DELETE' }),
  runSchedule: (id: string) => request<ScanJob>(`/api/v1/schedules/${id}/run`, { method: 'POST', body: '{}' }),
  getBlacklistRecheck: () => request<BlacklistRecheckSettings>('/api/v1/automation/blacklist-recheck'),
  updateBlacklistRecheck: (input: UpdateBlacklistRecheckSettings) => request<BlacklistRecheckSettings>('/api/v1/automation/blacklist-recheck', { method: 'PUT', body: JSON.stringify(input) }),
  runBlacklistRecheck: () => request<BlacklistRecheckResult>('/api/v1/automation/blacklist-recheck/run', { method: 'POST', body: '{}' }),
  getSourceSyncs: () => request<ItemsResponse<SourceSyncSchedule>>('/api/v1/automation/source-syncs'),
  updateSourceSync: (source: string, input: UpdateSourceSyncSchedule) => request<SourceSyncSchedule>(`/api/v1/automation/source-syncs/${source}`, { method: 'PUT', body: JSON.stringify(input) }),
  runSourceSync: (source: string) => request<{ summary: Record<string, unknown> }>(`/api/v1/automation/source-syncs/${source}/run`, { method: 'POST', body: '{}' }),
  getAutomationRuns: (limit = 100) => request<ItemsResponse<AutomationRun>>(`/api/v1/automation/runs?limit=${limit}`),
}
