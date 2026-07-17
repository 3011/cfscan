import { Bot, Cpu, Globe2, RadioTower } from 'lucide-react'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { ErrorState } from '@/components/shared/error-state'
import { MetricCard } from '@/components/shared/metric-card'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { AgentsTable } from '@/features/agents/components/agents-table'
import { useAgents } from '@/features/agents/hooks'

export function AgentsPage() {
  const agents = useAgents({ refetchInterval: 5_000 })
  if (agents.isPending) return <PageSkeleton rows={6} />
  if (agents.isError) return <ErrorState error={agents.error} onRetry={() => agents.refetch()} />
  const items = agents.data.items
  const online = items.filter((agent) => agent.status === 'online')
  const regions = new Set(items.map((agent) => `${agent.continent}/${agent.region}`)).size
  const concurrency = online.reduce((total, agent) => total + agent.concurrency, 0)
  return (
    <div className="page-grid">
      <PageHeader title="Agent 节点" description="Agent 运行在不同地区的小服务器上，只主动连接中心、领取任务并批量回传扫描结果。" />
      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard icon={Bot} title="节点总数" value={items.length} description="所有已注册 Agent" />
        <MetricCard icon={RadioTower} title="在线节点" value={`${online.length} / ${items.length}`} description="最近 45 秒内有心跳" />
        <MetricCard icon={Globe2} title="覆盖地区" value={regions} description="按大洲和地区去重" />
        <MetricCard icon={Cpu} title="在线并发容量" value={concurrency} description="所有在线 Agent 配置之和" />
      </section>
      <AgentsTable agents={items} />
      {!items.length ? (
        <Card>
          <CardHeader><CardTitle>部署第一个 Agent</CardTitle><CardDescription>Agent 无需开放入站端口，配置中心地址、Token 和地区信息后启动即可。</CardDescription></CardHeader>
          <CardContent><pre className="overflow-x-auto rounded-md bg-muted p-4 text-xs"><code>{`CFSCAN_CENTER_URL=https://<agent-api-domain>\nCFSCAN_AGENT_TOKEN=<token>\nCFSCAN_AGENT_NAME=hk-01\nCFSCAN_AGENT_CONTINENT=asia\nCFSCAN_AGENT_REGION=hong-kong`}</code></pre></CardContent>
        </Card>
      ) : null}
    </div>
  )
}
