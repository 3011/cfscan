import { Activity, CalendarClock, DatabaseZap, ShieldCheck } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { ErrorState } from '@/components/shared/error-state'
import { useOverview } from '@/features/dashboard/hooks'
import { useAutomationRuns, useBlacklistRecheckSettings, useScanSchedules, useSourceSyncSchedules } from '@/features/settings/hooks'
import { formatDateInTimezone, formatNumber } from '@/lib/format'

type Upcoming = { key: string; name: string; type: string; next: string; timezone: string; detail: string; enabled: boolean }

export function AutomationOverview() {
  const schedules = useScanSchedules()
  const blacklist = useBlacklistRecheckSettings()
  const sources = useSourceSyncSchedules()
  const runs = useAutomationRuns()
  const overview = useOverview()
  const queries = [schedules, blacklist, sources, runs, overview]
  if (queries.some((query) => query.isPending)) return <PageSkeleton rows={6} />
  const failed = queries.find((query) => query.isError)
  if (failed?.error) return <ErrorState error={failed.error} onRetry={() => queries.forEach((query) => query.refetch())} />

  const upcoming: Upcoming[] = [
    ...(schedules.data?.items ?? []).map((item) => ({
      key: item.id, name: item.name, type: '扫描计划', next: item.next_run_at, timezone: item.timezone,
      detail: item.sampling_mode === 'one_per_prefix' ? '每个启用前缀取 1 个 IP' : item.sampling_mode === 'league' ? `联赛预算 ${item.target_count} 个 IP / Agent` : `${item.target_count} 个 IP`, enabled: item.enabled,
    })),
    blacklist.data ? {
      key: 'blacklist', name: '黑名单复查', type: '恢复策略', next: blacklist.data.next_run_at, timezone: blacklist.data.timezone,
      detail: `当前 ${blacklist.data.eligible_targets} 个候选，取 ${(blacklist.data.fraction * 100).toFixed(0)}%，最多 ${blacklist.data.max_targets}`, enabled: blacklist.data.enabled,
    } : null,
    ...(sources.data?.items ?? []).map((item) => ({
      key: item.source, name: item.name, type: '数据同步', next: item.next_run_at, timezone: item.timezone,
      detail: item.run_on_startup ? '定时执行，并在中心启动时同步' : '仅按计划执行', enabled: item.enabled,
    })),
  ].filter((item): item is Upcoming => Boolean(item)).filter((item) => item.enabled).sort((a, b) => new Date(a.next).getTime() - new Date(b.next).getTime())

  const enabledCount = (schedules.data?.items.filter((item) => item.enabled).length ?? 0)
    + (blacklist.data?.enabled ? 1 : 0)
    + (sources.data?.items.filter((item) => item.enabled).length ?? 0)
  const recentFailures = runs.data?.items.filter((item) => item.status === 'failed').slice(0, 20).length ?? 0
  const estimatedRecheck = blacklist.data ? Math.min(Math.ceil(blacklist.data.eligible_targets * blacklist.data.fraction), blacklist.data.max_targets) : 0

  return (
    <div className="space-y-5">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card><CardHeader className="pb-2"><CardDescription>已启用自动化</CardDescription><CardTitle className="flex items-center gap-2 text-2xl"><Activity className="size-5 text-muted-foreground" />{enabledCount}</CardTitle></CardHeader><CardContent className="text-xs text-muted-foreground">所有非手动行为均在此处可见</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardDescription>下一次执行</CardDescription><CardTitle className="truncate text-base">{upcoming[0]?.name ?? '暂无计划'}</CardTitle></CardHeader><CardContent className="text-xs text-muted-foreground">{upcoming[0] ? formatDateInTimezone(upcoming[0].next, upcoming[0].timezone) : '启用自动化后显示'}</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardDescription>预计黑名单复查</CardDescription><CardTitle className="flex items-center gap-2 text-2xl"><ShieldCheck className="size-5 text-muted-foreground" />{formatNumber(estimatedRecheck)}</CardTitle></CardHeader><CardContent className="text-xs text-muted-foreground">按当前候选、比例与单轮上限估算</CardContent></Card>
        <Card><CardHeader className="pb-2"><CardDescription>最近异常执行</CardDescription><CardTitle className="flex items-center gap-2 text-2xl"><DatabaseZap className="size-5 text-muted-foreground" />{recentFailures}</CardTitle></CardHeader><CardContent className="text-xs text-muted-foreground">最近 20 条执行记录中的失败数</CardContent></Card>
      </div>

      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><CalendarClock className="size-4" />即将执行</CardTitle><CardDescription>按下次执行时间排序，便于提前确认中心会产生哪些网络行为。</CardDescription></CardHeader>
        <CardContent className="space-y-2">
          {upcoming.length ? upcoming.slice(0, 8).map((item) => (
            <div key={`${item.type}-${item.key}`} className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0"><div className="flex items-center gap-2"><p className="truncate font-medium">{item.name}</p><Badge variant="secondary">{item.type}</Badge></div><p className="mt-1 text-sm text-muted-foreground">{item.detail}</p></div>
              <div className="shrink-0 text-sm sm:text-right"><p>{formatDateInTimezone(item.next, item.timezone)}</p><p className="mt-1 text-xs text-muted-foreground">{item.timezone}</p></div>
            </div>
          )) : <p className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">当前没有已启用的自动化计划。</p>}
        </CardContent>
      </Card>
      <p className="text-xs text-muted-foreground">当前在线 Agent：{overview.data?.agents_online ?? 0}；启用前缀：{formatNumber(overview.data?.prefixes_total)}。实际任务数量还会受到 Agent 数量、IPv6 开关和黑名单过滤影响。</p>
    </div>
  )
}
