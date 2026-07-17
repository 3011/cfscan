import { RefreshCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { ErrorState } from '@/components/shared/error-state'
import { BlacklistTable } from '@/features/blacklist/components/blacklist-table'
import { useBlacklist, useRecheckBlacklist } from '@/features/blacklist/hooks'
import { useAuth } from '@/features/auth/auth-context'

export function BlacklistPage() {
  const auth = useAuth()
  const blacklist = useBlacklist()
  const recheck = useRecheckBlacklist()
  if (blacklist.isPending) return <PageSkeleton rows={8} />
  return (
    <div className="page-grid">
      <PageHeader
        title="黑名单"
        description="黑名单按 Agent 维度隔离。高延迟、高丢包或超时目标会暂时过滤，到期后的自动复查范围、比例和阈值由“设置 → 自动化 → 黑名单复查”统一管理。"
        actions={auth.canManage ? (
          <AlertDialog>
            <AlertDialogTrigger render={<Button variant="outline" />}><RefreshCcw />按当前策略立即复查</AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader><AlertDialogTitle>安排黑名单复检？</AlertDialogTitle><AlertDialogDescription>系统会使用“设置 → 自动化 → 黑名单复查”中保存的候选范围、比例、上限和探测阈值创建任务。</AlertDialogDescription></AlertDialogHeader>
              <AlertDialogFooter><AlertDialogCancel>取消</AlertDialogCancel><AlertDialogAction disabled={recheck.isPending} onClick={() => recheck.mutate()}>确认安排</AlertDialogAction></AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        ) : undefined}
      />
      {blacklist.isError ? <ErrorState error={blacklist.error} onRetry={() => blacklist.refetch()} /> : <BlacklistTable items={blacklist.data.items} />}
    </div>
  )
}
