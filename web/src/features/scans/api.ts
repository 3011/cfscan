import { request, type ItemsResponse } from '@/lib/http'
import type { CreateScanJob, ScanJob } from '@/features/scans/types'

export const scansApi = {
  getJobs: (limit = 100) => request<ItemsResponse<ScanJob>>(`/api/v1/jobs?limit=${limit}`),
  createJob: (input: CreateScanJob) => request<ScanJob>('/api/v1/jobs', { method: 'POST', body: JSON.stringify(input) }),
  stopJob: (id: string) => request<ScanJob>(`/api/v1/jobs/${id}/stop`, { method: 'POST', body: '{}' }),
}
