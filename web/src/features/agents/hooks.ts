import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  approveAgentEnrollment,
  createPreauthorizedEnrollment,
  getAgentEnrollment,
  getAgentEnrollmentConfig,
  getAgentEnrollments,
  getAgents,
  rejectAgentEnrollment,
} from '@/features/agents/api'
import type { ApproveAgentEnrollmentInput, CreatePreauthorizedEnrollmentInput, EnrollmentLocator } from '@/features/agents/types'

export const agentsQueryKey = ['agents'] as const
export const agentEnrollmentsQueryKey = ['agent-enrollments'] as const
export const agentEnrollmentConfigQueryKey = ['agent-enrollments', 'config'] as const

export function useAgents(options?: { refetchInterval?: number }) {
  return useQuery({ queryKey: agentsQueryKey, queryFn: getAgents, refetchInterval: options?.refetchInterval })
}

export function useAgentEnrollments(options?: { refetchInterval?: number }) {
  return useQuery({ queryKey: agentEnrollmentsQueryKey, queryFn: getAgentEnrollments, refetchInterval: options?.refetchInterval })
}

export function useAgentEnrollmentConfig() {
  return useQuery({ queryKey: agentEnrollmentConfigQueryKey, queryFn: getAgentEnrollmentConfig, staleTime: 60_000 })
}

export function useAgentEnrollment(locator: EnrollmentLocator) {
  return useQuery({
    queryKey: [...agentEnrollmentsQueryKey, locator.kind, locator.value],
    queryFn: () => getAgentEnrollment(locator),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'pending' || status === 'approved' ? 2_000 : false
    },
  })
}

export function useApproveAgentEnrollment(locator: EnrollmentLocator) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: ApproveAgentEnrollmentInput) => approveAgentEnrollment(locator, input),
    onSuccess: async (item) => {
      toast.success('配对已批准', { description: `正在等待 ${item.name} 完成连接。` })
      await Promise.all([
        client.invalidateQueries({ queryKey: agentEnrollmentsQueryKey }),
        client.invalidateQueries({ queryKey: agentsQueryKey }),
      ])
    },
    onError: (error: Error) => toast.error('批准失败', { description: error.message }),
  })
}

export function useRejectAgentEnrollment(locator: EnrollmentLocator) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: () => rejectAgentEnrollment(locator),
    onSuccess: async () => {
      toast.success('配对请求已拒绝')
      await client.invalidateQueries({ queryKey: agentEnrollmentsQueryKey })
    },
    onError: (error: Error) => toast.error('拒绝失败', { description: error.message }),
  })
}

export function useCreatePreauthorizedEnrollment() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: CreatePreauthorizedEnrollmentInput) => createPreauthorizedEnrollment(input),
    onSuccess: async () => {
      toast.success('一次性部署命令已生成')
      await client.invalidateQueries({ queryKey: agentEnrollmentsQueryKey })
    },
    onError: (error: Error) => toast.error('生成部署命令失败', { description: error.message }),
  })
}
