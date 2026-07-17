import { describe, expect, it } from 'vitest'
import { createUserSchema, resetPasswordSchema, updateUserSchema } from '@/features/users/schema'

describe('user schemas', () => {
  it('accepts admin and viewer accounts', () => {
    expect(createUserSchema.safeParse({ username: 'viewer-01', display_name: 'Viewer', password: 'password123', role: 'viewer' }).success).toBe(true)
    expect(updateUserSchema.safeParse({ display_name: 'Administrator', role: 'admin', enabled: true }).success).toBe(true)
  })

  it('rejects invalid usernames and short passwords', () => {
    expect(createUserSchema.safeParse({ username: 'Bad User', display_name: 'Bad', password: 'short', role: 'viewer' }).success).toBe(false)
  })

  it('requires matching reset passwords', () => {
    expect(resetPasswordSchema.safeParse({ password: 'password123', confirm_password: 'password124' }).success).toBe(false)
  })
})
