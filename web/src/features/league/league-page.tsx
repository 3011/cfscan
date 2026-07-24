import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Activity, Binoculars, ChevronLeft, ChevronRight, Network, ShieldCheck, Swords, Trophy } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ErrorState } from '@/components/shared/error-state'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { useAgents } from '@/features/agents/hooks'
import { useLeagueDashboard } from '@/features/league/hooks'
import type { LeagueCandidate, PrefixTier } from '@/features/league/types'
import { formatDate, formatMS, formatNumber, formatPercent } from '@/lib/format'

const tierLabels: Record<PrefixTier, string> = {
  champion: '冠军层',
  challenger: '挑战层',
  observation: '观察层',
}

function TierBadge({ tier }: { tier: PrefixTier }) {
  const variant = tier === 'champion' ? 'default' : tier === 'challenger' ? 'secondary' : 'outline'
  return <Badge variant={variant}>{tierLabels[tier]}</Badge>
}

function trendHref(item: LeagueCandidate) {
  const params = new URLSearchParams({
    prefix: item.prefix_cidr,
    tier: item.tier,
    scheme: item.scheme,
    hostname: item.hostname,
    path: item.path,
    port: String(item.port),
    attempts: String(item.attempts),
    timeout_ms: String(item.timeout_ms),
  })
  return `/league/ip/${encodeURIComponent(item.agent_id)}/${encodeURIComponent(item.target_ip)}?${params}`
}

function SummaryCard({ title, value, description, icon: Icon }: { title: string; value: number; description: string; icon: typeof Trophy }) {
  return (
    <Card size="sm">
      <CardContent className="flex items-start justify-between gap-4 p-5">
        <div><p className="text-sm text-muted-foreground">{title}</p><p className="metric-value mt-2 text-2xl">{formatNumber(value)}</p><p className="mt-2 text-xs leading-5 text-muted-foreground">{description}</p></div>
        <span className="rounded-2xl bg-muted p-2.5"><Icon className="size-4" /></span>
      </CardContent>
    </Card>
  )
}

function TablePagination({
  page,
  pageSize,
  total,
  totalPages,
  onPageChange,
  onPageSizeChange,
}: {
  page: number
  pageSize: number
  total: number
  totalPages: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
}) {
  if (total === 0) return null
  const first = (page - 1) * pageSize + 1
  const last = Math.min(page * pageSize, total)
  return (
    <div className="flex flex-col gap-3 px-1 pt-3 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-sm text-muted-foreground">共 {formatNumber(total)} 条，当前 {formatNumber(first)}–{formatNumber(last)} 条</p>
      <div className="flex items-center gap-2">
        <Select
          items={{ '50': '50 条', '100': '100 条', '200': '200 条' }}
          value={String(pageSize)}
          onValueChange={(value) => onPageSizeChange(Number(value))}
        >
          <SelectTrigger className="h-8 w-[92px]" aria-label="每页条数"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="50">50 条</SelectItem>
            <SelectItem value="100">100 条</SelectItem>
            <SelectItem value="200">200 条</SelectItem>
          </SelectContent>
        </Select>
        <span className="min-w-20 text-center text-sm text-muted-foreground">{page} / {Math.max(totalPages, 1)}</span>
        <Button variant="outline" size="icon" className="size-8" onClick={() => onPageChange(page - 1)} disabled={page <= 1} aria-label="上一页"><ChevronLeft /></Button>
        <Button variant="outline" size="icon" className="size-8" onClick={() => onPageChange(page + 1)} disabled={page >= totalPages} aria-label="下一页"><ChevronRight /></Button>
      </div>
    </div>
  )
}

export function LeaguePage() {
  const [agentId, setAgentId] = useState('')
  const [candidatePage, setCandidatePage] = useState(1)
  const [candidatePageSize, setCandidatePageSize] = useState(50)
  const [prefixPage, setPrefixPage] = useState(1)
  const [prefixPageSize, setPrefixPageSize] = useState(50)
  const agents = useAgents()
  const league = useLeagueDashboard({
    agent_id: agentId || undefined,
    candidate_page: candidatePage,
    candidate_page_size: candidatePageSize,
    prefix_page: prefixPage,
    prefix_page_size: prefixPageSize,
  })

  function changeAgent(value: string) {
    setAgentId(value === 'all' ? '' : value)
    setCandidatePage(1)
    setPrefixPage(1)
  }

  if (agents.isError) return <ErrorState title="Agent 列表加载失败" error={agents.error} onRetry={() => agents.refetch()} />
  if (league.isError) return <ErrorState title="最佳 IP 联赛加载失败" error={league.error} onRetry={() => league.refetch()} />
  if (!agents.data || !league.data) return <PageSkeleton rows={10} />

  const { summary } = league.data
  const candidates = league.data.candidates.items
  const prefixes = league.data.prefixes.items
  return (
    <div className="page-grid">
      <PageHeader
        title="最佳 IP 联赛"
        description="按 Agent 独立评估前缀质量：冠军层高频深挖，挑战层继续验证，观察层低频保留晋级机会；候选 IP 会固定复测。"
        actions={(
          <Select
            items={Object.fromEntries([['all', '全部 Agent'], ...agents.data.items.map((agent) => [agent.id, agent.name])])}
            value={agentId || 'all'}
            onValueChange={changeAgent}
          >
            <SelectTrigger className="w-full sm:w-52" aria-label="联赛 Agent"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部 Agent</SelectItem>
              {agents.data.items.map((agent) => <SelectItem key={agent.id} value={agent.id}>{agent.name}</SelectItem>)}
            </SelectContent>
          </Select>
        )}
      />

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <SummaryCard title="冠军前缀" value={summary.champion_prefixes} description="最短每小时进入一轮新 IP 探索与候选复测。" icon={Trophy} />
        <SummaryCard title="挑战前缀" value={summary.challenger_prefixes} description="最短每 6 小时继续积累跨 IP 证据。" icon={Swords} />
        <SummaryCard title="观察前缀" value={summary.observation_prefixes} description="最短每 24 小时保留一次晋级机会。" icon={Binoculars} />
        <SummaryCard title="候选 IP" value={summary.candidate_ips} description="至少具有重复样本，按稳定性与 P95 排序。" icon={ShieldCheck} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><ShieldCheck className="size-4" />候选最佳 IP</CardTitle>
          <CardDescription>每个冠军或挑战前缀最多展示两个候选；排序优先考虑可用率，再比较 P95 延迟和平均丢包。</CardDescription>
        </CardHeader>
        <CardContent>
          {candidates.length === 0 ? (
            <div className="rounded-2xl border border-dashed p-8 text-center text-sm text-muted-foreground">暂无候选 IP。创建“智能筛选最佳 IP”扫描计划后，系统会先从观察层积累多 IP 样本。</div>
          ) : (
            <Table>
              <TableHeader><TableRow><TableHead>IP / colo</TableHead><TableHead>Agent</TableHead><TableHead>前缀</TableHead><TableHead>样本</TableHead><TableHead>可用率</TableHead><TableHead>P95 延迟</TableHead><TableHead>平均丢包</TableHead><TableHead>最近扫描</TableHead><TableHead><span className="sr-only">操作</span></TableHead></TableRow></TableHeader>
              <TableBody>
                {candidates.map((item) => (
                  <TableRow key={`${item.agent_id}-${item.target_ip}-${item.scheme}-${item.hostname}-${item.path}-${item.port}-${item.attempts}-${item.timeout_ms}`}>
                    <TableCell><p className="font-mono font-medium">{item.target_ip}</p><p className="mt-1 text-xs text-muted-foreground">{item.colo || '未识别 colo'}</p></TableCell>
                    <TableCell><p>{item.agent_name}</p><p className="mt-1 text-xs text-muted-foreground">{item.continent} / {item.region}</p></TableCell>
                    <TableCell><div className="flex items-center gap-2"><span className="font-mono text-xs">{item.prefix_cidr}</span><TierBadge tier={item.tier} /></div></TableCell>
                    <TableCell>{formatNumber(item.sample_count)}</TableCell>
                    <TableCell>{formatPercent(item.availability_rate, 1)}</TableCell>
                    <TableCell>{formatMS(item.latency_p95_ms)}</TableCell>
                    <TableCell>{formatPercent(item.packet_loss_avg, 1)}</TableCell>
                    <TableCell className="text-muted-foreground">{formatDate(item.last_scanned_at)}</TableCell>
                    <TableCell><Button nativeButton={false} size="sm" variant="ghost" render={<Link to={trendHref(item)} />}><Activity />查看趋势</Button></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <TablePagination
            page={league.data.candidates.page}
            pageSize={league.data.candidates.page_size}
            total={league.data.candidates.total}
            totalPages={league.data.candidates.total_pages}
            onPageChange={setCandidatePage}
            onPageSizeChange={(value) => { setCandidatePageSize(value); setCandidatePage(1) }}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2"><Network className="size-4" />前缀升降级</CardTitle>
          <CardDescription>升级依赖 7 天内的跨 IP 样本；降级使用最近 24 小时的明显恶化，并通过连续坏窗口避免一次抖动误判。</CardDescription>
        </CardHeader>
        <CardContent>
          {prefixes.length === 0 ? (
            <div className="rounded-2xl border border-dashed p-8 text-center text-sm text-muted-foreground">尚未初始化前缀联赛。请创建或运行一次“智能筛选最佳 IP”任务。</div>
          ) : (
            <Table>
              <TableHeader><TableRow><TableHead>前缀</TableHead><TableHead>Agent</TableHead><TableHead>层级</TableHead><TableHead>样本 / IP</TableHead><TableHead>7 天可用率</TableHead><TableHead>7 天 P95</TableHead><TableHead>平均丢包</TableHead><TableHead>最近结果</TableHead><TableHead>最近调度</TableHead></TableRow></TableHeader>
              <TableBody>
                {prefixes.map((item) => (
                  <TableRow key={`${item.agent_id}-${item.prefix_cidr}-${item.scheme}-${item.hostname}-${item.path}-${item.port}-${item.attempts}-${item.timeout_ms}`}>
                    <TableCell><p className="font-mono font-medium">{item.prefix_cidr}</p><p className="mt-1 max-w-72 truncate font-mono text-xs text-muted-foreground">{item.scheme}://{item.hostname}:{item.port}{item.path}</p></TableCell>
                    <TableCell><p>{item.agent_name}</p><p className="mt-1 text-xs text-muted-foreground">{item.continent} / {item.region}</p></TableCell>
                    <TableCell><TierBadge tier={item.tier} />{item.bad_streak > 0 ? <p className="mt-1 text-xs text-destructive">坏窗口 {item.bad_streak}/2</p> : null}</TableCell>
                    <TableCell>{formatNumber(item.sample_count)} / {formatNumber(item.distinct_ip_count)}</TableCell>
                    <TableCell>{formatPercent(item.availability_rate, 1)}</TableCell>
                    <TableCell>{formatMS(item.latency_p95_ms)}</TableCell>
                    <TableCell>{formatPercent(item.packet_loss_avg, 1)}</TableCell>
                    <TableCell className="text-muted-foreground">{formatDate(item.last_result_at)}</TableCell>
                    <TableCell className="text-muted-foreground">{formatDate(item.last_scheduled_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <TablePagination
            page={league.data.prefixes.page}
            pageSize={league.data.prefixes.page_size}
            total={league.data.prefixes.total}
            totalPages={league.data.prefixes.total_pages}
            onPageChange={setPrefixPage}
            onPageSizeChange={(value) => { setPrefixPageSize(value); setPrefixPage(1) }}
          />
        </CardContent>
      </Card>
    </div>
  )
}
