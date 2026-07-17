import { createContext, useContext } from 'react'
import type { LoginInput, User } from '@/features/auth/types'

export interface AuthContextValue {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
  canManage: boolean
  login: (input: LoginInput) => Promise<User>
  logout: () => Promise<void>
  loginPending: boolean
  logoutPending: boolean
}

export const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used inside AuthProvider')
  return value
}
