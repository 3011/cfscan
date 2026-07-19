import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { DatabaseZap, Loader2, Pencil, Play } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Switch } from '@/components/ui/switch'
import { ErrorState } from '@/components/shared/error-state'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { TimezoneCombobox } from '@/features/settings/components/timezone-combobox'
import { sourceSyncSchema, type SourceSyncValues } from '@/features/settings/schema'
import { useRunSourceSyncSchedule, useSourceSyncSchedules, useUpdateSourceSyncSchedule } from '@/features/settings/hooks'
import type { SourceSyncSchedule } from '@/features/settings/types'
import { formatDateInTimezone } from '@/lib/format'

export function SourceSyncPanel({ canManage }: { canManage: boolean }) {
  const schedules = useSourceSyncSchedules()
  const update = useUpdateSourceSyncSchedule()
  const run = useRunSourceSyncSchedule()
  const [editing, setEditing] = useState<SourceSyncSchedule | null>(null)
  const form = useForm<SourceSyncValues>({ resolver: zodResolver(sourceSyncSchema) })
  useEffect(() => { if (editing) form.reset(editing) }, [editing, form])
  if (schedules.isPending) return <PageSkeleton rows={5} />
  if (schedules.isError) return <ErrorState error={schedules.error} onRetry={() => schedules.refetch()} />

  async function submit(values: SourceSyncValues) {
    if (!editing) return
    await update.mutateAsync({ source: editing.source, input: values })
    setEditing(null)
  }

  return (
    <>
      <div className="grid gap-4 lg:grid-cols-3">
        {schedules.data.items.map((item) => (
          <Card key={item.source}>
            <CardHeader className="gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div><CardTitle className="flex items-center gap-2"><DatabaseZap className="size-4" />{item.name}</CardTitle><CardDescription className="mt-1.5">{item.source === 'official' ? '同步 Cloudflare 官方公布的代理地址段。' : item.source === 'asn' ? '同步所有已启用 Cloudflare ASN 当前宣告的 BGP 前缀。' : '同步 Cloudflare 官方状态页公布的 colo 城市、国家和大洲信息。'}</CardDescription></div>
              <Badge variant={item.enabled ? 'default' : 'secondary'}>{item.enabled ? '已启用' : '已停用'}</Badge>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div><p className="text-xs text-muted-foreground">执行频率</p><p className="mt-1 font-mono">{item.cron_expression}</p></div>
                <div><p className="text-xs text-muted-foreground">时区</p><p className="mt-1">{item.timezone}</p></div>
                <div><p className="text-xs text-muted-foreground">下次执行</p><p className="mt-1">{formatDateInTimezone(item.next_run_at, item.timezone)}</p></div>
                <div><p className="text-xs text-muted-foreground">中心启动</p><p className="mt-1">{item.run_on_startup ? '立即同步' : '等待计划'}</p></div>
              </div>
              {item.last_error ? <p className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{item.last_error}</p> : null}
              {canManage ? <div className="flex gap-2"><Button variant="outline" className="flex-1" onClick={() => run.mutate(item.source)} disabled={run.isPending}><Play />立即同步</Button><Button className="flex-1" onClick={() => setEditing(item)}><Pencil />编辑</Button></div> : null}
            </CardContent>
          </Card>
        ))}
      </div>

      <Dialog open={canManage && Boolean(editing)} onOpenChange={(open) => { if (!open) setEditing(null) }}>
        <DialogContent className="flex max-h-[calc(100dvh-2rem)] w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl">
          <DialogHeader className="shrink-0 border-b px-6 py-5 pr-12"><DialogTitle>编辑数据源同步</DialogTitle><DialogDescription>{editing?.name}</DialogDescription></DialogHeader>
          <Form {...form}><form onSubmit={form.handleSubmit(submit)} className="flex min-h-0 flex-1 flex-col"><div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-6 py-6">
            <FormField control={form.control} name="enabled" render={({ field }) => <FormItem className="flex items-center justify-between rounded-lg border p-4"><div><FormLabel>启用定时同步</FormLabel><FormDescription>停用后仍可手动立即同步。</FormDescription></div><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></FormItem>} />
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField control={form.control} name="cron_expression" render={({ field }) => <FormItem><FormLabel>Cron 表达式</FormLabel><FormControl><Input className="font-mono" {...field} /></FormControl><FormDescription>标准 5 段 Cron。</FormDescription><FormMessage /></FormItem>} />
              <FormField control={form.control} name="timezone" render={({ field }) => <FormItem><FormLabel>时区</FormLabel><FormControl><TimezoneCombobox value={field.value} onValueChange={field.onChange} /></FormControl><FormMessage /></FormItem>} />
            </div>
            <FormField control={form.control} name="run_on_startup" render={({ field }) => <FormItem className="flex items-center justify-between rounded-lg border p-4"><div><FormLabel>中心启动时立即同步</FormLabel><FormDescription>每次中心服务重启后执行一次，并重新计算下次时间。</FormDescription></div><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></FormItem>} />
            </div><DialogFooter className="shrink-0 border-t bg-background px-6 py-4 pb-[max(1rem,env(safe-area-inset-bottom))]"><Button type="button" variant="outline" onClick={() => setEditing(null)}>取消</Button><Button type="submit" disabled={update.isPending}>{update.isPending ? <Loader2 className="animate-spin" /> : null}保存设置</Button></DialogFooter>
          </form></Form>
        </DialogContent>
      </Dialog>
    </>
  )
}
