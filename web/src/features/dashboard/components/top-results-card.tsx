import { Link } from 'react-router-dom'
import { ArrowRight, Copy } from 'lucide-react'
import { toast } from 'sonner'
import type { ScanResult } from '@/features/results/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyState } from '@/components/shared/empty-state'
import { formatMS } from '@/lib/format'
import { ColoLocationLabel } from '@/features/results/components/colo-location-label'

export function TopResultsCard({ results }: { results: ScanResult[] }) {
  const best = results.filter((item) => item.available).slice(0, 6)
  return (
    <Card className="xl:col-span-2">
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>当前优选 IP</CardTitle>
          <CardDescription>按 Agent 实测 TTFB 排序的可用结果。</CardDescription>
        </div>
        <Button nativeButton={false} variant="ghost" size="sm" render={<Link to="/results" />}>查看排行<ArrowRight /></Button>
      </CardHeader>
      <CardContent>
        {best.length ? (
          <div className="divide-y divide-border/60 rounded-2xl bg-muted/25">
            {best.map((item) => (
              <div key={item.id} className="flex items-center gap-3 px-3 py-3 text-sm">
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-7 shrink-0 text-muted-foreground"
                  aria-label={`复制 ${item.target_ip}`}
                  onClick={async () => {
                    await navigator.clipboard.writeText(item.target_ip)
                    toast.success('IP 已复制')
                  }}
                >
                  <Copy />
                </Button>
                <div className="min-w-0 flex-1">
                  <p className="truncate font-mono font-medium">{item.target_ip}</p>
                  <p className="truncate text-xs text-muted-foreground">{item.agent_name} · {item.region}</p>
                  <ColoLocationLabel className="mt-1 block max-w-full text-xs" code={item.colo} city={item.colo_city} country={item.colo_country} continent={item.colo_continent} />
                </div>
                <span className="metric-value min-w-20 text-right font-medium">{formatMS(item.latency_ms)}</span>
              </div>
            ))}
          </div>
        ) : <EmptyState title="暂无可用结果" description="当前还没有通过可用性与阈值筛选的 IP。" />}
      </CardContent>
    </Card>
  )
}
