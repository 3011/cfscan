import { Skeleton } from '@/components/ui/skeleton'

export function PageSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div className="space-y-5" aria-label="正在加载">
      <div className="space-y-2">
        <Skeleton className="h-8 w-48 rounded-2xl" />
        <Skeleton className="h-4 w-full max-w-xl rounded-xl" />
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => <Skeleton key={index} className="h-28 rounded-3xl" />)}
      </div>
      <div className="rounded-3xl bg-card p-4 shadow-sm ring-1 ring-foreground/5 dark:ring-foreground/10">
        <Skeleton className="mb-4 h-8 w-full max-w-sm rounded-2xl" />
        <div className="space-y-2">
          {Array.from({ length: rows }).map((_, index) => <Skeleton key={index} className="h-9 w-full rounded-xl" />)}
        </div>
      </div>
    </div>
  )
}
