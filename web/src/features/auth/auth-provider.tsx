import { useEffect, useMemo, type PropsWithChildren } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as authApi from '@/features/auth/api'
import type { User } from '@/features/auth/types'
import { AuthContext, type AuthContextValue } from '@/features/auth/auth-context'

const authQueryKey = ['auth', 'me'] as const

export function AuthProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient()
  const currentUser = useQuery<User | null>({
    queryKey: authQueryKey,
    queryFn: authApi.getCurrentUser,
    retry: false,
    staleTime: 60_000,
  })
  const loginMutation = useMutation({
    mutationFn: authApi.login,
    onSuccess: (user) => queryClient.setQueryData(authQueryKey, user),
  })
  const logoutMutation = useMutation({
    mutationFn: authApi.logout,
    onSettled: () => {
      queryClient.setQueryData(authQueryKey, null)
      queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== 'auth' })
    },
  })

  useEffect(() => {
    const handleUnauthorized = () => {
      queryClient.setQueryData(authQueryKey, null)
      queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== 'auth' })
    }
    window.addEventListener('cfscan:unauthorized', handleUnauthorized)
    return () => window.removeEventListener('cfscan:unauthorized', handleUnauthorized)
  }, [queryClient])

  const user = currentUser.data ?? null
  const value = useMemo<AuthContextValue>(() => ({
    user,
    isLoading: currentUser.isPending,
    isAuthenticated: Boolean(user),
    canManage: user?.role === 'admin',
    login: loginMutation.mutateAsync,
    logout: logoutMutation.mutateAsync,
    loginPending: loginMutation.isPending,
    logoutPending: logoutMutation.isPending,
  }), [currentUser.isPending, loginMutation.isPending, loginMutation.mutateAsync, logoutMutation.isPending, logoutMutation.mutateAsync, user])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

