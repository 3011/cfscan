import { useMemo } from 'react'
import { Activity, Gauge, MapPin, Sigma } from 'lucide-react'
import { Bar, BarChart, CartesianGrid, Cell, LabelList, ReferenceLine, XAxis, YAxis } from 'recharts'
import type { ScanResult } from '@/features/results/types'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ChartContainer, ChartTooltip, type ChartConfig } from '@/components/ui/chart'
import { EmptyState } from '@/components/shared/empty-state'
import { formatNumber } from '@/lib/format'

interface ColoMetric {
  colo: string
  city: string
  country: string
  continent: string
  shortLabel: string
  p50: number
  p95: number
  minimum: number
  average: number
  samples: number
}

interface TooltipPayloadItem {
  payload: ColoMetric
}

const chartConfig = {
  p50: { label: 'P50 TTFB', color: 'var(--chart-1)' },
} satisfies ChartConfig

function percentile(values: number[], ratio: number) {
  if (!values.length) return 0
  const index = Math.min(values.length - 1, Math.ceil(values.length * ratio) - 1)
  return values[index]
}

function ColoTooltip({ active, payload }: { active?: boolean; payload?: TooltipPayloadItem[] }) {
  const item = payload?.[0]?.payload
  if (!active || !item) return null
  return (
    <div className="min-w-48 rounded-lg border bg-popover p-3 text-popover-foreground shadow-lg">
      <div className="mb-2 flex items-center justify-between gap-4">
        <span className="font-medium">{item.colo} · {item.city || 'Location unknown'}{item.country ? `, ${item.country}` : ''}</span>
        <Badge variant="secondary">{item.samples} 个样本</Badge>
      </div>
      {item.continent ? <p className="mb-2 text-xs text-muted-foreground">{item.continent}</p> : null}
      <dl className="grid grid-cols-2 gap-x-5 gap-y-1.5 text-xs">
        <dt className="text-muted-foreground">P50 TTFB</dt><dd className="text-right font-mono font-medium">{item.p50} ms</dd>
        <dt className="text-muted-foreground">P95 TTFB</dt><dd className="text-right font-mono">{item.p95} ms</dd>
        <dt className="text-muted-foreground">平均值</dt><dd className="text-right font-mono">{item.average} ms</dd>
        <dt className="text-muted-foreground">最低值</dt><dd className="text-right font-mono">{item.minimum} ms</dd>
      </dl>
    </div>
  )
}

export function ColoPerformanceChart({ results }: { results: ScanResult[] }) {
  const data = useMemo(() => {
    const groups = new Map<string, { values: number[]; city: string; country: string; continent: string }>()
    results.filter((item) => item.available && item.colo && item.latency_ms > 0).forEach((item) => {
      const group = groups.get(item.colo) ?? {
        values: [], city: item.colo_city, country: item.colo_country, continent: item.colo_continent,
      }
      group.values.push(item.latency_ms)
      if (!group.city && item.colo_city) group.city = item.colo_city
      if (!group.country && item.colo_country) group.country = item.colo_country
      if (!group.continent && item.colo_continent) group.continent = item.colo_continent
      groups.set(item.colo, group)
    })
    return [...groups.entries()]
      .map(([colo, group]) => {
        const values = [...group.values].sort((a, b) => a - b)
        return {
          colo,
          city: group.city,
          country: group.country,
          continent: group.continent,
          shortLabel: `${colo} · ${group.city || 'Unknown'}`,
          p50: Math.round(percentile(values, 0.5)),
          p95: Math.round(percentile(values, 0.95)),
          minimum: Math.round(values[0]),
          average: Math.round(values.reduce((sum, value) => sum + value, 0) / values.length),
          samples: values.length,
        } satisfies ColoMetric
      })
      .sort((a, b) => a.p50 - b.p50)
      .slice(0, 10)
  }, [results])

  const totalSamples = data.reduce((sum, item) => sum + item.samples, 0)
  const overallMedian = data.length ? Math.round(percentile([...data.map((item) => item.p50)].sort((a, b) => a - b), 0.5)) : 0
  const chartHeight = Math.max(280, Math.min(430, data.length * 44 + 52))

  return (
    <Card className="xl:col-span-2">
      <CardHeader className="gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="space-y-1.5">
          <CardTitle>colo 延迟表现</CardTitle>
          <CardDescription>按 P50 TTFB 排名，横线表示当前 colo 中位水平；悬停可查看 P95 和样本数。</CardDescription>
        </div>
        {data.length ? (
          <div className="flex flex-wrap gap-2 text-xs">
            <Badge variant="outline" className="max-w-full gap-1.5 font-normal"><MapPin className="size-3.5 shrink-0" /><span className="truncate">最快 {data[0].colo} · {data[0].city || 'Location unknown'}{data[0].country ? `, ${data[0].country}` : ''}</span></Badge>
            <Badge variant="outline" className="gap-1.5 font-normal"><Gauge className="size-3.5" />P50 {data[0].p50} ms</Badge>
            <Badge variant="outline" className="gap-1.5 font-normal"><Sigma className="size-3.5" />{formatNumber(totalSamples)} 样本</Badge>
          </div>
        ) : null}
      </CardHeader>
      <CardContent>
        {data.length ? (
          <ChartContainer config={chartConfig} className="aspect-auto w-full" style={{ height: chartHeight }}>
            <BarChart accessibilityLayer data={data} layout="vertical" margin={{ left: 0, right: 44, top: 6, bottom: 6 }}>
              <CartesianGrid horizontal={false} strokeDasharray="3 5" />
              <XAxis type="number" tickLine={false} axisLine={false} tickMargin={8} tickFormatter={(value) => `${value}ms`} />
              <YAxis type="category" dataKey="shortLabel" tickLine={false} axisLine={false} width={132} tickMargin={8} />
              <ReferenceLine x={overallMedian} stroke="var(--muted-foreground)" strokeDasharray="4 4" strokeOpacity={0.45} />
              <ChartTooltip cursor={{ fill: 'color-mix(in oklch, var(--muted) 45%, transparent)' }} content={<ColoTooltip />} />
              <Bar dataKey="p50" radius={[0, 6, 6, 0]} maxBarSize={26}>
                {data.map((item, index) => (
                  <Cell key={item.colo} fill={index === 0 ? 'var(--chart-2)' : 'var(--chart-1)'} fillOpacity={Math.max(0.5, 1 - index * 0.055)} />
                ))}
                <LabelList dataKey="p50" position="right" formatter={(value) => `${Number(value ?? 0)}ms`} className="fill-foreground text-[11px] font-medium" />
              </Bar>
            </BarChart>
          </ChartContainer>
        ) : (
          <EmptyState icon={Activity} title="等待 colo 数据" description="运行扫描任务后，这里会按实际命中的 Cloudflare colo 展示延迟表现。" />
        )}
      </CardContent>
    </Card>
  )
}
