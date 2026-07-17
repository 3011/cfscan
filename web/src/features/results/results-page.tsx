import { useDeferredValue, useState } from 'react'
import type { PaginationState, SortingState } from '@tanstack/react-table'
import { History, ListFilter, Sparkles } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { ErrorState } from '@/components/shared/error-state'
import { useAgents } from '@/features/agents/hooks'
import { ResultsTable } from '@/features/results/components/results-table'
import { useResultFacets, useResultJobs, useScanResults } from '@/features/results/hooks'
import { normalizeGeoSelection } from '@/features/results/geo-facets'
import type { ResultSort, ResultTimeRange, ResultView } from '@/features/results/types'
import { formatNumber } from '@/lib/format'

const latestSorting: SortingState = [{ id: 'latency_ms', desc: false }]
const historySorting: SortingState = [{ id: 'scanned_at', desc: true }]

export function ResultsPage() {
  const [view, setView] = useState<ResultView>('latest')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(50)
  const [sorting, setSorting] = useState<SortingState>(latestSorting)
  const [search, setSearch] = useState('')
  const deferredSearch = useDeferredValue(search.trim())
  const [agentId, setAgentId] = useState('')
  const [jobId, setJobId] = useState('')
  const [colo, setColo] = useState('')
  const [coloContinent, setColoContinent] = useState('')
  const [coloCountry, setColoCountry] = useState('')
  const [coloCity, setColoCity] = useState('')
  const [available, setAvailable] = useState('true')
  const [timeRange, setTimeRange] = useState<ResultTimeRange>('24h')

  const sort = (sorting[0]?.id ?? (view === 'latest' ? 'latency_ms' : 'scanned_at')) as ResultSort
  const order = sorting[0]?.desc ? 'desc' : 'asc'
  const agents = useAgents()
  const facets = useResultFacets({
    view,
    agent_id: agentId,
    job_id: jobId,
    search: deferredSearch,
    available: available || undefined,
    time_range: timeRange,
  })
  const effectiveGeo = normalizeGeoSelection(facets.data?.items ?? [], {
    continent: coloContinent,
    country: coloCountry,
    city: coloCity,
    colo,
  })
  const jobs = useResultJobs({
    view,
    agent_id: agentId,
    available: available || undefined,
    time_range: timeRange,
  })
  const results = useScanResults({
    view,
    page,
    page_size: pageSize,
    sort,
    order,
    search: deferredSearch,
    agent_id: agentId,
    job_id: jobId,
    colo: effectiveGeo.colo,
    colo_continent: effectiveGeo.continent,
    colo_country: effectiveGeo.country,
    colo_city: effectiveGeo.city,
    available: available || undefined,
    time_range: timeRange,
  })

  function resetGeoFilters() {
    setColoContinent('')
    setColoCountry('')
    setColoCity('')
    setColo('')
  }

  function changeView(next: ResultView) {
    setView(next)
    setPage(1)
    setJobId('')
    setAvailable(next === 'latest' ? 'true' : '')
    resetGeoFilters()
    setSorting(next === 'latest' ? latestSorting : historySorting)
  }

  function resetAll() {
    setPage(1)
    setSearch('')
    setAgentId('')
    setJobId('')
    setAvailable('true')
    setTimeRange('24h')
    resetGeoFilters()
  }

  if (results.isError) return <ErrorState error={results.error} onRetry={() => results.refetch()} />
  if (agents.isError) return <ErrorState title="Agent 列表加载失败" error={agents.error} onRetry={() => agents.refetch()} />
  if (facets.isError) return <ErrorState title="结果地理筛选加载失败" error={facets.error} onRetry={() => facets.refetch()} />
  if (jobs.isError) return <ErrorState title="扫描任务筛选加载失败" error={jobs.error} onRetry={() => jobs.refetch()} />
  if (!results.data || !agents.data || !facets.data || !jobs.data) return <PageSkeleton rows={10} />

  return (
    <div className="page-grid">
      <PageHeader
        title="结果排行"
        description="默认查看每个 Agent、目标 IP 和探测配置的最新状态；历史记录保留每一次原始扫描结果。"
        actions={<Badge variant="outline" className="gap-1.5"><ListFilter className="size-3.5" />共 {formatNumber(results.data.total)} 条匹配结果</Badge>}
      />
      <Tabs value={view} onValueChange={(value) => changeView(value as ResultView)}>
        <TabsList variant="line">
          <TabsTrigger value="latest"><Sparkles />最新结果</TabsTrigger>
          <TabsTrigger value="history"><History />历史记录</TabsTrigger>
        </TabsList>
      </Tabs>
      <div className="rounded-2xl bg-muted/45 px-4 py-3 text-sm leading-6 text-muted-foreground">
        {view === 'latest'
          ? '最新结果按 Agent、目标 IP、协议、域名、路径、端口、尝试次数和超时配置去重；状态与 colo 筛选应用在最新一条结果上。'
          : '历史记录展示每次原始扫描，不做去重；默认按扫描时间从新到旧排列。'}
      </div>
      <ResultsTable
        results={results.data.items}
        total={results.data.total}
        page={results.data.page}
        pageSize={results.data.page_size}
        totalPages={results.data.total_pages}
        counts={results.data.counts}
        sorting={sorting}
        agents={agents.data.items}
        facets={facets.data.items}
        jobs={jobs.data.items}
        search={search}
        agentId={agentId}
        jobId={jobId}
        colo={effectiveGeo.colo}
        coloContinent={effectiveGeo.continent}
        coloCountry={effectiveGeo.country}
        coloCity={effectiveGeo.city}
        available={available}
        timeRange={timeRange}
        onSearchChange={(value) => { setSearch(value); setPage(1) }}
        onAgentChange={(value) => { setAgentId(value); setJobId(''); resetGeoFilters(); setPage(1) }}
        onJobChange={(value) => { setJobId(value); resetGeoFilters(); setPage(1) }}
        onColoChange={(value) => { setColo(value); setPage(1) }}
        onColoContinentChange={(value) => { setColoContinent(value); setColoCountry(''); setColoCity(''); setColo(''); setPage(1) }}
        onColoCountryChange={(value) => { setColoCountry(value); setColoCity(''); setColo(''); setPage(1) }}
        onColoCityChange={(value) => { setColoCity(value); setColo(''); setPage(1) }}
        onAvailableChange={(value) => { setAvailable(value); resetGeoFilters(); setPage(1) }}
        onTimeRangeChange={(value) => { setTimeRange(value); setJobId(''); resetGeoFilters(); setPage(1) }}
        onPaginationChange={(pagination: PaginationState) => { setPage(pagination.pageIndex + 1); setPageSize(pagination.pageSize) }}
        onSortingChange={(next) => { setSorting(next.slice(0, 1)); setPage(1) }}
        onReset={resetAll}
      />
    </div>
  )
}
