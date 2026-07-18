import { Bot, Cpu, Globe2, RadioTower } from 'lucide-react'
import { ErrorState } from '@/components/shared/error-state'
import { MetricCard } from '@/components/shared/metric-card'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { PermissionGate } from '@/components/shared/permission-gate'
import { AddAgentDialog } from '@/features/agents/components/add-agent-dialog'
import { AgentsTable } from '@/features/agents/components/agents-table'
import { PendingEnrollmentsCard } from '@/features/agents/components/pending-enrollments-card'
import { useAgentEnrollments, useAgents } from '@/features/agents/hooks'
import { useAuth } from '@/features/auth/auth-context'

export function AgentsPage() {
  const auth = useAuth()
  const agents = useAgents({ refetchInterval: 5_000 })
  const enrollments = useAgentEnrollments({ refetchInterval: 3_000 })
  if (agents.isPending || enrollments.isPending) return <PageSkeleton rows={6} />
  if (agents.isError) return <ErrorState error={agents.error} onRetry={() => agents.refetch()} />
  if (enrollments.isError) return <ErrorState title="配对请求加载失败" error={enrollments.error} onRetry={() => enrollments.refetch()} />
  const items = agents.data.items
  const activeEnrollments = enrollments.data.items.filter((item) => item.status === 'pending' || item.status === 'approved')
  const online = items.filter((agent) => agent.status === 'online')
  const regions = new Set(items.map((agent) => `${agent.continent}/${agent.region}`)).size
  const concurrency = online.reduce((total, agent) => total + agent.concurrency, 0)
  return (
    <div className="page-grid">
      <PageHeader
        title="Agent 节点"
        description="管理分布式扫描节点、连接状态和运行能力。"
        actions={<PermissionGate allowed={auth.canManage}><AddAgentDialog /></PermissionGate>}
      />
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard icon={Bot} title="节点总数" value={items.length} description="已完成连接的 Agent" />
        <MetricCard icon={RadioTower} title="在线节点" value={`${online.length} / ${items.length}`} description="最近 45 秒内有心跳" />
        <MetricCard icon={Globe2} title="覆盖地区" value={regions} description="按大洲和地区去重" />
        <MetricCard icon={Cpu} title="在线并发容量" value={concurrency} description={activeEnrollments.length ? `另有 ${activeEnrollments.length} 个连接请求进行中` : '所有在线 Agent 配置之和'} />
      </section>
      <PendingEnrollmentsCard enrollments={activeEnrollments} />
      <AgentsTable agents={items} />
    </div>
  )
}
