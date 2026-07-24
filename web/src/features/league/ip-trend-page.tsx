import { useMemo, useState } from 'react'
import { Activity, ArrowLeft, Gauge, ShieldCheck, Sigma } from 'lucide-react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from 'recharts'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from '@/components/ui/chart'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ErrorState } from '@/components/shared/error-state'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { useIPTrend } from '@/features/league/hooks'
import type { LeagueCandidate, TrendTimeRange } from '@/features/league/types'
import { formatMS, formatNumber, formatPercent } from '@/lib/format'

const latencyConfig = {
  latency_ms: { label: '总延迟', color: 'var(--chart-1)' },
  ttfb_ms: { label: 'TTFB', color: 'var(--chart-2)' },
} satisfies ChartConfig

const lossConfig = {
  packet_loss: { label: '丢包率', color: 'var(--chart-3)' },
} satisfies ChartConfig

function numberParam(params: URLSearchParams, key: string, fallback: number) {
  const value = Number(params.get(key))
  return Number.isFinite(value) && value > 0 ? value : fallback
}

function timeLabel(value: string) {
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

function MetricCard({ label, value, description }: { label: string; value: string; description: string }) {
  return <Card size="sm"><CardContent className="p-5"><p className="text-sm text-muted-foreground">{label}</p><p className="metric-value mt-2 text-2xl">{value}</p><p className="mt-2 text-xs text-muted-foreground">{description}</p></CardContent></Card>
}

export function IPTrendPage() {
  const { agentID = '', targetIP = '' } = useParams()
  const [params] = useSearchParams()
  const [timeRange, setTimeRange] = useState<TrendTimeRange>('7d')
  const candidate = useMemo<LeagueCandidate>(() => ({
    agent_id: agentID,
    agent_name: '',
    region: '',
    continent: '',
    prefix_cidr: params.get('prefix') ?? '',
    tier: (params.get('tier') as LeagueCandidate['tier']) || 'observation',
    scheme: params.get('scheme') ?? 'https',
    hostname: params.get('hostname') ?? 'cloudflare.com',
    path: params.get('path') ?? '/cdn-cgi/trace',
    port: numberParam(params, 'port', 443),
    attempts: numberParam(params, 'attempts', 3),
    timeout_ms: numberParam(params, 'timeout_ms', 5000),
    target_ip: targetIP,
    colo: '',
    sample_count: 0,
    availability_rate: 0,
    latency_p95_ms: 0,
    packet_loss_avg: 0,
    last_scanned_at: '',
  }), [agentID, params, targetIP])
  const trend = useIPTrend(candidate, timeRange)

  if (trend.isError) return <ErrorState title="IP 趋势加载失败" error={trend.error} onRetry={() => trend.refetch()} />
  if (!trend.data) return <PageSkeleton rows={8} />

  const data = trend.data.points.map((point) => ({
    ...point,
    label: timeLabel(point.scanned_at),
    latency_ms: point.available ? point.latency_ms : null,
    ttfb_ms: point.available ? point.ttfb_ms : null,
  }))
  const summary = trend.data.summary
  return (
    <div className="page-grid">
      <PageHeader
        eyebrow="最佳 IP 联赛"
        title={trend.data.target_ip}
        description={`${trend.data.agent_name} · ${candidate.scheme}://${candidate.hostname}:${candidate.port}${candidate.path} · 仅比较相同探测配置。`}
        actions={(
          <div className="flex flex-wrap gap-2">
            <Button nativeButton={false} variant="outline" render={<Link to="/league" />}><ArrowLeft />返回联赛</Button>
            <Select items={{ '24h': '最近 24 小时', '7d': '最近 7 天', '30d': '最近 30 天' }} value={timeRange} onValueChange={(value) => setTimeRange(value as TrendTimeRange)}>
              <SelectTrigger className="w-40" aria-label="趋势时间范围"><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="24h">最近 24 小时</SelectItem><SelectItem value="7d">最近 7 天</SelectItem><SelectItem value="30d">最近 30 天</SelectItem></SelectContent>
            </Select>
          </div>
        )}
      />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
        <MetricCard label="样本数" value={formatNumber(summary.sample_count)} description="当前时间范围内的原始扫描次数" />
        <MetricCard label="可用率" value={formatPercent(summary.availability_rate, 1)} description="成功探测占全部样本的比例" />
        <MetricCard label="P50 延迟" value={formatMS(summary.latency_p50_ms)} description="典型网络表现" />
        <MetricCard label="P95 延迟" value={formatMS(summary.latency_p95_ms)} description="用于观察尾部抖动" />
        <MetricCard label="平均丢包" value={formatPercent(summary.packet_loss_avg, 1)} description={`最近 colo：${summary.latest_colo || '未识别'}`} />
      </div>

      {data.length === 0 ? (
        <Card><CardContent className="p-10 text-center text-sm text-muted-foreground">当前范围没有该 IP 的历史样本。</CardContent></Card>
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          <Card>
            <CardHeader><CardTitle className="flex items-center gap-2"><Gauge className="size-4" />延迟趋势</CardTitle><CardDescription>总延迟与 TTFB 分开显示；失败样本不伪造为 0 ms。</CardDescription></CardHeader>
            <CardContent>
              <ChartContainer config={latencyConfig} className="h-80 w-full">
                <LineChart accessibilityLayer data={data} margin={{ left: 4, right: 12, top: 8, bottom: 4 }}>
                  <CartesianGrid vertical={false} strokeDasharray="3 5" />
                  <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={42} />
                  <YAxis tickLine={false} axisLine={false} tickFormatter={(value) => `${value}ms`} width={54} />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Line type="monotone" dataKey="latency_ms" stroke="var(--color-latency_ms)" strokeWidth={2} dot={false} connectNulls={false} />
                  <Line type="monotone" dataKey="ttfb_ms" stroke="var(--color-ttfb_ms)" strokeWidth={2} dot={false} connectNulls={false} />
                </LineChart>
              </ChartContainer>
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle className="flex items-center gap-2"><Activity className="size-4" />丢包率趋势</CardTitle><CardDescription>独立坐标轴显示 0%～100%，避免和毫秒指标混淆。</CardDescription></CardHeader>
            <CardContent>
              <ChartContainer config={lossConfig} className="h-80 w-full">
                <LineChart accessibilityLayer data={data} margin={{ left: 4, right: 12, top: 8, bottom: 4 }}>
                  <CartesianGrid vertical={false} strokeDasharray="3 5" />
                  <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={42} />
                  <YAxis domain={[0, 100]} tickLine={false} axisLine={false} tickFormatter={(value) => `${value}%`} width={48} />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Line type="monotone" dataKey="packet_loss" stroke="var(--color-packet_loss)" strokeWidth={2} dot={false} />
                </LineChart>
              </ChartContainer>
            </CardContent>
          </Card>
        </div>
      )}

      <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
        <Badge variant="outline" className="gap-1.5"><ShieldCheck className="size-3.5" />前缀 {candidate.prefix_cidr || '未知'}</Badge>
        <Badge variant="outline" className="gap-1.5"><Sigma className="size-3.5" />尝试 {candidate.attempts} 次 · 超时 {candidate.timeout_ms} ms</Badge>
      </div>
    </div>
  )
}
