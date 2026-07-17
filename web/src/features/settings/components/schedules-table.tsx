import { useMemo, useState } from 'react'
import type { ColumnDef, Table as TanStackTable } from '@tanstack/react-table'
import { MoreHorizontal, Pencil, Play, Trash2 } from 'lucide-react'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Switch } from '@/components/ui/switch'
import { DataTable } from '@/components/shared/data-table/data-table'
import { DataTableColumnHeader } from '@/components/shared/data-table/data-table-column-header'
import { DataTableViewOptions } from '@/components/shared/data-table/data-table-view-options'
import { DataTableResetButton } from '@/components/shared/data-table/data-table-reset-button'
import { SearchInput } from '@/components/shared/search-input'
import { useDeleteSchedule, useRunSchedule, useUpdateSchedule } from '@/features/settings/hooks'
import type { ScanSchedule, UpsertScanSchedule } from '@/features/settings/types'
import { formatDateInTimezone } from '@/lib/format'

const cronLabels: Record<string, string> = {
  '0 * * * *': '每小时',
  '0 */6 * * *': '每 6 小时',
  '0 2 * * *': '每天 02:00',
  '0 8 * * *': '每天 08:00',
  '0 8 * * 1': '每周一 08:00',
  '@hourly': '每小时',
  '@daily': '每天',
  '@weekly': '每周',
}

function scheduleInput(item: ScanSchedule, overrides: Partial<UpsertScanSchedule> = {}): UpsertScanSchedule {
  return {
    name: item.name,
    enabled: item.enabled,
    cron_expression: item.cron_expression,
    timezone: item.timezone,
    agent_ids: item.agent_ids,
    sampling_mode: item.sampling_mode,
    target_count: item.target_count,
    scheme: item.scheme,
    hostname: item.hostname,
    path: item.path,
    port: item.port,
    attempts: item.attempts,
    timeout_ms: item.timeout_ms,
    max_latency_ms: item.max_latency_ms,
    max_packet_loss: item.max_packet_loss,
    blacklist_minutes: item.blacklist_minutes,
    include_ipv6: item.include_ipv6,
    include_blocked: item.include_blocked,
    ...overrides,
  }
}

function SchedulesToolbar({ table }: { table: TanStackTable<ScanSchedule> }) {
  const filtered = Boolean(table.getColumn('name')?.getFilterValue())
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <SearchInput
        className="flex-1 sm:max-w-sm"
        placeholder="搜索计划名称"
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

export function SchedulesTable({ schedules, onEdit, canManage }: { schedules: ScanSchedule[]; onEdit: (schedule: ScanSchedule) => void; canManage: boolean }) {
  const [deleting, setDeleting] = useState<ScanSchedule | null>(null)
  const update = useUpdateSchedule()
  const remove = useDeleteSchedule()
  const run = useRunSchedule()

  const columns = useMemo<ColumnDef<ScanSchedule>[]>(() => [
    {
      accessorKey: 'name',
      meta: { label: '计划' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="计划" />,
      cell: ({ row }) => (
        <div className="min-w-48">
          <p className="font-medium">{row.original.name}</p>
          <p className="mt-1 font-mono text-xs text-muted-foreground">{row.original.hostname}{row.original.path}</p>
        </div>
      ),
    },
    {
      accessorKey: 'cron_expression',
      meta: { label: '频率' },
      header: '频率',
      cell: ({ row }) => (
        <div>
          <p>{cronLabels[row.original.cron_expression] ?? '自定义 Cron'}</p>
          <p className="mt-1 font-mono text-xs text-muted-foreground">{row.original.cron_expression} · {row.original.timezone}</p>
        </div>
      ),
    },
    {
      accessorKey: 'target_count',
      meta: { label: '扫描规模' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="扫描规模" />,
      cell: ({ row }) => <div><p>{row.original.sampling_mode === 'one_per_prefix' ? '每前缀 1 个 IP' : `${row.original.target_count} 个 IP`}</p><p className="mt-1 text-xs text-muted-foreground">{row.original.agent_ids.length || '全部在线'} Agent</p></div>,
    },
    {
      accessorKey: 'next_run_at',
      meta: { label: '下次执行' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="下次执行" />,
      cell: ({ row }) => <div><p>{formatDateInTimezone(row.original.next_run_at, row.original.timezone)}</p><p className="mt-1 text-xs text-muted-foreground">上次 {formatDateInTimezone(row.original.last_run_at, row.original.timezone)}</p></div>,
    },
    {
      accessorKey: 'enabled',
      meta: { label: '状态' },
      header: '状态',
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <Switch
            checked={row.original.enabled}
            disabled={update.isPending}
            onCheckedChange={(enabled) => update.mutate({ id: row.original.id, input: scheduleInput(row.original, { enabled }) })}
            aria-label={`${row.original.enabled ? '停用' : '启用'} ${row.original.name}`}
          />
          <Badge variant={row.original.enabled ? 'default' : 'secondary'}>{row.original.enabled ? '启用' : '停用'}</Badge>
        </div>
      ),
    },
    {
      id: 'actions',
      header: () => <span className="sr-only">操作</span>,
      cell: ({ row }) => canManage ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon" aria-label={`操作 ${row.original.name}`} />}><MoreHorizontal /></DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => run.mutate(row.original.id)} disabled={run.isPending}><Play />立即运行</DropdownMenuItem>
            <DropdownMenuItem onClick={() => onEdit(row.original)}><Pencil />编辑计划</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={() => setDeleting(row.original)}><Trash2 />删除计划</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : null,
    },
  ], [canManage, onEdit, run, update])

  return (
    <>
      <DataTable
        columns={columns}
        data={schedules}
        getRowId={(item) => item.id}
        initialPageSize={10}
        emptyTitle="暂无定时计划"
        emptyDescription="创建计划后，中心端会按 Cron 和时区自动生成扫描任务。"
        renderToolbar={(table) => <SchedulesToolbar table={table} />}
        renderMobileItem={(item) => (
          <Card>
            <CardContent className="space-y-4 p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0"><p className="truncate font-medium">{item.name}</p><p className="mt-1 font-mono text-xs text-muted-foreground">{item.cron_expression}</p></div>
                {canManage ? <Switch checked={item.enabled} onCheckedChange={(enabled) => update.mutate({ id: item.id, input: scheduleInput(item, { enabled }) })} aria-label={`${item.enabled ? '停用' : '启用'} ${item.name}`} /> : <Badge variant={item.enabled ? 'default' : 'secondary'}>{item.enabled ? '启用' : '停用'}</Badge>}
              </div>
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div><p className="text-xs text-muted-foreground">频率</p><p className="mt-1">{cronLabels[item.cron_expression] ?? '自定义'}</p></div>
                <div><p className="text-xs text-muted-foreground">下次执行</p><p className="mt-1">{formatDateInTimezone(item.next_run_at, item.timezone)}</p></div>
                <div><p className="text-xs text-muted-foreground">扫描规模</p><p className="mt-1">{item.sampling_mode === 'one_per_prefix' ? '每前缀 1 IP' : `${item.target_count} IP`}</p></div>
                <div><p className="text-xs text-muted-foreground">Agent</p><p className="mt-1">{item.agent_ids.length || '全部在线'}</p></div>
              </div>
              {item.last_error ? <p className="rounded-md bg-destructive/10 p-2 text-xs text-destructive">{item.last_error}</p> : null}
              {canManage ? <div className="flex gap-2"><Button variant="outline" size="sm" className="flex-1" onClick={() => run.mutate(item.id)}><Play />立即运行</Button><Button variant="outline" size="sm" className="flex-1" onClick={() => onEdit(item)}><Pencil />编辑</Button></div> : null}
            </CardContent>
          </Card>
        )}
      />
      <AlertDialog open={canManage && Boolean(deleting)} onOpenChange={(open) => { if (!open) setDeleting(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>删除定时计划？</AlertDialogTitle><AlertDialogDescription>“{deleting?.name}”的配置会被永久删除，已经创建的历史扫描任务不会受到影响。</AlertDialogDescription></AlertDialogHeader>
          <AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" onClick={() => { if (deleting) remove.mutate(deleting.id); setDeleting(null) }}>删除</AlertDialogAction></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
