import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

interface MetricCardProps {
  title: string
  value: ReactNode
  description: string
  icon: LucideIcon
  trend?: ReactNode
}

export function MetricCard({ title, value, description, icon: Icon, trend }: MetricCardProps) {
  return (
    <Card size="sm" className="min-h-28">
      <CardHeader className="grid grid-cols-[1fr_auto] items-start gap-3">
        <CardTitle className="text-sm text-muted-foreground">{title}</CardTitle>
        <span className="flex size-7 items-center justify-center rounded-xl bg-muted text-muted-foreground"><Icon className="size-3.5" /></span>
      </CardHeader>
      <CardContent className="mt-auto">
        <div className="metric-value font-heading text-2xl font-medium tracking-tight">{value}</div>
        <CardDescription className="mt-1 flex items-center gap-2 text-xs">
          {trend}
          <span>{description}</span>
        </CardDescription>
      </CardContent>
    </Card>
  )
}
