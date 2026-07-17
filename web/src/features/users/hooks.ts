import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import * as usersApi from '@/features/users/api'
import type { UpdateUserInput } from '@/features/users/types'

export const usersQueryKey = ['users'] as const

export function useUsers() {
  return useQuery({ queryKey: usersQueryKey, queryFn: usersApi.listUsers })
}

export function useCreateUser() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: usersApi.createUser,
    onSuccess: async () => { toast.success('账号已创建'); await client.invalidateQueries({ queryKey: usersQueryKey }) },
    onError: (error: Error) => toast.error(error.message),
  })
}

export function useUpdateUser() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateUserInput }) => usersApi.updateUser(id, input),
    onSuccess: async () => { toast.success('账号权限已更新'); await Promise.all([client.invalidateQueries({ queryKey: usersQueryKey }), client.invalidateQueries({ queryKey: ['auth', 'me'] })]) },
    onError: (error: Error) => toast.error(error.message),
  })
}

export function useResetUserPassword() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) => usersApi.resetPassword(id, password),
    onSuccess: async () => { toast.success('密码已重置，原有会话已失效'); await client.invalidateQueries({ queryKey: usersQueryKey }) },
    onError: (error: Error) => toast.error(error.message),
  })
}

export function useDeleteUser() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: usersApi.deleteUser,
    onSuccess: async () => { toast.success('账号已删除'); await client.invalidateQueries({ queryKey: usersQueryKey }) },
    onError: (error: Error) => toast.error(error.message),
  })
}
