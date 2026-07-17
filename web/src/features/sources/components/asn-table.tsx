import { useMemo, useState } from 'react'
import type { ColumnDef, Table as TanStackTable } from '@tanstack/react-table'
import { MoreHorizontal, RefreshCw, Trash2 } from 'lucide-react'
import type { ASNSource } from '@/features/sources/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { DataTable } from '@/components/shared/data-table/data-table'
import { DataTableColumnHeader } from '@/components/shared/data-table/data-table-column-header'
import { DataTableViewOptions } from '@/components/shared/data-table/data-table-view-options'
import { DataTableResetButton } from '@/components/shared/data-table/data-table-reset-button'
import { StatusBadge } from '@/components/shared/status-badge'
import { SearchInput } from '@/components/shared/search-input'
import { formatDate, formatNumber } from '@/lib/format'
import { useDeleteASN, useSyncASN, useUpdateASN } from '@/features/sources/hooks'

function ASNToolbar({ table }: { table: TanStackTable<ASNSource> }) {
  const enabled = String(table.getColumn('enabled')?.getFilterValue() ?? 'all')
  const filtered = Boolean(table.getColumn('search')?.getFilterValue() || enabled !== 'all')
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <SearchInput className="flex-1 sm:max-w-sm" value={String(table.getColumn('search')?.getFilterValue() ?? '')} onChange={(event) => table.getColumn('search')?.setFilterValue(event.target.value)} placeholder="搜索 ASN、名称或组织" />
      <Select
        items={{ all: '全部状态', enabled: '已启用', disabled: '已停用' }}
        value={enabled}
        onValueChange={(value) => table.getColumn('enabled')?.setFilterValue(value === 'all' ? undefined : value === 'enabled')}
      >
        <SelectTrigger className="w-full sm:w-36" aria-label="启用状态"><SelectValue /></SelectTrigger>
        <SelectContent><SelectItem value="all">全部状态</SelectItem><SelectItem value="enabled">已启用</SelectItem><SelectItem value="disabled">已停用</SelectItem></SelectContent>
      </Select>
      <DataTableResetButton disabled={!filtered} onClick={() => { table.getColumn('search')?.setFilterValue(''); table.getColumn('enabled')?.setFilterValue(undefined) }} />
      <DataTableViewOptions table={table} />
    </div>
  )
}

export function ASNTable({ items, canManage }: { items: ASNSource[]; canManage: boolean }) {
  const [deleting, setDeleting] = useState<ASNSource | null>(null)
  const sync = useSyncASN()
  const update = useUpdateASN()
  const remove = useDeleteASN()
  const columns = useMemo<ColumnDef<ASNSource>[]>(() => {
    const base: ColumnDef<ASNSource>[] = [
    {
      id: 'search',
      accessorFn: (row) => `AS${row.asn} ${row.name} ${row.organization}`,
      filterFn: 'includesString',
      enableHiding: false,
      header: () => null,
      cell: () => null,
    },
    {
      accessorKey: 'asn',
      meta: { label: 'ASN' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="ASN" />,
      cell: ({ row }) => <span className="font-mono font-semibold">AS{row.original.asn}</span>,
    },
    {
      accessorKey: 'name',
      meta: { label: '名称' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="名称 / 组织" />,
      cell: ({ row }) => <div className="min-w-56"><div className="flex items-center gap-2"><p className="font-medium">{row.original.name}</p>{row.original.managed ? <Badge variant="secondary">内置</Badge> : null}</div><p className="mt-1 text-xs text-muted-foreground">{row.original.organization}</p></div>,
    },
    {
      accessorKey: 'enabled',
      meta: { label: '启用' },
      header: '启用',
      filterFn: 'equals',
      cell: ({ row }) => canManage ? <Switch checked={row.original.enabled} disabled={update.isPending} onCheckedChange={(enabled) => update.mutate({ asn: row.original.asn, input: { enabled } })} aria-label={`${row.original.enabled ? '停用' : '启用'} AS${row.original.asn}`} /> : <Badge variant={row.original.enabled ? 'default' : 'secondary'}>{row.original.enabled ? '已启用' : '已停用'}</Badge>,
    },
    {
      accessorKey: 'status',
      meta: { label: '同步状态' },
      header: '同步状态',
      cell: ({ row }) => <div><StatusBadge status={row.original.status} />{row.original.last_error ? <p className="mt-1 max-w-56 truncate text-xs text-destructive" title={row.original.last_error}>{row.original.last_error}</p> : null}</div>,
    },
    {
      accessorKey: 'prefix_count',
      meta: { label: '前缀数量' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="前缀" />,
      cell: ({ row }) => <span className="metric-value">{formatNumber(row.original.prefix_count)}</span>,
    },
    {
      id: 'ip_versions',
      meta: { label: 'IPv4 / IPv6' },
      accessorFn: (row) => row.ipv4_count + row.ipv6_count,
      header: 'IPv4 / IPv6',
      cell: ({ row }) => <span className="metric-value">{formatNumber(row.original.ipv4_count)} / {formatNumber(row.original.ipv6_count)}</span>,
    },
    {
      accessorKey: 'last_synced_at',
      meta: { label: '最后同步' },
      header: ({ column }) => <DataTableColumnHeader column={column} title="最后同步" />,
      cell: ({ row }) => <span className="whitespace-nowrap text-muted-foreground">{formatDate(row.original.last_synced_at)}</span>,
    },
    ]
    return canManage ? [...base, {
      id: 'actions',
      enableHiding: false,
      header: () => <span className="sr-only">操作</span>,
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" aria-label="ASN 操作" />}><MoreHorizontal /></DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>AS{row.original.asn}</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem disabled={sync.isPending} onClick={() => sync.mutate(row.original.asn)}><RefreshCw />立即同步</DropdownMenuItem>
            <DropdownMenuItem disabled={update.isPending} onClick={() => update.mutate({ asn: row.original.asn, input: { enabled: !row.original.enabled } })}>{row.original.enabled ? '停用数据源' : '启用数据源'}</DropdownMenuItem>
            {!row.original.managed ? <><DropdownMenuSeparator /><DropdownMenuItem className="text-destructive focus:text-destructive" onClick={() => setDeleting(row.original)}><Trash2 />删除数据源</DropdownMenuItem></> : null}
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },] : base
  }, [canManage, sync, update])

  return (
    <>
      <DataTable
        columns={columns}
        data={items}
        getRowId={(item) => String(item.asn)}
        initialColumnVisibility={{ search: false }}
        emptyTitle="暂无 ASN 数据源"
        emptyDescription="添加确认属于 Cloudflare 的 ASN，或重新加载内置数据源。"
        renderToolbar={(table) => <ASNToolbar table={table} />}
        renderMobileItem={(item) => (
          <Card size="sm" className="gap-0 p-4 py-4">
            <div className="flex items-start justify-between gap-3"><div><div className="flex items-center gap-2"><p className="font-mono font-semibold">AS{item.asn}</p>{item.managed ? <Badge variant="secondary">内置</Badge> : null}</div><p className="mt-1 text-sm font-medium">{item.name}</p><p className="text-xs text-muted-foreground">{item.organization}</p></div>{canManage ? <Switch checked={item.enabled} onCheckedChange={(enabled) => update.mutate({ asn: item.asn, input: { enabled } })} /> : <Badge variant={item.enabled ? 'default' : 'secondary'}>{item.enabled ? '已启用' : '已停用'}</Badge>}</div>
            <div className="mt-4 grid grid-cols-3 gap-3 text-sm"><div><p className="text-xs text-muted-foreground">状态</p><div className="mt-1"><StatusBadge status={item.status} /></div></div><div><p className="text-xs text-muted-foreground">前缀</p><p className="mt-1 font-medium">{formatNumber(item.prefix_count)}</p></div><div><p className="text-xs text-muted-foreground">IPv4 / IPv6</p><p className="mt-1">{item.ipv4_count} / {item.ipv6_count}</p></div></div>
            <div className="mt-4 flex items-center justify-between"><span className="text-xs text-muted-foreground">{formatDate(item.last_synced_at)}</span>{canManage ? <Button variant="outline" size="sm" onClick={() => sync.mutate(item.asn)} disabled={sync.isPending}><RefreshCw />同步</Button> : null}</div>
          </Card>
        )}
      />
      <AlertDialog open={canManage && Boolean(deleting)} onOpenChange={(open) => { if (!open) setDeleting(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>删除 AS{deleting?.asn}？</AlertDialogTitle><AlertDialogDescription>该 ASN 的前缀将不再参与扫描采样。已有扫描结果不会被删除。</AlertDialogDescription></AlertDialogHeader>
          <AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction onClick={() => { if (deleting) remove.mutate(deleting.asn); setDeleting(null) }} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">确认删除</AlertDialogAction></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
