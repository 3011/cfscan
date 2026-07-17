import type { LucideIcon } from 'lucide-react'
import { Inbox } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  title: string
  description: string
  icon?: LucideIcon
  action?: ReactNode
  className?: string
}

export function EmptyState({ title, description, icon: Icon = Inbox, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex min-h-48 flex-col items-center justify-center rounded-3xl px-6 py-10 text-center', className)}>
      <div className="mb-4 flex size-10 items-center justify-center rounded-2xl bg-muted text-muted-foreground">
        <Icon className="size-5" />
      </div>
      <h3 className="font-heading text-sm font-medium">{title}</h3>
      <p className="mt-1 max-w-md text-sm leading-6 text-muted-foreground">{description}</p>
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  )
}
