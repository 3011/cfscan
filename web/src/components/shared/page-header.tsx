import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface PageHeaderProps {
  eyebrow?: string
  title: string
  description: string
  actions?: ReactNode
  className?: string
}

export function PageHeader({ eyebrow, title, description, actions, className }: PageHeaderProps) {
  return (
    <div className={cn('flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between', className)}>
      <div className="min-w-0 space-y-1.5">
        {eyebrow ? <p className="text-[11px] font-medium uppercase tracking-[0.16em] text-muted-foreground">{eyebrow}</p> : null}
        <h1 className="font-heading text-2xl font-medium tracking-tight md:text-[1.75rem]">{title}</h1>
        <p className="max-w-3xl text-sm leading-6 text-muted-foreground">{description}</p>
      </div>
      {actions ? <div className="flex shrink-0 flex-wrap items-center gap-2 sm:pt-0.5">{actions}</div> : null}
    </div>
  )
}
