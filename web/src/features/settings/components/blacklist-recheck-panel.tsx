import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Loader2, Pencil, Play, ShieldCheck } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { ErrorState } from '@/components/shared/error-state'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { TimezoneCombobox } from '@/features/settings/components/timezone-combobox'
import { blacklistRecheckSchema, type BlacklistRecheckValues } from '@/features/settings/schema'
import { useBlacklistRecheckSettings, useRunBlacklistRecheckAutomation, useUpdateBlacklistRecheckSettings } from '@/features/settings/hooks'
import { formatDateInTimezone, formatNumber } from '@/lib/format'

export function BlacklistRecheckPanel({ canManage }: { canManage: boolean }) {
  const settings = useBlacklistRecheckSettings()
  const update = useUpdateBlacklistRecheckSettings()
  const run = useRunBlacklistRecheckAutomation()
  const [open, setOpen] = useState(false)
  const form = useForm<BlacklistRecheckValues>({ resolver: zodResolver(blacklistRecheckSchema) })
  useEffect(() => { if (settings.data && open) form.reset(settings.data) }, [form, open, settings.data])
  if (settings.isPending) return <PageSkeleton rows={5} />
  if (settings.isError) return <ErrorState error={settings.error} onRetry={() => settings.refetch()} />
  const item = settings.data
  const estimated = Math.min(Math.ceil(item.eligible_targets * item.fraction), item.max_targets)

  async function submit(values: BlacklistRecheckValues) {
    await update.mutateAsync(values)
    setOpen(false)
  }

  return (
    <>
      <Card>
        <CardHeader className="gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div><CardTitle className="flex items-center gap-2"><ShieldCheck className="size-4" />黑名单复查</CardTitle><CardDescription className="mt-1.5">决定什么时候复查、选择多少目标，以及复查成功或失败后的处理阈值。</CardDescription></div>
          {canManage ? <div className="flex gap-2"><Button variant="outline" onClick={() => run.mutate()} disabled={run.isPending}><Play />立即运行</Button><Button onClick={() => setOpen(true)}><Pencil />编辑策略</Button></div> : null}
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="flex items-center justify-between rounded-lg border p-4"><div><p className="font-medium">自动复查</p><p className="mt-1 text-sm text-muted-foreground">{item.cron_expression} · {item.timezone}</p></div><Badge variant={item.enabled ? 'default' : 'secondary'}>{item.enabled ? '已启用' : '已停用'}</Badge></div>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <div><p className="text-xs text-muted-foreground">当前候选</p><p className="mt-1 text-xl font-semibold">{formatNumber(item.eligible_targets)}</p></div>
            <div><p className="text-xs text-muted-foreground">预计本轮目标</p><p className="mt-1 text-xl font-semibold">{formatNumber(estimated)}</p></div>
            <div><p className="text-xs text-muted-foreground">选择规则</p><p className="mt-1 text-sm">{item.due_only ? '仅已到期' : '全部黑名单'} · {(item.fraction * 100).toFixed(0)}% · 最多 {item.max_targets}</p></div>
            <div><p className="text-xs text-muted-foreground">下次执行</p><p className="mt-1 text-sm">{formatDateInTimezone(item.next_run_at, item.timezone)}</p></div>
          </div>
          <div className="rounded-lg bg-muted/50 p-4 text-sm"><p className="font-medium">恢复与再次失败</p><p className="mt-1 leading-6 text-muted-foreground">复查结果可用、延迟不超过 {item.max_latency_ms} ms 且丢包不超过 {item.max_packet_loss}% 时移出黑名单；否则继续保留，并在 {item.retry_minutes} 分钟后再次进入候选。</p></div>
          {item.last_error ? <p className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">{item.last_error}</p> : null}
        </CardContent>
      </Card>

      <Sheet open={canManage && open} onOpenChange={setOpen}>
        <SheetContent className="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl">
          <SheetHeader className="shrink-0 border-b px-6 py-5 pr-12"><SheetTitle>编辑黑名单复查</SheetTitle><SheetDescription>所有自动复查和黑名单页的“立即复查”都会使用这套配置。</SheetDescription></SheetHeader>
          <Form {...form}><form onSubmit={form.handleSubmit(submit)} className="flex min-h-0 flex-1 flex-col"><div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-6 py-6">
            <FormField control={form.control} name="enabled" render={({ field }) => <FormItem className="flex items-center justify-between rounded-lg border p-4"><div><FormLabel>启用自动复查</FormLabel><FormDescription>停用后仍可手动立即运行。</FormDescription></div><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></FormItem>} />
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField control={form.control} name="cron_expression" render={({ field }) => <FormItem><FormLabel>Cron 表达式</FormLabel><FormControl><Input className="font-mono" {...field} /></FormControl><FormDescription>标准 5 段 Cron。</FormDescription><FormMessage /></FormItem>} />
              <FormField control={form.control} name="timezone" render={({ field }) => <FormItem><FormLabel>时区</FormLabel><FormControl><TimezoneCombobox value={field.value} onValueChange={field.onChange} /></FormControl><FormMessage /></FormItem>} />
            </div>
            <Separator />
            <div className="grid gap-4 sm:grid-cols-2">
              <FormField control={form.control} name="fraction" render={({ field }) => <FormItem><FormLabel>每轮选择比例</FormLabel><FormControl><Input type="number" min={1} max={100} value={Math.round(field.value * 100)} onChange={(event) => field.onChange(event.target.valueAsNumber / 100)} /></FormControl><FormDescription>填写 1–100 的百分比。</FormDescription><FormMessage /></FormItem>} />
              <FormField control={form.control} name="max_targets" render={({ field }) => <FormItem><FormLabel>单轮目标上限</FormLabel><FormControl><Input type="number" min={1} max={5000} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} /></FormControl><FormMessage /></FormItem>} />
            </div>
            <FormField control={form.control} name="due_only" render={({ field }) => <FormItem className="flex items-center justify-between rounded-lg border p-4"><div><FormLabel>仅复查已到期目标</FormLabel><FormDescription>关闭后会从整个黑名单中选择。</FormDescription></div><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></FormItem>} />
            <FormField control={form.control} name="skip_if_running" render={({ field }) => <FormItem className="flex items-center justify-between rounded-lg border p-4"><div><FormLabel>避免任务重叠</FormLabel><FormDescription>上一轮仍在运行时跳过本轮。</FormDescription></div><FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl></FormItem>} />
            <Separator />
            <div className="grid gap-4 sm:grid-cols-2">
              {([
                ['attempts', '每个 IP 尝试次数', 1, 10], ['timeout_ms', '单次超时（毫秒）', 500, 30000],
                ['max_latency_ms', '恢复延迟阈值（毫秒）', 1, 60000], ['max_packet_loss', '恢复丢包阈值（%）', 0, 100],
                ['retry_minutes', '再次失败等待（分钟）', 1, 10080],
              ] as const).map(([name, label, min, max]) => <FormField key={name} control={form.control} name={name} render={({ field }) => <FormItem><FormLabel>{label}</FormLabel><FormControl><Input type="number" min={min} max={max} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} /></FormControl><FormMessage /></FormItem>} />)}
            </div>
            </div><SheetFooter className="shrink-0 border-t bg-background px-6 py-4 pb-[max(1rem,env(safe-area-inset-bottom))]"><Button type="button" variant="outline" onClick={() => setOpen(false)}>取消</Button><Button type="submit" disabled={update.isPending}>{update.isPending ? <Loader2 className="animate-spin" /> : null}保存设置</Button></SheetFooter>
          </form></Form>
        </SheetContent>
      </Sheet>
    </>
  )
}
