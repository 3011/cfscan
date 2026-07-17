import { useEffect } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Loader2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { useAuth } from '@/features/auth/auth-context'
import { createUserSchema, updateUserSchema, type CreateUserValues, type UpdateUserValues } from '@/features/users/schema'
import { useCreateUser, useUpdateUser } from '@/features/users/hooks'
import type { User } from '@/features/users/types'

interface UserDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user?: User | null
}

export function UserDialog({ open, onOpenChange, user }: UserDialogProps) {
  const auth = useAuth()
  const create = useCreateUser()
  const update = useUpdateUser()
  const editingSelf = user?.id === auth.user?.id
  const form = useForm<CreateUserValues | UpdateUserValues>({
    resolver: zodResolver(user ? updateUserSchema : createUserSchema),
    defaultValues: user
      ? { display_name: user.display_name, role: user.role, enabled: user.enabled }
      : { username: '', display_name: '', password: '', role: 'viewer' },
  })

  useEffect(() => {
    if (!open) return
    form.reset(user
      ? { display_name: user.display_name, role: user.role, enabled: user.enabled }
      : { username: '', display_name: '', password: '', role: 'viewer' })
  }, [form, open, user])

  async function submit(values: CreateUserValues | UpdateUserValues) {
    if (user) await update.mutateAsync({ id: user.id, input: values as UpdateUserValues })
    else await create.mutateAsync(values as CreateUserValues)
    onOpenChange(false)
  }

  const pending = create.isPending || update.isPending
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader><DialogTitle>{user ? '编辑账号' : '创建账号'}</DialogTitle><DialogDescription>{user ? '调整显示名称、角色和账号状态。' : '新账号创建后可以立即登录管理平台。'}</DialogDescription></DialogHeader>
        <Form {...form}>
          <form id="user-form" className="space-y-4" onSubmit={form.handleSubmit(submit)}>
            {!user ? <>
              <FormField control={form.control} name="username" render={({ field }) => <FormItem><FormLabel>用户名</FormLabel><FormControl><Input autoComplete="off" placeholder="viewer01" {...field} /></FormControl><FormDescription>创建后不可修改。</FormDescription><FormMessage /></FormItem>} />
              <FormField control={form.control} name="password" render={({ field }) => <FormItem><FormLabel>初始密码</FormLabel><FormControl><Input type="password" autoComplete="new-password" {...field} /></FormControl><FormMessage /></FormItem>} />
            </> : null}
            <FormField control={form.control} name="display_name" render={({ field }) => <FormItem><FormLabel>显示名称</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>} />
            <FormField control={form.control} name="role" render={({ field }) => <FormItem><FormLabel>权限</FormLabel><Select items={{ admin: '管理员', viewer: '查看者' }} value={field.value} onValueChange={field.onChange} disabled={editingSelf}><FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl><SelectContent><SelectItem value="admin">管理员</SelectItem><SelectItem value="viewer">查看者</SelectItem></SelectContent></Select><FormDescription>{editingSelf ? '不能修改当前登录账号的管理员权限。' : '管理员可执行所有操作；查看者只能读取数据。'}</FormDescription><FormMessage /></FormItem>} />
            {user ? <FormField control={form.control} name="enabled" render={({ field }) => <FormItem className="flex items-center justify-between rounded-lg border p-4"><div><FormLabel>启用账号</FormLabel><FormDescription>{editingSelf ? '当前登录账号不能停用自己。' : '停用后已有会话会立即失效。'}</FormDescription></div><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} disabled={editingSelf} /></FormControl></FormItem>} /> : null}
          </form>
        </Form>
        <DialogFooter><Button type="button" variant="outline" onClick={() => onOpenChange(false)}>取消</Button><Button type="submit" form="user-form" disabled={pending}>{pending ? <Loader2 className="animate-spin" /> : null}{user ? '保存修改' : '创建账号'}</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
