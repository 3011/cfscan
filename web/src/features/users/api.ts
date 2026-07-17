import { request, type ItemsResponse } from '@/lib/http'
import type { CreateUserInput, UpdateUserInput, User } from '@/features/users/types'

export function listUsers() {
  return request<ItemsResponse<User>>('/api/v1/users/')
}

export function createUser(input: CreateUserInput) {
  return request<User>('/api/v1/users/', { method: 'POST', body: JSON.stringify(input) })
}

export function updateUser(userID: string, input: UpdateUserInput) {
  return request<User>(`/api/v1/users/${userID}`, { method: 'PUT', body: JSON.stringify(input) })
}

export function resetPassword(userID: string, password: string) {
  return request<void>(`/api/v1/users/${userID}/reset-password`, { method: 'POST', body: JSON.stringify({ password }) })
}

export function deleteUser(userID: string) {
  return request<void>(`/api/v1/users/${userID}`, { method: 'DELETE' })
}
