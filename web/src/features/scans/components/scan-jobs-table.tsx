import { useMemo, useState } from 'react'
import type { ColumnDef, Table as TanStackTable } from '@tanstack/react-table'
import { Copy, MoreHorizontal, Square } from 'lucide-react'
import { toast } from 'sonner'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import type { ScanJob } from '@/features/scans/types'
import { useStopScanJob } from '@/features/scans/hooks'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Progress } from '@/components/ui/progress'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { DataTable } from '@/components/shared/data-table/data-table'
import { DataTableColumnHeader } from '@/components/shared/data-table/data-table-column-header'
import { DataTableViewOptions } from '@/components/shared/data-table/data-table-view-options'
import { DataTableResetButton } from '@/components/shared/data-table/data-table-reset-button'
import { StatusBadge } from '@/components/shared/status-badge'
import { SearchInput } from '@/components/shared/search-input'
import { formatDate } from '@/lib/format'
import { useAuth } from '@/features/auth/auth-context'

function JobToolbar({ table }: { table: TanStackTable<ScanJob> }) {
  const status = String(table.getColumn('status')?.getFilterValue() ?? 'all')
  const filtered = Boolean(table.getColumn('name')?.getFilterValue() || status !== 'all')
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <SearchInput
        className="flex-1 sm:max-w-sm"
        value={String(table.getColumn('name')?.getFilterValue() ?? '')}
        onChange={(event) => table.getColumn('name')?.setFilterValue(event.target.value)}
        placeholder="搜索任务名称"
      />
      <Select
        items={{ all: '全部状态', pending: '等待中', running: '运行中', completed: '已完成', stopped: '已停止' }}
        value={status}
        onValueChange={(value) => table.getColumn('status')?.setFilterValue(value === 'all' ? undefined : value)}
      >
        <SelectTrigger className="w-full sm:w-36" aria-label="任务状态"><SelectValue placeholder="任务状态" /></SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部状态</SelectItem>
          <SelectItem value="pending">等待中</SelectItem>
          <SelectItem value="running">运行中</SelectItem>
          <SelectItem value="completed">已完成</SelectItem>
          <SelectItem value="stopped">已停止</SelectItem>
        </SelectContent>
      </Select>
      <DataTableResetButton disabled={!filtered} onClick={() => { table.getColumn('name')?.setFilterValue(''); table.getColumn('status')?.setFilterValue(undefined) }} />
      <DataTableViewOptions table={table} />
    </div>
  )
}

export function ScanJobsTable({ jobs }: { jobs: ScanJob[] }) {
  const auth = useAuth()
  const [stopping, setStopping] = useState<ScanJob | null>(null)
  const stopJob = useStopScanJob()
  const columns = useMemo<ColumnDef<ScanJob>[]>(() => [
    {
      accessorKey: 'name',
      meta: { label: '任务' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="任务" />,
      cell: ({ row }) => (
        <div className="min-w-56">
          <p className="font-medium">{row.original.name}</p>
          <p className="mt-1 font-mono text-xs text-muted-foreground">{row.original.scheme}://{row.original.hostname}:{row.original.port}{row.original.path}</p>
        </div>
      ),
    },
    {
      accessorKey: 'kind',
      meta: { label: '类型' },
      header: '类型',
      cell: ({ row }) => <div className="flex flex-wrap gap-1"><Badge variant="secondary">{row.original.kind === 'blacklist_recheck' ? '黑名单复检' : row.original.kind === 'scheduled' ? '定时扫描' : '常规扫描'}</Badge>{row.original.sampling_mode === 'one_per_prefix' ? <Badge variant="outline">每前缀 1 IP</Badge> : null}</div>,
      filterFn: 'equalsString',
    },
    {
      accessorKey: 'status',
      meta: { label: '状态' },
      header: '状态',
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
      filterFn: 'equalsString',
    },
    {
      accessorKey: 'progress',
      meta: { label: '进度' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="进度" />,
      cell: ({ row }) => (
        <div className="min-w-44 space-y-2">
          <div className="flex justify-between text-xs text-muted-foreground"><span>{row.original.completed_targets} / {row.original.total_targets}</span><span>{row.original.progress.toFixed(0)}%</span></div>
          <Progress value={row.original.progress} className="h-1.5" />
        </div>
      ),
    },
    {
      id: 'outcome',
      meta: { label: '结果' },
      header: '成功 / 失败',
      accessorFn: (row) => row.success_targets,
      cell: ({ row }) => <span className="metric-value"><span className="text-emerald-600 dark:text-emerald-400">{row.original.success_targets}</span><span className="mx-2 text-muted-foreground">/</span><span className="text-destructive">{row.original.failed_targets}</span></span>,
    },
    {
      accessorKey: 'created_at',
      meta: { label: '创建时间' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="创建时间" />,
      cell: ({ row }) => <span className="whitespace-nowrap text-muted-foreground">{formatDate(row.original.created_at)}</span>,
    },
    {
      id: 'actions',
      enableHiding: false,
      header: () => <span className="sr-only">操作</span>,
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" aria-label="任务操作" />}><MoreHorizontal /></DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>任务操作</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={async () => { await navigator.clipboard.writeText(row.original.id); toast.success('任务 ID 已复制') }}><Copy />复制任务 ID</DropdownMenuItem>
            <DropdownMenuItem onClick={async () => { await navigator.clipboard.writeText(`${row.original.scheme}://${row.original.hostname}:${row.original.port}${row.original.path}`); toast.success('探测地址已复制') }}><Copy />复制探测地址</DropdownMenuItem>
            {auth.canManage && ['pending', 'running'].includes(row.original.status) ? <><DropdownMenuSeparator /><DropdownMenuItem className="text-destructive focus:text-destructive" onSelect={() => setStopping(row.original)}><Square />{row.original.status === 'pending' ? '取消任务' : '停止剩余任务'}</DropdownMenuItem></> : null}
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ], [auth.canManage])

  return (
    <>
    <DataTable
      columns={columns}
      data={jobs}
      getRowId={(job) => job.id}
      emptyTitle="暂无扫描任务"
      emptyDescription="创建任务后，在线 Agent 会主动领取并执行。"
      renderToolbar={(table) => <JobToolbar table={table} />}
      renderMobileItem={(job) => (
        <Card size="sm" className="gap-0 p-4 py-4">
          <div className="flex items-start justify-between gap-3"><div className="min-w-0"><p className="truncate font-medium">{job.name}</p><p className="mt-1 truncate font-mono text-xs text-muted-foreground">{job.hostname}{job.path}</p></div><StatusBadge status={job.status} /></div>
          <div className="mt-4 space-y-2"><div className="flex justify-between text-xs text-muted-foreground"><span>{job.completed_targets} / {job.total_targets}</span><span>{job.progress.toFixed(0)}%</span></div><Progress value={job.progress} className="h-1.5" /></div>
          <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground"><span>成功 {job.success_targets} · 失败 {job.failed_targets}</span><span>{formatDate(job.created_at)}</span></div>{auth.canManage && ['pending', 'running'].includes(job.status) ? <Button className="mt-4 w-full" variant="outline" onClick={() => setStopping(job)}><Square />{job.status === 'pending' ? '取消任务' : '停止剩余任务'}</Button> : null}
        </Card>
      )}
    />
    <AlertDialog open={auth.canManage && Boolean(stopping)} onOpenChange={(open) => { if (!open) setStopping(null) }}>
      <AlertDialogContent>
        <AlertDialogHeader><AlertDialogTitle>{stopping?.status === 'pending' ? '取消这个扫描任务？' : '停止剩余扫描任务？'}</AlertDialogTitle><AlertDialogDescription>已完成的结果会保留；尚未领取和 Agent 当前批次中的未完成目标将被取消。停止后不能继续原任务，可复制配置重新创建。</AlertDialogDescription></AlertDialogHeader>
        <AlertDialogFooter><AlertDialogCancel>返回</AlertDialogCancel><AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" disabled={stopJob.isPending} onClick={async () => { if (!stopping) return; await stopJob.mutateAsync(stopping.id); setStopping(null) }}>{stopping?.status === 'pending' ? '确认取消' : '停止剩余任务'}</AlertDialogAction></AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
    </>
  )
}
