import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { ErrorState } from '@/components/shared/error-state'
import { CreateScanSheet } from '@/features/scans/components/create-scan-sheet'
import { ScanJobsTable } from '@/features/scans/components/scan-jobs-table'
import { useScanJobs } from '@/features/scans/hooks'
import { useAuth } from '@/features/auth/auth-context'

export function ScansPage() {
  const jobs = useScanJobs(100)
  const auth = useAuth()
  if (jobs.isPending) return <PageSkeleton rows={8} />
  return (
    <div className="page-grid">
      <PageHeader
        title="扫描任务"
        description="从已启用的官方与 ASN 前缀中采样目标，将同一批 IP 分发给不同地区 Agent，并持续跟踪进度和筛选结果。"
        actions={auth.canManage ? <CreateScanSheet /> : undefined}
      />
      {jobs.isError ? <ErrorState error={jobs.error} onRetry={() => jobs.refetch()} /> : <ScanJobsTable jobs={jobs.data.items} />}
    </div>
  )
}
