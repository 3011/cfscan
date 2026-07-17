import { useMemo, useState } from 'react'
import type { ColumnDef, Table as TanStackTable } from '@tanstack/react-table'
import { KeyRound, MoreHorizontal, Pencil, Trash2 } from 'lucide-react'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { DataTable } from '@/components/shared/data-table/data-table'
import { DataTableColumnHeader } from '@/components/shared/data-table/data-table-column-header'
import { DataTableViewOptions } from '@/components/shared/data-table/data-table-view-options'
import { DataTableResetButton } from '@/components/shared/data-table/data-table-reset-button'
import { LiveRelativeTime } from '@/components/shared/live-relative-time'
import { StatusBadge } from '@/components/shared/status-badge'
import { SearchInput } from '@/components/shared/search-input'
import { useAuth } from '@/features/auth/auth-context'
import { ResetPasswordDialog } from '@/features/users/components/reset-password-dialog'
import { UserDialog } from '@/features/users/components/user-dialog'
import { useDeleteUser } from '@/features/users/hooks'
import type { User } from '@/features/users/types'
import { formatDate } from '@/lib/format'

function UsersToolbar({ table }: { table: TanStackTable<User> }) {
  const role = String(table.getColumn('role')?.getFilterValue() ?? 'all')
  const filtered = Boolean(table.getColumn('search')?.getFilterValue() || role !== 'all')
  const reset = () => {
    table.getColumn('search')?.setFilterValue('')
    table.getColumn('role')?.setFilterValue(undefined)
  }

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <SearchInput
        className="flex-1 sm:max-w-sm"
        placeholder="搜索用户名或显示名称"
        value={String(table.getColumn('search')?.getFilterValue() ?? '')}
        onChange={(event) => table.getColumn('search')?.setFilterValue(event.target.value)}
      />
      <Select
        items={{ all: '全部权限', admin: '管理员', viewer: '查看者' }}
        value={role}
        onValueChange={(value) => table.getColumn('role')?.setFilterValue(value === 'all' ? undefined : value)}
      >
        <SelectTrigger className="w-full sm:w-36" aria-label="账号权限"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部权限</SelectItem>
          <SelectItem value="admin">管理员</SelectItem>
          <SelectItem value="viewer">查看者</SelectItem>
        </SelectContent>
      </Select>
      <DataTableResetButton disabled={!filtered} onClick={reset} />
      <DataTableViewOptions table={table} />
    </div>
  )
}

export function UsersTable({ users }: { users: User[] }) {
  const auth = useAuth()
  const remove = useDeleteUser()
  const [editing, setEditing] = useState<User | null>(null)
  const [resetting, setResetting] = useState<User | null>(null)
  const [deleting, setDeleting] = useState<User | null>(null)
  const columns = useMemo<ColumnDef<User>[]>(() => [
    { id: 'search', accessorFn: (row) => `${row.username} ${row.display_name}`, filterFn: 'includesString', enableHiding: false, header: () => null, cell: () => null },
    { accessorKey: 'username', meta: { label: '账号' }, header: ({ column }) => <DataTableColumnHeader column={column} title="账号" />, cell: ({ row }) => <div><p className="font-mono font-medium">{row.original.username}</p><p className="mt-1 text-xs text-muted-foreground">{row.original.display_name}</p></div> },
    { accessorKey: 'role', meta: { label: '权限' }, header: '权限', filterFn: 'equalsString', cell: ({ row }) => <Badge variant={row.original.role === 'admin' ? 'default' : 'secondary'}>{row.original.role === 'admin' ? '管理员' : '查看者'}</Badge> },
    { accessorKey: 'enabled', meta: { label: '状态' }, header: '状态', cell: ({ row }) => <StatusBadge status={row.original.enabled ? 'enabled' : 'disabled'} /> },
    { accessorKey: 'last_login_at', meta: { label: '最后登录' }, header: ({ column }) => <DataTableColumnHeader column={column} title="最后登录" />, cell: ({ row }) => row.original.last_login_at ? <div><p><LiveRelativeTime value={row.original.last_login_at} /></p><p className="text-xs text-muted-foreground">{formatDate(row.original.last_login_at)}</p></div> : <span className="text-muted-foreground">从未登录</span> },
    { accessorKey: 'created_at', meta: { label: '创建时间' }, header: ({ column }) => <DataTableColumnHeader column={column} title="创建时间" />, cell: ({ row }) => <span className="text-muted-foreground">{formatDate(row.original.created_at)}</span> },
    { id: 'actions', enableHiding: false, header: () => <span className="sr-only">操作</span>, cell: ({ row }) => <DropdownMenu><DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="size-8" aria-label={`操作 ${row.original.username}`} />}><MoreHorizontal /></DropdownMenuTrigger><DropdownMenuContent align="end"><DropdownMenuLabel>账号操作</DropdownMenuLabel><DropdownMenuSeparator /><DropdownMenuItem onClick={() => setEditing(row.original)}><Pencil />编辑账号</DropdownMenuItem><DropdownMenuItem onClick={() => setResetting(row.original)}><KeyRound />重置密码</DropdownMenuItem><DropdownMenuSeparator /><DropdownMenuItem disabled={row.original.id === auth.user?.id} className="text-destructive focus:text-destructive" onClick={() => setDeleting(row.original)}><Trash2 />删除账号</DropdownMenuItem></DropdownMenuContent></DropdownMenu> },
  ], [auth.user?.id])

  return <><DataTable columns={columns} data={users} getRowId={(user) => user.id} initialColumnVisibility={{ search: false }} emptyTitle="暂无账号" emptyDescription="创建管理员或查看者账号后，用户即可登录平台。" renderToolbar={(table) => <UsersToolbar table={table} />} renderMobileItem={(user) => <Card size="sm" className="gap-0 p-4 py-4"><div className="flex items-start justify-between gap-3"><div><p className="font-mono font-medium">{user.username}</p><p className="mt-1 text-sm text-muted-foreground">{user.display_name}</p></div><StatusBadge status={user.enabled ? 'enabled' : 'disabled'} /></div><div className="mt-4 flex items-center justify-between"><Badge variant={user.role === 'admin' ? 'default' : 'secondary'}>{user.role === 'admin' ? '管理员' : '查看者'}</Badge><DropdownMenu><DropdownMenuTrigger render={<Button variant="outline" size="sm" />}><MoreHorizontal />操作</DropdownMenuTrigger><DropdownMenuContent align="end"><DropdownMenuItem onClick={() => setEditing(user)}><Pencil />编辑账号</DropdownMenuItem><DropdownMenuItem onClick={() => setResetting(user)}><KeyRound />重置密码</DropdownMenuItem><DropdownMenuSeparator /><DropdownMenuItem disabled={user.id === auth.user?.id} className="text-destructive focus:text-destructive" onClick={() => setDeleting(user)}><Trash2 />删除账号</DropdownMenuItem></DropdownMenuContent></DropdownMenu></div></Card>} /><UserDialog open={Boolean(editing)} onOpenChange={(open) => { if (!open) setEditing(null) }} user={editing} /><ResetPasswordDialog open={Boolean(resetting)} onOpenChange={(open) => { if (!open) setResetting(null) }} user={resetting} /><AlertDialog open={Boolean(deleting)} onOpenChange={(open) => { if (!open) setDeleting(null) }}><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>删除账号 {deleting?.username}？</AlertDialogTitle><AlertDialogDescription>该账号的所有会话会立即失效。此操作不可撤销。</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" disabled={remove.isPending} onClick={async () => { if (!deleting) return; await remove.mutateAsync(deleting.id); setDeleting(null) }}>确认删除</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog></>
}
