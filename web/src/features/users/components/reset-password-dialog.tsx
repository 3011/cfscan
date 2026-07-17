import { useEffect } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Loader2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useResetUserPassword } from '@/features/users/hooks'
import { useAuth } from '@/features/auth/auth-context'
import { resetPasswordSchema, type ResetPasswordValues } from '@/features/users/schema'
import type { User } from '@/features/users/types'

export function ResetPasswordDialog({ user, open, onOpenChange }: { user: User | null; open: boolean; onOpenChange: (open: boolean) => void }) {
  const auth = useAuth()
  const reset = useResetUserPassword()
  const form = useForm<ResetPasswordValues>({ resolver: zodResolver(resetPasswordSchema), defaultValues: { password: '', confirm_password: '' } })
  useEffect(() => { if (open) form.reset({ password: '', confirm_password: '' }) }, [form, open])
  async function submit(values: ResetPasswordValues) {
    if (!user) return
    await reset.mutateAsync({ id: user.id, password: values.password })
    onOpenChange(false)
    if (user.id === auth.user?.id) window.dispatchEvent(new Event('cfscan:unauthorized'))
  }
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent><DialogHeader><DialogTitle>重置 {user?.display_name} 的密码</DialogTitle><DialogDescription>保存后该账号所有已登录会话都会立即失效。</DialogDescription></DialogHeader><Form {...form}><form id="reset-password-form" className="space-y-4" onSubmit={form.handleSubmit(submit)}><FormField control={form.control} name="password" render={({ field }) => <FormItem><FormLabel>新密码</FormLabel><FormControl><Input type="password" autoComplete="new-password" {...field} /></FormControl><FormMessage /></FormItem>} /><FormField control={form.control} name="confirm_password" render={({ field }) => <FormItem><FormLabel>确认新密码</FormLabel><FormControl><Input type="password" autoComplete="new-password" {...field} /></FormControl><FormMessage /></FormItem>} /></form></Form><DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button><Button type="submit" form="reset-password-form" disabled={reset.isPending}>{reset.isPending ? <Loader2 className="animate-spin" /> : null}重置密码</Button></DialogFooter></DialogContent></Dialog>
}
