import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

const labels: Record<string, string> = {
  completed: '已完成',
  running: '运行中',
  pending: '等待中',
  online: '在线',
  offline: '离线',
  ok: '正常',
  error: '异常',
  never: '未同步',
  failed: '失败',
  skipped: '已跳过',
  stopped: '已停止',
  scheduled: '定时扫描',
  blacklist_recheck: '复检',
  normal: '常规扫描',
  enabled: '已启用',
  disabled: '已停用',
}

export function StatusBadge({ status }: { status: string }) {
  const key = status.toLowerCase()
  const dot = key === 'online' || key === 'ok' || key === 'completed' || key === 'enabled'
    ? 'bg-emerald-500'
    : key === 'running' || key === 'pending' || key === 'skipped'
      ? 'bg-amber-500'
      : key === 'error' || key === 'offline' || key === 'failed' || key === 'stopped' || key === 'disabled'
        ? 'bg-destructive'
        : 'bg-muted-foreground'
  return (
    <Badge variant="outline" className="gap-1.5 font-normal">
      <span className={cn('size-1.5 rounded-full', dot)} aria-hidden="true" />
      {labels[key] ?? status}
    </Badge>
  )
}
