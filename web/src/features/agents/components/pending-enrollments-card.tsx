import { Link } from 'react-router-dom'
import { Clock3, Laptop, ShieldCheck } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { LiveRelativeTime } from '@/components/shared/live-relative-time'
import type { AgentEnrollment } from '@/features/agents/types'

export function PendingEnrollmentsCard({ enrollments }: { enrollments: AgentEnrollment[] }) {
  if (!enrollments.length) return null
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>连接请求</CardTitle>
          <CardDescription>审批新请求，并跟踪已经批准但尚未完成连接的 Agent。</CardDescription>
        </div>
        <span className="flex size-7 items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-foreground">{enrollments.length}</span>
      </CardHeader>
      <CardContent className="divide-y p-0">
        {enrollments.map((item) => (
          <div key={item.id} className="flex flex-col gap-4 px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0 space-y-1.5">
              <div className="flex flex-wrap items-center gap-2"><p className="truncate font-medium">{item.requested_name || item.name || '未命名 Agent'}</p><Badge variant={item.status === 'approved' ? 'secondary' : 'default'}>{item.status === 'approved' ? '已批准，等待连接' : '待审批'}</Badge></div>
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                <span className="inline-flex items-center gap-1"><Laptop />{[item.os, item.architecture].filter(Boolean).join(' · ') || '设备信息待上报'}</span>
                {item.version ? <span>Agent {item.version}</span> : null}
                <span className="inline-flex items-center gap-1"><Clock3 />到期时间：<LiveRelativeTime value={item.expires_at} /></span>
              </div>
            </div>
            <Button nativeButton={false} render={<Link to={`/agents/enrollments/${item.id}`} />}><ShieldCheck />{item.status === 'approved' ? '查看状态' : '查看并批准'}</Button>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
