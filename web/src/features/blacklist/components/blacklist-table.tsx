import { useMemo } from 'react'
import type { ColumnDef, Table as TanStackTable } from '@tanstack/react-table'
import type { BlacklistEntry } from '@/features/blacklist/types'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { DataTable } from '@/components/shared/data-table/data-table'
import { DataTableColumnHeader } from '@/components/shared/data-table/data-table-column-header'
import { DataTableViewOptions } from '@/components/shared/data-table/data-table-view-options'
import { DataTableResetButton } from '@/components/shared/data-table/data-table-reset-button'
import { SearchInput } from '@/components/shared/search-input'
import { formatDate } from '@/lib/format'

function reasonLabel(reason: string) {
  const labels: Record<string, string> = {
    TIMEOUT: '超时',
    NETWORK_ERROR: '网络错误',
    HTTP_ERROR: 'HTTP 异常',
    PROBE_ERROR: '探测失败',
    HIGH_PACKET_LOSS: '高丢包',
    HIGH_LATENCY: '高延迟',
    UNAVAILABLE: '不可用',
  }
  return labels[reason] ?? reason
}

function BlacklistToolbar({ table }: { table: TanStackTable<BlacklistEntry> }) {
  const reason = String(table.getColumn('reason')?.getFilterValue() ?? 'all')
  const hasFilter = Boolean(table.getColumn('target_ip')?.getFilterValue() || reason !== 'all')
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <SearchInput className="flex-1 sm:max-w-sm" value={String(table.getColumn('target_ip')?.getFilterValue() ?? '')} onChange={(event) => table.getColumn('target_ip')?.setFilterValue(event.target.value)} placeholder="搜索 IP" />
      <Select
        items={{ all: '全部原因', TIMEOUT: '超时', NETWORK_ERROR: '网络错误', HIGH_PACKET_LOSS: '高丢包', HIGH_LATENCY: '高延迟' }}
        value={reason}
        onValueChange={(value) => table.getColumn('reason')?.setFilterValue(value === 'all' ? undefined : value)}
      >
        <SelectTrigger className="w-full sm:w-40" aria-label="黑名单原因"><SelectValue /></SelectTrigger>
        <SelectContent><SelectItem value="all">全部原因</SelectItem><SelectItem value="TIMEOUT">超时</SelectItem><SelectItem value="NETWORK_ERROR">网络错误</SelectItem><SelectItem value="HIGH_PACKET_LOSS">高丢包</SelectItem><SelectItem value="HIGH_LATENCY">高延迟</SelectItem></SelectContent>
      </Select>
      <DataTableResetButton disabled={!hasFilter} onClick={() => { table.getColumn('target_ip')?.setFilterValue(''); table.getColumn('reason')?.setFilterValue(undefined) }} />
      <DataTableViewOptions table={table} />
    </div>
  )
}

export function BlacklistTable({ items }: { items: BlacklistEntry[] }) {
  const columns = useMemo<ColumnDef<BlacklistEntry>[]>(() => [
    {
      accessorKey: 'target_ip',
      meta: { label: 'IP' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="IP" />,
      cell: ({ row }) => <span className="font-mono font-medium">{row.original.target_ip}</span>,
    },
    {
      accessorKey: 'agent_name',
      meta: { label: 'Agent' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="Agent / 地区" />,
      cell: ({ row }) => <div><p className="font-medium">{row.original.agent_name}</p><p className="text-xs text-muted-foreground">{row.original.continent} / {row.original.region}</p></div>,
    },
    {
      accessorKey: 'reason',
      meta: { label: '原因' },
      header: '原因',
      filterFn: 'equalsString',
      cell: ({ row }) => <Badge variant="destructive">{reasonLabel(row.original.reason)}</Badge>,
    },
    {
      accessorKey: 'failure_count',
      meta: { label: '失败次数' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="失败次数" />,
      cell: ({ row }) => <span className="metric-value">{row.original.failure_count}</span>,
    },
    {
      accessorKey: 'retry_after',
      meta: { label: '下次复检' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="下次复检" />,
      cell: ({ row }) => <span className="whitespace-nowrap">{formatDate(row.original.retry_after)}</span>,
    },
    {
      accessorKey: 'updated_at',
      meta: { label: '更新时间' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="更新时间" />,
      cell: ({ row }) => <span className="whitespace-nowrap text-muted-foreground">{formatDate(row.original.updated_at)}</span>,
    },
  ], [])

  return (
    <DataTable
      columns={columns}
      data={items}
      getRowId={(item) => `${item.agent_id}-${item.target_ip}`}
      emptyTitle="黑名单为空"
      emptyDescription="当前没有需要暂时过滤的 IP。"
      renderToolbar={(table) => <BlacklistToolbar table={table} />}
      renderMobileItem={(item) => (
        <Card size="sm" className="gap-0 p-4 py-4">
          <div className="flex items-start justify-between gap-3"><div><p className="font-mono font-medium">{item.target_ip}</p><p className="mt-1 text-xs text-muted-foreground">{item.agent_name} · {item.region}</p></div><Badge variant="destructive">{reasonLabel(item.reason)}</Badge></div>
          <div className="mt-4 grid grid-cols-2 gap-3 text-sm"><div><p className="text-xs text-muted-foreground">失败次数</p><p className="mt-1 font-medium">{item.failure_count}</p></div><div><p className="text-xs text-muted-foreground">下次复检</p><p className="mt-1">{formatDate(item.retry_after)}</p></div></div>
        </Card>
      )}
    />
  )
}
