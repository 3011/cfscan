import { Plus, ShieldCheck, UserCheck, Users } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ErrorState } from '@/components/shared/error-state'
import { MetricCard } from '@/components/shared/metric-card'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { UserDialog } from '@/features/users/components/user-dialog'
import { UsersTable } from '@/features/users/components/users-table'
import { useUsers } from '@/features/users/hooks'

export function UsersPage() {
  const users = useUsers()
  const [createOpen, setCreateOpen] = useState(false)
  if (users.isPending) return <PageSkeleton rows={6} />
  if (users.isError) return <ErrorState error={users.error} onRetry={() => users.refetch()} />
  const items = users.data.items
  return <div className="page-grid"><PageHeader title="账号与权限" description="创建平台账号，并在管理员与查看者两种权限之间分配访问能力。" actions={<Button onClick={() => setCreateOpen(true)}><Plus />创建账号</Button>} /><section className="grid gap-4 sm:grid-cols-3"><MetricCard icon={Users} title="账号总数" value={items.length} description="所有平台登录账号" /><MetricCard icon={ShieldCheck} title="管理员" value={items.filter((item) => item.role === 'admin' && item.enabled).length} description="可以管理系统与发起扫描" /><MetricCard icon={UserCheck} title="查看者" value={items.filter((item) => item.role === 'viewer' && item.enabled).length} description="只能查看当前数据" /></section><UsersTable users={items} /><UserDialog open={createOpen} onOpenChange={setCreateOpen} /></div>
}
