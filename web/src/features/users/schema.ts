import { z } from 'zod'

const roleSchema = z.enum(['admin', 'viewer'])

export const createUserSchema = z.object({
  username: z.string().trim().min(3, '用户名至少 3 个字符').max(64, '用户名最多 64 个字符').regex(/^[a-z0-9._-]+$/, '仅支持小写字母、数字、点、下划线和连字符'),
  display_name: z.string().trim().min(1, '请输入显示名称').max(80, '显示名称最多 80 个字符'),
  password: z.string().min(8, '密码至少 8 个字符').max(128, '密码最多 128 个字符'),
  role: roleSchema,
})

export const updateUserSchema = z.object({
  display_name: z.string().trim().min(1, '请输入显示名称').max(80, '显示名称最多 80 个字符'),
  role: roleSchema,
  enabled: z.boolean(),
})

export const resetPasswordSchema = z.object({
  password: z.string().min(8, '密码至少 8 个字符').max(128, '密码最多 128 个字符'),
  confirm_password: z.string().min(1, '请再次输入密码'),
}).refine((value) => value.password === value.confirm_password, { path: ['confirm_password'], message: '两次输入的密码不一致' })

export type CreateUserValues = z.infer<typeof createUserSchema>
export type UpdateUserValues = z.infer<typeof updateUserSchema>
export type ResetPasswordValues = z.infer<typeof resetPasswordSchema>
