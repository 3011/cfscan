import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { overviewQueryKey } from '@/features/dashboard/hooks'
import { sourcesApi } from '@/features/sources/api'
import type { CreateASNSource, UpdateASNSource } from '@/features/sources/types'

export const officialSourceQueryKey = ['sources', 'official'] as const
export const asnSourcesQueryKey = ['sources', 'asns'] as const

function useInvalidateSources() {
  const client = useQueryClient()
  return () => Promise.all([
    client.invalidateQueries({ queryKey: officialSourceQueryKey }),
    client.invalidateQueries({ queryKey: asnSourcesQueryKey }),
    client.invalidateQueries({ queryKey: overviewQueryKey }),
  ])
}

export function useOfficialSource() {
  return useQuery({ queryKey: officialSourceQueryKey, queryFn: sourcesApi.getOfficial, refetchInterval: 30_000 })
}

export function useASNSources() {
  return useQuery({ queryKey: asnSourcesQueryKey, queryFn: sourcesApi.getASNs, refetchInterval: 30_000 })
}

export function useSyncOfficial() {
  const invalidate = useInvalidateSources()
  return useMutation({
    mutationFn: sourcesApi.syncOfficial,
    onSuccess: async (source) => {
      await invalidate()
      toast.success('官方地址段已同步', { description: `当前共 ${source.prefix_count} 个前缀。` })
    },
    onError: (error) => toast.error('官方地址段同步失败', { description: error.message }),
  })
}

export function useSyncAllASNs() {
  const invalidate = useInvalidateSources()
  return useMutation({
    mutationFn: sourcesApi.syncAllASNs,
    onSuccess: async (summary) => {
      await invalidate()
      if (summary.failed > 0) toast.warning('ASN 同步完成，但存在失败项', { description: `成功 ${summary.synced}，失败 ${summary.failed}。` })
      else toast.success('ASN 数据源已全部同步', { description: `已同步 ${summary.synced} 个 ASN。` })
    },
    onError: (error) => toast.error('ASN 同步失败', { description: error.message }),
  })
}

export function useSyncASN() {
  const invalidate = useInvalidateSources()
  return useMutation({
    mutationFn: sourcesApi.syncASN,
    onSuccess: async (item) => {
      await invalidate()
      toast.success(`AS${item.asn} 已同步`, { description: `获取 ${item.prefix_count} 个前缀。` })
    },
    onError: (error) => toast.error('ASN 同步失败', { description: error.message }),
  })
}

export function useCreateASN() {
  const invalidate = useInvalidateSources()
  return useMutation({
    mutationFn: (input: CreateASNSource) => sourcesApi.createASN(input),
    onSuccess: async (item) => {
      await invalidate()
      toast.success(`AS${item.asn} 已添加`)
    },
    onError: (error) => toast.error('添加 ASN 失败', { description: error.message }),
  })
}

export function useUpdateASN() {
  const invalidate = useInvalidateSources()
  return useMutation({
    mutationFn: ({ asn, input }: { asn: number; input: UpdateASNSource }) => sourcesApi.updateASN(asn, input),
    onSuccess: async (item) => {
      await invalidate()
      toast.success(`AS${item.asn} 已${item.enabled ? '启用' : '停用'}`)
    },
    onError: (error) => toast.error('更新 ASN 失败', { description: error.message }),
  })
}

export function useDeleteASN() {
  const invalidate = useInvalidateSources()
  return useMutation({
    mutationFn: sourcesApi.deleteASN,
    onSuccess: async () => {
      await invalidate()
      toast.success('ASN 数据源已删除')
    },
    onError: (error) => toast.error('删除 ASN 失败', { description: error.message }),
  })
}
