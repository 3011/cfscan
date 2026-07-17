import { Link } from 'react-router-dom'
import { ArrowRight } from 'lucide-react'
import type { ScanJob } from '@/features/scans/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { StatusBadge } from '@/components/shared/status-badge'
import { EmptyState } from '@/components/shared/empty-state'

export function RecentJobsCard({ jobs }: { jobs: ScanJob[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>最近任务</CardTitle>
          <CardDescription>当前队列与最近完成的扫描。</CardDescription>
        </div>
        <Button nativeButton={false} variant="ghost" size="sm" render={<Link to="/jobs" />}>全部任务<ArrowRight /></Button>
      </CardHeader>
      <CardContent className="space-y-5">
        {jobs.length ? jobs.slice(0, 5).map((job) => (
          <div key={job.id} className="space-y-2">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{job.name}</p>
                <p className="mt-0.5 text-xs text-muted-foreground">{job.completed_targets} / {job.total_targets} · 成功 {job.success_targets}</p>
              </div>
              <StatusBadge status={job.status} />
            </div>
            <Progress value={job.progress} className="h-1.5" />
          </div>
        )) : <EmptyState className="min-h-48" title="还没有任务" description="创建第一轮扫描后，任务进度会显示在这里。" />}
      </CardContent>
    </Card>
  )
}
