import { useMemo } from 'react'
import type { ColumnDef, Table as TanStackTable } from '@tanstack/react-table'
import type { Agent } from '@/features/agents/types'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { DataTable } from '@/components/shared/data-table/data-table'
import { DataTableColumnHeader } from '@/components/shared/data-table/data-table-column-header'
import { DataTableViewOptions } from '@/components/shared/data-table/data-table-view-options'
import { DataTableResetButton } from '@/components/shared/data-table/data-table-reset-button'
import { StatusBadge } from '@/components/shared/status-badge'
import { formatDate } from '@/lib/format'
import { LiveRelativeTime } from '@/components/shared/live-relative-time'
import { SearchInput } from '@/components/shared/search-input'

function AgentsToolbar({ table }: { table: TanStackTable<Agent> }) {
  const status = String(table.getColumn('status')?.getFilterValue() ?? 'all')
  const hasFilter = Boolean(table.getColumn('search')?.getFilterValue() || status !== 'all')
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <SearchInput className="flex-1 sm:max-w-sm" value={String(table.getColumn('search')?.getFilterValue() ?? '')} onChange={(event) => table.getColumn('search')?.setFilterValue(event.target.value)} placeholder="搜索名称、地区或大洲" />
      <Select
        items={{ all: '全部状态', online: '在线', offline: '离线' }}
        value={status}
        onValueChange={(value) => table.getColumn('status')?.setFilterValue(value === 'all' ? undefined : value)}
      >
        <SelectTrigger className="w-full sm:w-32" aria-label="Agent 状态"><SelectValue /></SelectTrigger>
        <SelectContent><SelectItem value="all">全部状态</SelectItem><SelectItem value="online">在线</SelectItem><SelectItem value="offline">离线</SelectItem></SelectContent>
      </Select>
      <DataTableResetButton disabled={!hasFilter} onClick={() => { table.getColumn('search')?.setFilterValue(''); table.getColumn('status')?.setFilterValue(undefined) }} />
      <DataTableViewOptions table={table} />
    </div>
  )
}

export function AgentsTable({ agents }: { agents: Agent[] }) {
  const columns = useMemo<ColumnDef<Agent>[]>(() => [
    { id: 'search', accessorFn: (row) => `${row.name} ${row.region} ${row.continent}`, filterFn: 'includesString', enableHiding: false, header: () => null, cell: () => null },
    {
      accessorKey: 'name',
      meta: { label: '名称' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="Agent" />,
      cell: ({ row }) => <div><p className="font-medium">{row.original.name}</p><p className="mt-1 font-mono text-xs text-muted-foreground">{row.original.id.slice(0, 8)}</p></div>,
    },
    {
      accessorKey: 'continent',
      meta: { label: '位置' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="大洲 / 地区" />,
      cell: ({ row }) => <div className="flex items-center gap-2"><Badge variant="secondary">{row.original.continent}</Badge><span>{row.original.region}</span></div>,
    },
    {
      accessorKey: 'concurrency',
      meta: { label: '并发' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="并发" />,
      cell: ({ row }) => <span className="metric-value">{row.original.concurrency}</span>,
    },
    {
      accessorKey: 'status',
      meta: { label: '状态' },
      header: '状态',
      filterFn: 'equalsString',
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      accessorKey: 'last_seen_at',
      meta: { label: '最后心跳' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="最后心跳" />,
      cell: ({ row }) => <div><p><LiveRelativeTime value={row.original.last_seen_at} /></p><p className="text-xs text-muted-foreground">{formatDate(row.original.last_seen_at)}</p></div>,
    },
    {
      accessorKey: 'created_at',
      meta: { label: '注册时间' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="注册时间" />,
      cell: ({ row }) => <span className="text-muted-foreground">{formatDate(row.original.created_at)}</span>,
    },
  ], [])

  return (
    <DataTable
      columns={columns}
      data={agents}
      getRowId={(agent) => agent.id}
      initialColumnVisibility={{ search: false }}
      emptyTitle="暂无 Agent"
      emptyDescription="在地区服务器启动 Agent 并配置中心地址与 Token 后，节点会自动注册。"
      renderToolbar={(table) => <AgentsToolbar table={table} />}
      renderMobileItem={(agent) => (
        <Card size="sm" className="gap-0 p-4 py-4">
          <div className="flex items-start justify-between gap-3"><div><p className="font-medium">{agent.name}</p><p className="mt-1 text-xs text-muted-foreground">{agent.continent} / {agent.region}</p></div><StatusBadge status={agent.status} /></div>
          <div className="mt-4 grid grid-cols-2 gap-3 text-sm"><div><p className="text-xs text-muted-foreground">并发</p><p className="mt-1 font-medium">{agent.concurrency}</p></div><div><p className="text-xs text-muted-foreground">最后心跳</p><p className="mt-1"><LiveRelativeTime value={agent.last_seen_at} /></p></div></div>
        </Card>
      )}
    />
  )
}
