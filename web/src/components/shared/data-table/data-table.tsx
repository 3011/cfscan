import { useState, type ReactNode } from 'react'
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type ColumnFiltersState,
  type PaginationState,
  type SortingState,
  type Updater,
  type VisibilityState,
  type Table as TanStackTable,
} from '@tanstack/react-table'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { EmptyState } from '@/components/shared/empty-state'

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[]
  data: TData[]
  emptyTitle: string
  emptyDescription: string
  renderToolbar?: (table: TanStackTable<TData>) => ReactNode
  renderMobileItem?: (item: TData) => ReactNode
  initialPageSize?: number
  getRowId?: (row: TData) => string
  initialColumnVisibility?: VisibilityState
  serverPagination?: {
    pageIndex: number
    pageSize: number
    pageCount: number
    total: number
    onChange: (pagination: PaginationState) => void
  }
  serverSorting?: {
    sorting: SortingState
    onChange: (sorting: SortingState) => void
  }
}

export function DataTable<TData, TValue>({
  columns,
  data,
  emptyTitle,
  emptyDescription,
  renderToolbar,
  renderMobileItem,
  initialPageSize = 20,
  getRowId,
  initialColumnVisibility,
  serverPagination,
  serverSorting,
}: DataTableProps<TData, TValue>) {
  const [localSorting, setLocalSorting] = useState<SortingState>([])
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(() => initialColumnVisibility ?? {})
  const sorting = serverSorting?.sorting ?? localSorting
  const pagination = serverPagination ? { pageIndex: serverPagination.pageIndex, pageSize: serverPagination.pageSize } : undefined
  const updateSorting = (updater: Updater<SortingState>) => {
    const next = typeof updater === 'function' ? updater(sorting) : updater
    if (serverSorting) serverSorting.onChange(next)
    else setLocalSorting(next)
  }
  const updatePagination = (updater: Updater<PaginationState>) => {
    if (!serverPagination) return
    const next = typeof updater === 'function' ? updater(pagination!) : updater
    serverPagination.onChange(next)
  }
  const table = useReactTable({
    data,
    columns,
    getRowId,
    state: { sorting, columnFilters, columnVisibility, ...(pagination ? { pagination } : {}) },
    onSortingChange: updateSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: serverPagination ? updatePagination : undefined,
    manualPagination: Boolean(serverPagination),
    pageCount: serverPagination?.pageCount,
    manualSorting: Boolean(serverSorting),
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: serverSorting ? undefined : getSortedRowModel(),
    getPaginationRowModel: serverPagination ? undefined : getPaginationRowModel(),
    initialState: { pagination: { pageSize: initialPageSize } },
  })

  const rows = table.getRowModel().rows
  return (
    <div className="space-y-3">
      {renderToolbar ? renderToolbar(table) : null}
      {renderMobileItem ? (
        <div className="grid gap-3 md:hidden">
          {rows.length ? rows.map((row) => <div key={row.id}>{renderMobileItem(row.original)}</div>) : <EmptyState title={emptyTitle} description={emptyDescription} />}
        </div>
      ) : null}
      <div className={renderMobileItem ? 'hidden overflow-hidden rounded-3xl bg-card shadow-sm ring-1 ring-foreground/5 md:block dark:ring-foreground/10' : 'overflow-hidden rounded-3xl bg-card shadow-sm ring-1 ring-foreground/5 dark:ring-foreground/10'}>
        <Table className="[&_th:first-child]:pl-4 [&_th:last-child]:pr-4 [&_td:first-child]:pl-4 [&_td:last-child]:pr-4">
          <TableHeader className="bg-muted/35">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {rows.length ? rows.map((row) => (
              <TableRow key={row.id} data-state={row.getIsSelected() ? 'selected' : undefined}>
                {row.getVisibleCells().map((cell) => (
                  <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>
                ))}
              </TableRow>
            )) : (
              <TableRow>
                <TableCell colSpan={table.getVisibleLeafColumns().length} className="p-0">
                  <EmptyState title={emptyTitle} description={emptyDescription} />
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      {serverPagination || data.length > initialPageSize ? (
        <div className="flex flex-col gap-3 px-1 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">
            {serverPagination
              ? `共 ${serverPagination.total.toLocaleString('zh-CN')} 条${serverPagination.total ? `，当前 ${(serverPagination.pageIndex * serverPagination.pageSize + 1).toLocaleString('zh-CN')}–${Math.min((serverPagination.pageIndex + 1) * serverPagination.pageSize, serverPagination.total).toLocaleString('zh-CN')} 条` : ''}`
              : `共 ${table.getFilteredRowModel().rows.length.toLocaleString('zh-CN')} 条`}
          </p>
          <div className="flex items-center gap-2">
            <Select
              items={Object.fromEntries((serverPagination ? [50, 100, 200] : [10, 20, 50, 100]).map((size) => [String(size), `${size} 条`]))}
              value={String(table.getState().pagination.pageSize)}
              onValueChange={(value) => table.setPageSize(Number(value))}
            >
              <SelectTrigger className="h-8 w-[92px]" aria-label="每页条数"><SelectValue /></SelectTrigger>
              <SelectContent>
                {(serverPagination ? [50, 100, 200] : [10, 20, 50, 100]).map((size) => <SelectItem key={size} value={String(size)}>{size} 条</SelectItem>)}
              </SelectContent>
            </Select>
            <span className="min-w-20 text-center text-sm text-muted-foreground">
              {table.getState().pagination.pageIndex + 1} / {Math.max(table.getPageCount(), 1)}
            </span>
            <Button variant="outline" size="icon" className="size-8" onClick={() => table.previousPage()} disabled={!table.getCanPreviousPage()} aria-label="上一页">
              <ChevronLeft />
            </Button>
            <Button variant="outline" size="icon" className="size-8" onClick={() => table.nextPage()} disabled={!table.getCanNextPage()} aria-label="下一页">
              <ChevronRight />
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
