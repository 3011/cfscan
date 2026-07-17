import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { blacklistApi } from '@/features/blacklist/api'
import { jobsQueryKey } from '@/features/scans/hooks'
import { overviewQueryKey } from '@/features/dashboard/hooks'

export const blacklistQueryKey = ['blacklist'] as const

export function useBlacklist() {
  return useQuery({ queryKey: blacklistQueryKey, queryFn: () => blacklistApi.getEntries(500), refetchInterval: 10_000 })
}

export function useRecheckBlacklist() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: blacklistApi.recheck,
    onSuccess: async (result) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: blacklistQueryKey }),
        client.invalidateQueries({ queryKey: jobsQueryKey }),
        client.invalidateQueries({ queryKey: overviewQueryKey }),
      ])
      if (result.skipped) toast.info('本轮黑名单复检已跳过', { description: result.reason })
      else toast.success('黑名单复检已安排', { description: `已创建 ${result.targets} 个重新扫描目标。` })
    },
    onError: (error) => toast.error('安排复检失败', { description: error.message }),
  })
}
