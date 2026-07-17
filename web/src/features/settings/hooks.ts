import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { jobsQueryKey } from '@/features/scans/hooks'
import { settingsApi } from '@/features/settings/api'
import type { UpsertScanSchedule } from '@/features/settings/types'

export const schedulesQueryKey = ['scan-schedules'] as const

export function useScanSchedules() {
  return useQuery({ queryKey: schedulesQueryKey, queryFn: settingsApi.getSchedules, refetchInterval: 15_000 })
}

export function useCreateSchedule() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.createSchedule,
    onSuccess: async (item) => {
      await client.invalidateQueries({ queryKey: schedulesQueryKey })
      toast.success('定时计划已创建', { description: `下次执行：${new Date(item.next_run_at).toLocaleString('zh-CN')}` })
    },
    onError: (error) => toast.error('创建定时计划失败', { description: error.message }),
  })
}

export function useUpdateSchedule() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpsertScanSchedule }) => settingsApi.updateSchedule(id, input),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: schedulesQueryKey })
      toast.success('定时计划已更新')
    },
    onError: (error) => toast.error('更新定时计划失败', { description: error.message }),
  })
}

export function useDeleteSchedule() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.deleteSchedule,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: schedulesQueryKey })
      toast.success('定时计划已删除')
    },
    onError: (error) => toast.error('删除定时计划失败', { description: error.message }),
  })
}

export function useRunSchedule() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.runSchedule,
    onSuccess: async (job) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: schedulesQueryKey }),
        client.invalidateQueries({ queryKey: jobsQueryKey }),
      ])
      toast.success('计划已立即执行', { description: `${job.total_targets} 个 Agent 目标任务已进入队列。` })
    },
    onError: (error) => toast.error('执行定时计划失败', { description: error.message }),
  })
}

export const blacklistAutomationQueryKey = ['automation', 'blacklist-recheck'] as const
export const sourceSyncAutomationQueryKey = ['automation', 'source-syncs'] as const
export const automationRunsQueryKey = ['automation', 'runs'] as const

export function useBlacklistRecheckSettings() {
  return useQuery({ queryKey: blacklistAutomationQueryKey, queryFn: settingsApi.getBlacklistRecheck, refetchInterval: 15_000 })
}

export function useUpdateBlacklistRecheckSettings() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.updateBlacklistRecheck,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: blacklistAutomationQueryKey })
      toast.success('黑名单复查设置已保存')
    },
    onError: (error) => toast.error('保存黑名单复查设置失败', { description: error.message }),
  })
}

export function useRunBlacklistRecheckAutomation() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.runBlacklistRecheck,
    onSuccess: async (result) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: blacklistAutomationQueryKey }),
        client.invalidateQueries({ queryKey: automationRunsQueryKey }),
        client.invalidateQueries({ queryKey: jobsQueryKey }),
      ])
      if (result.skipped) toast.info('本轮黑名单复查已跳过', { description: result.reason })
      else toast.success('黑名单复查已创建', { description: `${result.targets} 个目标，${result.jobs} 个 Agent 任务组。` })
    },
    onError: (error) => toast.error('执行黑名单复查失败', { description: error.message }),
  })
}

export function useSourceSyncSchedules() {
  return useQuery({ queryKey: sourceSyncAutomationQueryKey, queryFn: settingsApi.getSourceSyncs, refetchInterval: 15_000 })
}

export function useUpdateSourceSyncSchedule() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ source, input }: { source: string; input: import('@/features/settings/types').UpdateSourceSyncSchedule }) => settingsApi.updateSourceSync(source, input),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: sourceSyncAutomationQueryKey })
      toast.success('数据源同步设置已保存')
    },
    onError: (error) => toast.error('保存数据源同步设置失败', { description: error.message }),
  })
}

export function useRunSourceSyncSchedule() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: settingsApi.runSourceSync,
    onSuccess: async (_, source) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: sourceSyncAutomationQueryKey }),
        client.invalidateQueries({ queryKey: automationRunsQueryKey }),
        client.invalidateQueries({ queryKey: ['sources'] }),
        client.invalidateQueries({ queryKey: ['colo-locations'] }),
      ])
      toast.success(source === 'official' ? '官方地址段同步完成' : source === 'asn' ? 'ASN 前缀同步完成' : 'colo 位置同步完成')
    },
    onError: (error) => toast.error('数据源同步失败', { description: error.message }),
  })
}

export function useAutomationRuns() {
  return useQuery({ queryKey: automationRunsQueryKey, queryFn: () => settingsApi.getAutomationRuns(200), refetchInterval: 15_000 })
}
