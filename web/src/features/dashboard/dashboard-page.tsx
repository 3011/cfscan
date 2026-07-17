import { Activity, Ban, Bot, Network, PlayCircle, Server } from 'lucide-react'
import { PageHeader } from '@/components/shared/page-header'
import { MetricCard } from '@/components/shared/metric-card'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { ErrorState } from '@/components/shared/error-state'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useOverview } from '@/features/dashboard/hooks'
import { useScanJobs } from '@/features/scans/hooks'
import { useScanResults } from '@/features/results/hooks'
import { useAgents } from '@/features/agents/hooks'
import { ColoPerformanceChart } from '@/features/dashboard/components/colo-performance-chart'
import { RecentJobsCard } from '@/features/dashboard/components/recent-jobs-card'
import { TopResultsCard } from '@/features/dashboard/components/top-results-card'
import { formatMS, formatNumber } from '@/lib/format'

export function DashboardPage() {
  const overview = useOverview()
  const jobs = useScanJobs(8, 5_000)
  const results = useScanResults({ view: 'latest', page: 1, page_size: 200, sort: 'latency_ms', order: 'asc', available: true, time_range: '24h' }, 10_000)
  const agents = useAgents({ refetchInterval: 5_000 })

  if (overview.isPending) return <PageSkeleton />
  if (overview.isError) return <ErrorState error={overview.error} onRetry={() => overview.refetch()} />

  const data = overview.data
  const offlineAgents = agents.data?.items.filter((agent) => agent.status === 'offline') ?? []
  const attentionCount = offlineAgents.length + (data.ips_blacklisted > 0 ? 1 : 0)

  return (
    <div className="page-grid">
      <PageHeader title="运行总览" description="从系统健康、扫描执行到各地区优选结果，集中查看当前 Cloudflare IP 池的运行状态。" />

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard icon={Bot} title="在线 Agent" value={`${data.agents_online} / ${data.agents_total}`} description="最近 45 秒内有心跳" />
        <MetricCard icon={Network} title="活跃前缀" value={formatNumber(data.prefixes_total)} description="官方地址段与启用 ASN 去重后" />
        <MetricCard icon={PlayCircle} title="运行中任务" value={formatNumber(data.running_jobs)} description={`累计完成 ${formatNumber(data.completed_jobs)} 个任务`} />
        <MetricCard icon={Activity} title="平均可用延迟" value={formatMS(data.average_latency_ms)} description={`近 24 小时 ${formatNumber(data.results_last_24h)} 条结果`} />
      </section>

      <section className="grid gap-4 xl:grid-cols-3">
        {results.isError ? (
          <Card className="xl:col-span-2"><CardContent className="pt-6"><ErrorState title="colo 数据加载失败" error={results.error} onRetry={() => results.refetch()} /></CardContent></Card>
        ) : <ColoPerformanceChart results={results.data?.items ?? []} />}
        {jobs.isError ? (
          <Card><CardContent className="pt-6"><ErrorState title="任务数据加载失败" error={jobs.error} onRetry={() => jobs.refetch()} /></CardContent></Card>
        ) : <RecentJobsCard jobs={jobs.data?.items ?? []} />}
      </section>

      <section className="grid gap-4 xl:grid-cols-3">
        <TopResultsCard results={results.data?.items ?? []} />
        <Card>
          <CardHeader>
            <CardTitle>需要关注</CardTitle>
            <CardDescription>影响当前扫描覆盖和结果质量的事项。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between rounded-2xl bg-muted/35 p-3">
              <div className="flex items-center gap-3">
                <Server className="size-4 text-muted-foreground" />
                <div><p className="text-sm font-medium">离线 Agent</p><p className="text-xs text-muted-foreground">无法领取新的扫描任务</p></div>
              </div>
              <Badge variant={offlineAgents.length ? 'destructive' : 'secondary'}>{offlineAgents.length}</Badge>
            </div>
            <div className="flex items-center justify-between rounded-2xl bg-muted/35 p-3">
              <div className="flex items-center gap-3">
                <Ban className="size-4 text-muted-foreground" />
                <div><p className="text-sm font-medium">黑名单 IP</p><p className="text-xs text-muted-foreground">等待释放或定期复检</p></div>
              </div>
              <Badge variant={data.ips_blacklisted ? 'outline' : 'secondary'}>{formatNumber(data.ips_blacklisted)}</Badge>
            </div>
            <div className="flex items-center justify-between rounded-2xl bg-muted/35 p-3">
              <div className="flex items-center gap-3">
                <Activity className="size-4 text-muted-foreground" />
                <div><p className="text-sm font-medium">可用 IP</p><p className="text-xs text-muted-foreground">最近结果中通过探测</p></div>
              </div>
              <Badge variant="secondary">{formatNumber(data.ips_available)}</Badge>
            </div>
            {!attentionCount ? <p className="pt-1 text-xs text-muted-foreground">当前没有需要立即处理的系统异常。</p> : null}
          </CardContent>
        </Card>
      </section>
    </div>
  )
}
