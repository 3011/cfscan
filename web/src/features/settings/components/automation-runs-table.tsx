import { useMemo } from 'react'
import type { ColumnDef, Table as TanStackTable } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { DataTable } from '@/components/shared/data-table/data-table'
import { DataTableColumnHeader } from '@/components/shared/data-table/data-table-column-header'
import { DataTableViewOptions } from '@/components/shared/data-table/data-table-view-options'
import { DataTableResetButton } from '@/components/shared/data-table/data-table-reset-button'
import { ErrorState } from '@/components/shared/error-state'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { StatusBadge } from '@/components/shared/status-badge'
import { SearchInput } from '@/components/shared/search-input'
import { useAutomationRuns } from '@/features/settings/hooks'
import type { AutomationRun } from '@/features/settings/types'
import { formatDate } from '@/lib/format'

function typeLabel(type: AutomationRun['automation_type']) {
  return type === 'scan_schedule' ? '扫描计划' : type === 'blacklist_recheck' ? '黑名单复查' : '数据源同步'
}
function triggerLabel(trigger: AutomationRun['trigger']) {
  return trigger === 'scheduled' ? '按计划' : trigger === 'startup' ? '启动执行' : '手动触发'
}
function summaryText(run: AutomationRun) {
  const summary = run.summary ?? {}
  if (typeof summary.tasks === 'number') return `${summary.tasks} 个 Agent 目标任务`
  if (typeof summary.targets === 'number') return run.status === 'skipped' ? String(summary.reason ?? '已跳过') : `${summary.targets} 个目标 · ${summary.jobs ?? 0} 个任务组`
  if (typeof summary.prefix_count === 'number') return `${summary.prefix_count} 个前缀`
  if (typeof summary.synced === 'number') return `成功 ${summary.synced} · 失败 ${summary.failed ?? 0}`
  if (typeof summary.locations === 'number') return `${summary.locations} 个 colo 位置`
  return run.error || '已完成'
}

function AutomationRunsToolbar({ table }: { table: TanStackTable<AutomationRun> }) {
  const filtered = Boolean(table.getColumn('name')?.getFilterValue())
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <SearchInput
        className="flex-1 sm:max-w-sm"
        placeholder="搜索自动化名称"
        value={String(table.getColumn('name')?.getFilterValue() ?? '')}
        onChange={(event) => table.getColumn('name')?.setFilterValue(event.target.value)}
      />
      <DataTableResetButton
        disabled={!filtered}
        onClick={() => table.getColumn('name')?.setFilterValue('')}
      />
      <DataTableViewOptions table={table} />
    </div>
  )
}

export function AutomationRunsTable() {
  const runs = useAutomationRuns()
  const columns = useMemo<ColumnDef<AutomationRun>[]>(() => [
    { accessorKey: 'name', meta: { label: '自动化' }, header: ({ column }) => <DataTableColumnHeader column={column} title="自动化" />, cell: ({ row }) => <div className="min-w-48"><p className="font-medium">{row.original.name}</p><div className="mt-1 flex gap-1"><Badge variant="secondary">{typeLabel(row.original.automation_type)}</Badge><Badge variant="outline">{triggerLabel(row.original.trigger)}</Badge></div></div> },
    { accessorKey: 'status', meta: { label: '状态' }, header: '状态', cell: ({ row }) => <StatusBadge status={row.original.status} /> },
    { id: 'summary', meta: { label: '执行结果' }, header: '执行结果', accessorFn: (row) => summaryText(row), cell: ({ row }) => <div className="max-w-md"><p>{summaryText(row.original)}</p>{row.original.error ? <p className="mt-1 text-xs text-destructive">{row.original.error}</p> : null}</div> },
    { accessorKey: 'started_at', meta: { label: '开始时间' }, header: ({ column }) => <DataTableColumnHeader column={column} title="开始时间" />, cell: ({ row }) => <span className="whitespace-nowrap text-muted-foreground">{formatDate(row.original.started_at)}</span> },
  ], [])
  if (runs.isPending) return <PageSkeleton rows={8} />
  if (runs.isError) return <ErrorState error={runs.error} onRetry={() => runs.refetch()} />
  return <DataTable columns={columns} data={runs.data.items} getRowId={(item) => item.id} initialPageSize={20} emptyTitle="暂无执行记录" emptyDescription="自动计划运行后会在这里保存配置快照、触发原因和结果。" renderToolbar={(table) => <AutomationRunsToolbar table={table} />} renderMobileItem={(item) => <Card size="sm" className="gap-0 p-4 py-4"><div className="flex items-start justify-between gap-3"><div><p className="font-medium">{item.name}</p><p className="mt-1 text-xs text-muted-foreground">{typeLabel(item.automation_type)} · {triggerLabel(item.trigger)}</p></div><StatusBadge status={item.status} /></div><p className="mt-3 text-sm">{summaryText(item)}</p><p className="mt-3 text-xs text-muted-foreground">{formatDate(item.started_at)}</p></Card>} />
}
