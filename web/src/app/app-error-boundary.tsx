import { ErrorBoundary } from 'react-error-boundary'
import type { PropsWithChildren } from 'react'
import { AlertTriangle, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export function AppErrorBoundary({ children }: PropsWithChildren) {
  return (
    <ErrorBoundary
      fallbackRender={({ error, resetErrorBoundary }) => (
        <main className="grid min-h-svh place-items-center bg-muted/30 p-6">
          <Card className="w-full max-w-lg">
            <CardHeader>
              <div className="mb-2 flex size-10 items-center justify-center rounded-full bg-destructive/10 text-destructive">
                <AlertTriangle className="size-5" />
              </div>
              <CardTitle>页面发生错误</CardTitle>
              <CardDescription>前端没有正确完成本次渲染。可以重试，或刷新页面重新加载数据。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <pre className="max-h-36 overflow-auto rounded-md bg-muted p-3 text-xs text-muted-foreground">
                {error instanceof Error ? error.message : String(error)}
              </pre>
              <Button onClick={resetErrorBoundary}>
                <RotateCcw />
                重新加载页面
              </Button>
            </CardContent>
          </Card>
        </main>
      )}
    >
      {children}
    </ErrorBoundary>
  )
}
