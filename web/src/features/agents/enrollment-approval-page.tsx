import { useEffect, useMemo } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Link, useParams } from 'react-router-dom'
import { CheckCircle2, Clock3, Loader2, Laptop, ShieldCheck, XCircle } from 'lucide-react'
import { z } from 'zod'
import { ErrorState } from '@/components/shared/error-state'
import { LiveRelativeTime } from '@/components/shared/live-relative-time'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useAgentEnrollment, useApproveAgentEnrollment, useRejectAgentEnrollment } from '@/features/agents/hooks'
import { useAuth } from '@/features/auth/auth-context'
import type { EnrollmentLocator } from '@/features/agents/types'

const continents = ['Africa', 'Asia', 'Europe', 'North America', 'Oceania', 'South America'] as const
const approvalSchema = z.object({
  name: z.string().trim().min(1, '请输入节点名称').max(128),
  region: z.string().trim().min(1, '请输入地区').max(128),
  continent: z.string().min(1, '请选择大洲'),
  concurrency: z.number().int().min(1).max(4096),
})
type ApprovalValues = z.infer<typeof approvalSchema>

function DeviceDetails({ enrollment }: { enrollment: ReturnType<typeof useAgentEnrollment>['data'] }) {
  if (!enrollment) return null
  const details = [
    ['请求名称', enrollment.requested_name || '未提供'],
    ['操作系统', enrollment.os || '未提供'],
    ['架构', enrollment.architecture || '未提供'],
    ['Agent 版本', enrollment.version || '未提供'],
    ['请求并发数', String(enrollment.requested_concurrency)],
  ]
  return (
    <div className="rounded-2xl border bg-muted/25 p-4">
      <div className="mb-3 flex items-center gap-2 font-medium"><Laptop />请求设备</div>
      <dl className="grid gap-2 text-sm sm:grid-cols-2">
        {details.map(([label, value]) => <div key={label} className="flex items-center justify-between gap-4 sm:block"><dt className="text-xs text-muted-foreground">{label}</dt><dd className="mt-0.5 truncate">{value}</dd></div>)}
      </dl>
      <p className="mt-3 flex items-center gap-1.5 border-t pt-3 text-xs text-muted-foreground"><Clock3 />配对请求将在 <LiveRelativeTime value={enrollment.expires_at} />到期</p>
    </div>
  )
}

export function EnrollmentApprovalPage() {
  const params = useParams<{ pairingToken?: string; enrollmentID?: string }>()
  const locator = useMemo<EnrollmentLocator>(() => params.enrollmentID
    ? { kind: 'id', value: params.enrollmentID }
    : { kind: 'token', value: params.pairingToken ?? '' }, [params.enrollmentID, params.pairingToken])
  const auth = useAuth()
  const enrollment = useAgentEnrollment(locator)
  const approve = useApproveAgentEnrollment(locator)
  const reject = useRejectAgentEnrollment(locator)
  const form = useForm<ApprovalValues>({ resolver: zodResolver(approvalSchema), defaultValues: { name: '', region: '', continent: 'Asia', concurrency: 64 } })

  useEffect(() => {
    if (!enrollment.data) return
    form.reset({
      name: enrollment.data.name || enrollment.data.requested_name,
      region: enrollment.data.region || '',
      continent: enrollment.data.continent || 'Asia',
      concurrency: enrollment.data.concurrency || enrollment.data.requested_concurrency || 64,
    })
  }, [enrollment.data, form])

  if (enrollment.isPending) return <main className="grid min-h-svh place-items-center bg-muted/30 p-5"><Loader2 className="size-6 animate-spin text-muted-foreground" aria-label="正在加载配对请求" /></main>
  if (enrollment.isError) return <main className="grid min-h-svh place-items-center bg-muted/30 p-5"><div className="w-full max-w-xl"><ErrorState title="配对请求加载失败" error={enrollment.error} onRetry={() => enrollment.refetch()} /></div></main>
  const item = enrollment.data
  if (!item) return null

  let content
  if (item.status === 'approved') {
    content = <div className="py-8 text-center"><Loader2 className="mx-auto size-8 animate-spin text-primary" /><h2 className="mt-4 font-heading text-lg font-medium">已批准，正在等待 Agent 完成连接</h2><p className="mt-2 text-sm text-muted-foreground">请保持 Agent 进程运行。连接成功后此页面会自动更新。</p></div>
  } else if (item.status === 'claimed') {
    content = <div className="py-8 text-center"><CheckCircle2 className="mx-auto size-10 text-primary" /><h2 className="mt-4 font-heading text-lg font-medium">Agent 已成功连接</h2><p className="mt-2 text-sm text-muted-foreground">{item.name} 已获得独立身份并开始发送心跳。</p><Button className="mt-5" nativeButton={false} render={<Link to="/agents" />}>查看 Agent</Button></div>
  } else if (item.status === 'rejected') {
    content = <div className="py-8 text-center"><XCircle className="mx-auto size-10 text-destructive" /><h2 className="mt-4 font-heading text-lg font-medium">此连接请求已被拒绝</h2><p className="mt-2 text-sm text-muted-foreground">需要连接时，请在 Agent 机器重新运行 connect 命令。</p></div>
  } else if (item.status === 'expired' || item.status === 'revoked') {
    content = <div className="py-8 text-center"><Clock3 className="mx-auto size-10 text-muted-foreground" /><h2 className="mt-4 font-heading text-lg font-medium">此配对请求已失效</h2><p className="mt-2 text-sm text-muted-foreground">请在 Agent 机器重新运行连接命令，生成新的配对请求。</p></div>
  } else if (!auth.canManage) {
    content = <div className="space-y-5"><DeviceDetails enrollment={item} /><div className="rounded-2xl border p-4 text-sm text-muted-foreground">当前账号只有查看权限，请联系管理员批准此 Agent。</div></div>
  } else {
    content = (
      <div className="space-y-5">
        <DeviceDetails enrollment={item} />
        <Form {...form}>
          <form id="approve-agent-form" className="space-y-4" onSubmit={form.handleSubmit((values) => approve.mutateAsync(values))}>
            <FormField control={form.control} name="name" render={({ field }) => <FormItem><FormLabel>节点名称</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>} />
            <FormField control={form.control} name="region" render={({ field }) => <FormItem><FormLabel>地区</FormLabel><FormControl><Input placeholder="Guangzhou" {...field} /></FormControl><FormMessage /></FormItem>} />
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField control={form.control} name="continent" render={({ field }) => <FormItem><FormLabel>大洲</FormLabel><Select items={Object.fromEntries(continents.map((value) => [value, value]))} value={field.value} onValueChange={field.onChange}><FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl><SelectContent>{continents.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectContent></Select><FormMessage /></FormItem>} />
              <FormField control={form.control} name="concurrency" render={({ field }) => <FormItem><FormLabel>最大并发数</FormLabel><FormControl><Input type="number" min={1} max={4096} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} /></FormControl><FormMessage /></FormItem>} />
            </div>
          </form>
        </Form>
      </div>
    )
  }

  return (
    <main className="flex min-h-svh items-center justify-center bg-muted/30 p-4 sm:p-8">
      <Card className="w-full max-w-2xl shadow-lg">
        <CardHeader>
          <div className="mb-2 flex size-10 items-center justify-center rounded-2xl bg-primary text-primary-foreground"><ShieldCheck /></div>
          <CardTitle className="text-xl">批准 Agent 连接</CardTitle>
          <CardDescription>一个新的 Agent 正在请求加入 CF Scanner。长期 Token 不会显示在浏览器中。</CardDescription>
        </CardHeader>
        <CardContent>{content}</CardContent>
        {item.status === 'pending' && auth.canManage ? (
          <CardFooter className="flex-col-reverse gap-2 border-t bg-card sm:flex-row sm:justify-end">
            <Button type="button" variant="outline" className="w-full sm:w-auto" disabled={reject.isPending || approve.isPending} onClick={() => reject.mutate()}>{reject.isPending ? <Loader2 className="animate-spin" /> : null}拒绝</Button>
            <Button type="submit" form="approve-agent-form" className="w-full sm:w-auto" disabled={approve.isPending || reject.isPending}>{approve.isPending ? <Loader2 className="animate-spin" /> : null}批准并连接</Button>
          </CardFooter>
        ) : null}
      </Card>
    </main>
  )
}
