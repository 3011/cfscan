export type UserRole = 'admin' | 'viewer'

export interface User {
  id: string
  username: string
  display_name: string
  role: UserRole
  enabled: boolean
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface LoginInput {
  username: string
  password: string
}
