import { AlertCircle, RefreshCw } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

interface ErrorStateProps {
  title?: string
  error: unknown
  onRetry?: () => void
}

export function ErrorState({ title = '数据加载失败', error, onRetry }: ErrorStateProps) {
  const message = error instanceof Error ? error.message : '发生未知错误'
  return (
    <Alert variant="destructive">
      <AlertCircle className="size-4" />
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <span>{message}</span>
        {onRetry ? (
          <Button variant="outline" size="sm" onClick={onRetry} className="w-fit border-destructive/30 bg-background text-foreground">
            <RefreshCw />重试
          </Button>
        ) : null}
      </AlertDescription>
    </Alert>
  )
}
