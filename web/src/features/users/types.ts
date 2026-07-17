import type { User, UserRole } from '@/features/auth/types'

export type { User, UserRole }

export interface CreateUserInput {
  username: string
  display_name: string
  password: string
  role: UserRole
}

export interface UpdateUserInput {
  display_name: string
  role: UserRole
  enabled: boolean
}
