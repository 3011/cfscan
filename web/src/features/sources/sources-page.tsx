import { Database, Network, RefreshCw, Route, ShieldCheck } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { ErrorState } from '@/components/shared/error-state'
import { MetricCard } from '@/components/shared/metric-card'
import { StatusBadge } from '@/components/shared/status-badge'
import { ASNTable } from '@/features/sources/components/asn-table'
import { CreateASNDialog } from '@/features/sources/components/create-asn-dialog'
import { useASNSources, useOfficialSource, useSyncAllASNs, useSyncOfficial } from '@/features/sources/hooks'
import { formatDate, formatNumber } from '@/lib/format'
import { useAuth } from '@/features/auth/auth-context'

export function SourcesPage() {
  const auth = useAuth()
  const official = useOfficialSource()
  const asns = useASNSources()
  const syncOfficial = useSyncOfficial()
  const syncASNs = useSyncAllASNs()
  const syncing = syncOfficial.isPending || syncASNs.isPending

  if (official.isPending || asns.isPending) return <PageSkeleton rows={8} />
  const items = asns.data?.items ?? []
  const enabled = items.filter((item) => item.enabled)
  const totals = enabled.reduce((sum, item) => ({ prefixes: sum.prefixes + item.prefix_count, ipv4: sum.ipv4 + item.ipv4_count, ipv6: sum.ipv6 + item.ipv6_count }), { prefixes: 0, ipv4: 0, ipv6: 0 })

  return (
    <div className="page-grid">
      <PageHeader
        title="IP 数据源"
        description="统一管理 Cloudflare 官方代理地址段和已确认的 ASN 宣告前缀。任务创建时使用所有启用来源的去重并集。"
        actions={auth.canManage ? (
          <Button disabled={syncing} onClick={async () => { await Promise.all([syncOfficial.mutateAsync(), syncASNs.mutateAsync()]) }}>
            <RefreshCw className={syncing ? 'animate-spin' : ''} />同步全部数据源
          </Button>
        ) : undefined}
      />

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard icon={Database} title="官方前缀" value={formatNumber(official.data?.prefix_count)} description="Cloudflare 明确公布的代理地址段" />
        <MetricCard icon={Route} title="启用 ASN 前缀" value={formatNumber(totals.prefixes)} description={`${enabled.length} 个启用 ASN 的原始前缀`} />
        <MetricCard icon={Network} title="ASN IPv4 / IPv6" value={`${formatNumber(totals.ipv4)} / ${formatNumber(totals.ipv6)}`} description="IPv6 按 CIDR 保存并在任务中采样" />
        <MetricCard icon={ShieldCheck} title="启用 ASN" value={`${enabled.length} / ${items.length}`} description="内置 ASN 可停用但不可误删" />
      </section>

      {official.isError ? <ErrorState title="官方数据源加载失败" error={official.error} onRetry={() => official.refetch()} /> : null}
      {asns.isError ? <ErrorState title="ASN 数据源加载失败" error={asns.error} onRetry={() => asns.refetch()} /> : null}

      <Tabs defaultValue="asns" className="space-y-4">
        <TabsList variant="line">
          <TabsTrigger value="asns">ASN 数据源</TabsTrigger>
          <TabsTrigger value="official">官方地址段</TabsTrigger>
        </TabsList>
        <TabsContent value="asns" className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div><h2 className="text-lg font-semibold">Cloudflare ASN</h2><p className="text-sm text-muted-foreground">按 ASN 同步当前 BGP 宣告，并独立控制是否参与扫描。</p></div>
            {auth.canManage ? <CreateASNDialog /> : null}
          </div>
          {!asns.isError ? <ASNTable items={items} canManage={auth.canManage} /> : null}
        </TabsContent>
        <TabsContent value="official">
          <Card>
            <CardHeader className="flex flex-row items-start justify-between gap-4">
              <div><CardTitle>Cloudflare 官方代理前缀</CardTitle><CardDescription>该来源与 ASN 前缀同时保留；扫描池按 CIDR 去重，不会重复采样同一前缀。</CardDescription></div>
              {auth.canManage ? <Button variant="outline" onClick={() => syncOfficial.mutate()} disabled={syncOfficial.isPending}><RefreshCw className={syncOfficial.isPending ? 'animate-spin' : ''} />同步官方段</Button> : null}
            </CardHeader>
            <CardContent className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
              <div><p className="text-sm text-muted-foreground">同步状态</p><div className="mt-2"><StatusBadge status={official.data?.status ?? 'never'} /></div></div>
              <div><p className="text-sm text-muted-foreground">IPv4 / IPv6</p><p className="metric-value mt-2 text-lg font-medium">{official.data?.ipv4_count ?? 0} / {official.data?.ipv6_count ?? 0}</p></div>
              <div><p className="text-sm text-muted-foreground">最后同步</p><p className="mt-2 text-sm font-medium">{formatDate(official.data?.last_synced_at)}</p></div>
              <div><p className="text-sm text-muted-foreground">自动同步</p><p className="mt-2 text-sm font-medium">中心启动时，之后每 6 小时</p></div>
              {official.data?.last_error ? <div className="sm:col-span-2 lg:col-span-4"><ErrorState title="最近一次同步失败" error={new Error(official.data.last_error)} /></div> : null}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
