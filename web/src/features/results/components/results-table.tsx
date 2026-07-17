import { useMemo } from 'react'
import type { ColumnDef, PaginationState, SortingState, Table as TanStackTable } from '@tanstack/react-table'
import { Copy, MoreHorizontal } from 'lucide-react'
import { toast } from 'sonner'
import type { Agent } from '@/features/agents/types'
import type { ResultColoFacet, ResultJobFacet, ResultStatusCounts, ResultTimeRange, ScanResult } from '@/features/results/types'
import type { GeoFacetOption } from '@/features/results/geo-facets'
import { buildGeoFacetOptions } from '@/features/results/geo-facets'
import { ColoLocationLabel } from '@/features/results/components/colo-location-label'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { SearchableCombobox, type SearchableComboboxOption } from '@/components/shared/searchable-combobox'
import { SearchInput } from '@/components/shared/search-input'
import { DataTable } from '@/components/shared/data-table/data-table'
import { DataTableColumnHeader } from '@/components/shared/data-table/data-table-column-header'
import { DataTableViewOptions } from '@/components/shared/data-table/data-table-view-options'
import { DataTableResetButton } from '@/components/shared/data-table/data-table-reset-button'
import { formatDate, formatMS, formatNumber, formatPercent } from '@/lib/format'

interface ResultsTableProps {
  results: ScanResult[]
  total: number
  page: number
  pageSize: number
  totalPages: number
  counts: ResultStatusCounts
  sorting: SortingState
  agents: Agent[]
  facets: ResultColoFacet[]
  jobs: ResultJobFacet[]
  search: string
  agentId: string
  jobId: string
  colo: string
  coloContinent: string
  coloCountry: string
  coloCity: string
  available: string
  timeRange: ResultTimeRange
  onSearchChange: (value: string) => void
  onAgentChange: (value: string) => void
  onJobChange: (value: string) => void
  onColoChange: (value: string) => void
  onColoContinentChange: (value: string) => void
  onColoCountryChange: (value: string) => void
  onColoCityChange: (value: string) => void
  onAvailableChange: (value: string) => void
  onTimeRangeChange: (value: ResultTimeRange) => void
  onPaginationChange: (pagination: PaginationState) => void
  onSortingChange: (sorting: SortingState) => void
  onReset: () => void
}

function facetOptions(options: GeoFacetOption[]): SearchableComboboxOption[] {
  return options.map((item) => ({
    value: item.value,
    label: item.label,
    searchText: item.label,
    countLabel: formatNumber(item.count),
  }))
}

function FacetCombobox({
  value,
  placeholder,
  searchPlaceholder,
  options,
  onChange,
}: {
  value: string
  placeholder: string
  searchPlaceholder: string
  options: GeoFacetOption[]
  onChange: (value: string) => void
}) {
  return (
    <SearchableCombobox
      value={value}
      options={facetOptions(options)}
      onValueChange={onChange}
      placeholder={placeholder}
      searchPlaceholder={searchPlaceholder}
      emptyText="没有匹配的地理选项"
      allOption={{ value: '', label: placeholder }}
      aria-label={placeholder}
      disabled={options.length === 0}
    />
  )
}

function ResultsToolbar({ table, ...props }: ResultsTableProps & { table: TanStackTable<ScanResult> }) {
  const geoOptions = useMemo(
    () => buildGeoFacetOptions(props.facets, { continent: props.coloContinent, country: props.coloCountry, city: props.coloCity }),
    [props.facets, props.coloContinent, props.coloCountry, props.coloCity],
  )
  const filtered = Boolean(
    props.search || props.agentId || props.jobId || props.colo || props.coloContinent ||
    props.coloCountry || props.coloCity || props.available !== 'true' || props.timeRange !== '24h',
  )

  return (
    <div className="space-y-3">
      <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-[minmax(14rem,1fr)_minmax(14rem,1fr)_13rem_10rem_auto_auto]">
        <SearchInput value={props.search} onChange={(event) => props.onSearchChange(event.target.value)} placeholder="搜索 IP" />
        <SearchableCombobox
          value={props.jobId}
          options={props.jobs.map((job) => ({
            value: job.id,
            label: job.name,
            searchText: `${job.name} ${job.kind}`,
            countLabel: formatNumber(job.count),
          }))}
          onValueChange={props.onJobChange}
          placeholder="全部扫描任务"
          searchPlaceholder="搜索扫描任务…"
          emptyText="没有匹配的扫描任务"
          allOption={{ value: '', label: '全部扫描任务' }}
          aria-label="扫描任务"
        />
        <SearchableCombobox
          value={props.agentId}
          options={props.agents.map((agent) => ({
            value: agent.id,
            label: `${agent.name} · ${agent.region}`,
            searchText: `${agent.name} ${agent.region} ${agent.continent}`,
          }))}
          onValueChange={props.onAgentChange}
          placeholder="全部 Agent"
          searchPlaceholder="搜索 Agent、区域…"
          emptyText="没有匹配的 Agent"
          allOption={{ value: '', label: '全部 Agent' }}
          aria-label="Agent"
        />
        <Select
          items={{
            all: `全部状态 · ${formatNumber(props.counts.all)}`,
            true: `仅可用 · ${formatNumber(props.counts.available)}`,
            false: `仅失败 · ${formatNumber(props.counts.failed)}`,
          }}
          value={props.available || 'all'}
          onValueChange={(value) => props.onAvailableChange(value === 'all' ? '' : value)}
        >
          <SelectTrigger className="w-full" aria-label="可用状态"><SelectValue placeholder="全部状态" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态 · {formatNumber(props.counts.all)}</SelectItem>
            <SelectItem value="true">仅可用 · {formatNumber(props.counts.available)}</SelectItem>
            <SelectItem value="false">仅失败 · {formatNumber(props.counts.failed)}</SelectItem>
          </SelectContent>
        </Select>
        <Select
          items={{ '1h': '最近 1 小时', '24h': '最近 24 小时', '7d': '最近 7 天', '30d': '最近 30 天', all: '全部历史' }}
          value={props.timeRange}
          onValueChange={(value) => props.onTimeRangeChange(value as ResultTimeRange)}
        >
          <SelectTrigger className="w-full md:w-40" aria-label="时间范围"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="1h">最近 1 小时</SelectItem>
            <SelectItem value="24h">最近 24 小时</SelectItem>
            <SelectItem value="7d">最近 7 天</SelectItem>
            <SelectItem value="30d">最近 30 天</SelectItem>
            <SelectItem value="all">全部历史</SelectItem>
          </SelectContent>
        </Select>
        <div className="flex items-center justify-end gap-1">
          <DataTableResetButton disabled={!filtered} onClick={props.onReset} />
          <DataTableViewOptions table={table} />
        </div>
      </div>
      <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
        <FacetCombobox value={props.coloContinent} placeholder="全部大洲" searchPlaceholder="搜索大洲…" options={geoOptions.continents} onChange={props.onColoContinentChange} />
        <FacetCombobox value={props.coloCountry} placeholder="全部国家/地区" searchPlaceholder="搜索国家或地区…" options={geoOptions.countries} onChange={props.onColoCountryChange} />
        <FacetCombobox value={props.coloCity} placeholder="全部城市" searchPlaceholder="搜索城市…" options={geoOptions.cities} onChange={props.onColoCityChange} />
        <FacetCombobox value={props.colo} placeholder="全部 colo" searchPlaceholder="搜索 colo、城市或国家…" options={geoOptions.colos} onChange={props.onColoChange} />
      </div>
    </div>
  )
}

export function ResultsTable(props: ResultsTableProps) {
  const columns = useMemo<ColumnDef<ScanResult>[]>(() => [
    { accessorKey: 'target_ip', meta: { label: 'IP' }, header: ({ column }) => <DataTableColumnHeader column={column} title="IP" />, cell: ({ row }) => <div className="min-w-40"><p className="font-mono font-medium">{row.original.target_ip}</p><p className="mt-1 max-w-56 truncate text-xs text-muted-foreground">{row.original.job_name}</p></div> },
    { accessorKey: 'agent_name', meta: { label: 'Agent' }, header: ({ column }) => <DataTableColumnHeader column={column} title="Agent / 出口" />, cell: ({ row }) => <div><p className="font-medium">{row.original.agent_name}</p><p className="text-xs text-muted-foreground">{row.original.continent} / {row.original.region}</p></div> },
    { accessorKey: 'colo', meta: { label: 'colo 位置' }, header: ({ column }) => <DataTableColumnHeader column={column} title="colo / 位置" />, cell: ({ row }) => <div className="min-w-64"><ColoLocationLabel code={row.original.colo} city={row.original.colo_city} country={row.original.colo_country} continent={row.original.colo_continent} /></div> },
    { accessorKey: 'latency_ms', meta: { label: 'TTFB' }, header: ({ column }) => <DataTableColumnHeader column={column} title="TTFB" />, cell: ({ row }) => row.original.available ? <span className="metric-value font-medium text-emerald-700 dark:text-emerald-400">{formatMS(row.original.latency_ms)}</span> : <Badge variant="destructive">{row.original.error_code || '失败'}</Badge> },
    { id: 'connection', enableSorting: false, meta: { label: '连接耗时' }, accessorFn: (row) => row.tcp_connect_ms + row.tls_handshake_ms, header: 'TCP / TLS', cell: ({ row }) => <div className="metric-value"><p>{formatMS(row.original.tcp_connect_ms)}</p><p className="text-xs text-muted-foreground">{formatMS(row.original.tls_handshake_ms)}</p></div> },
    { accessorKey: 'packet_loss', meta: { label: '丢包率' }, header: ({ column }) => <DataTableColumnHeader column={column} title="丢包率" />, cell: ({ row }) => <span className={row.original.packet_loss > 0 ? 'metric-value text-amber-700 dark:text-amber-400' : 'metric-value'}>{formatPercent(row.original.packet_loss)}</span> },
    { accessorKey: 'http_status', meta: { label: 'HTTP' }, header: ({ column }) => <DataTableColumnHeader column={column} title="HTTP" />, cell: ({ row }) => <div><p>{row.original.http_status || '—'}</p><p className="text-xs text-muted-foreground">{row.original.http_version || row.original.tls_version || '—'}</p></div> },
    { accessorKey: 'scanned_at', meta: { label: '扫描时间' }, header: ({ column }) => <DataTableColumnHeader column={column} title="扫描时间" />, cell: ({ row }) => <span className="whitespace-nowrap text-muted-foreground">{formatDate(row.original.scanned_at)}</span> },
    { id: 'actions', enableSorting: false, enableHiding: false, header: () => <span className="sr-only">操作</span>, cell: ({ row }) => <DropdownMenu><DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" aria-label="结果操作" />}><MoreHorizontal /></DropdownMenuTrigger><DropdownMenuContent align="end"><DropdownMenuLabel>结果操作</DropdownMenuLabel><DropdownMenuSeparator /><DropdownMenuItem onClick={async () => { await navigator.clipboard.writeText(row.original.target_ip); toast.success('IP 已复制') }}><Copy />复制 IP</DropdownMenuItem>{row.original.cf_ray ? <DropdownMenuItem onClick={async () => { await navigator.clipboard.writeText(row.original.cf_ray); toast.success('CF-RAY 已复制') }}><Copy />复制 CF-RAY</DropdownMenuItem> : null}</DropdownMenuContent></DropdownMenu> },
  ], [])

  return (
    <DataTable
      columns={columns}
      data={props.results}
      getRowId={(result) => String(result.id)}
      initialPageSize={props.pageSize}
      emptyTitle="暂无匹配结果"
      emptyDescription="运行扫描任务，或调整任务、时间范围、Agent、地理位置和可用状态筛选。"
      renderToolbar={(table) => <ResultsToolbar {...props} table={table} />}
      serverPagination={{
        pageIndex: props.page - 1,
        pageSize: props.pageSize,
        pageCount: props.totalPages,
        total: props.total,
        onChange: props.onPaginationChange,
      }}
      serverSorting={{ sorting: props.sorting, onChange: props.onSortingChange }}
      renderMobileItem={(item) => <Card size="sm" className="gap-0 p-4 py-4"><div className="min-w-0"><p className="truncate font-mono font-medium">{item.target_ip}</p><p className="mt-1 text-xs text-muted-foreground">{item.agent_name} · {item.region}</p><p className="mt-1 truncate text-xs text-muted-foreground">{item.job_name}</p></div><div className="mt-3"><ColoLocationLabel code={item.colo} city={item.colo_city} country={item.colo_country} continent={item.colo_continent} /></div><div className="mt-4 grid grid-cols-3 gap-3 text-sm"><div><p className="text-xs text-muted-foreground">TTFB</p><p className={item.available ? 'mt-1 font-medium text-emerald-700 dark:text-emerald-400' : 'mt-1 text-destructive'}>{item.available ? formatMS(item.latency_ms) : item.error_code || '失败'}</p></div><div><p className="text-xs text-muted-foreground">丢包</p><p className="mt-1">{formatPercent(item.packet_loss)}</p></div><div><p className="text-xs text-muted-foreground">HTTP</p><p className="mt-1">{item.http_status || '—'}</p></div></div><p className="mt-4 text-xs text-muted-foreground">{formatDate(item.scanned_at)}</p></Card>}
    />
  )
}
