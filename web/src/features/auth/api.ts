import { request } from '@/lib/http'
import type { LoginInput, User } from '@/features/auth/types'

export function login(input: LoginInput) {
  return request<User>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(input) })
}

export function logout() {
  return request<void>('/api/v1/auth/logout', { method: 'POST' })
}

export function getCurrentUser() {
  return request<User>('/api/v1/auth/me')
}
