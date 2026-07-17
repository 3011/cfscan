import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { overviewQueryKey } from '@/features/dashboard/hooks'
import { scansApi } from '@/features/scans/api'
import type { CreateScanJob } from '@/features/scans/types'

export const jobsQueryKey = ['scan-jobs'] as const

export function useScanJobs(limit = 100, refetchInterval = 3_000) {
  return useQuery({ queryKey: [...jobsQueryKey, limit], queryFn: () => scansApi.getJobs(limit), refetchInterval })
}

export function useCreateScanJob() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateScanJob) => scansApi.createJob(input),
    onSuccess: async (job) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: jobsQueryKey }),
        client.invalidateQueries({ queryKey: overviewQueryKey }),
      ])
      toast.success('扫描任务已创建', { description: `${job.total_targets} 个 Agent 目标任务已进入队列。` })
    },
    onError: (error) => toast.error('创建扫描任务失败', { description: error.message }),
  })
}

export function useStopScanJob() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: scansApi.stopJob,
    onSuccess: async (job) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: jobsQueryKey }),
        client.invalidateQueries({ queryKey: overviewQueryKey }),
      ])
      toast.success(job.status === 'stopped' ? '剩余任务已停止' : '任务状态已更新')
    },
    onError: (error) => toast.error('停止任务失败', { description: error.message }),
  })
}
