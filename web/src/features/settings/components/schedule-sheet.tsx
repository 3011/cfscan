import { useEffect, useMemo, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { CalendarClock, Loader2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { TimezoneCombobox } from '@/features/settings/components/timezone-combobox'
import { ScanConfigurationFields } from '@/features/scans/components/scan-configuration-fields'
import { useCreateSchedule, useUpdateSchedule } from '@/features/settings/hooks'
import { defaultScanScheduleValues, scanScheduleSchema, type ScanScheduleValues } from '@/features/settings/schema'
import type { ScanSchedule } from '@/features/settings/types'

const cronPresets = [
  { label: '每小时', value: '0 * * * *' },
  { label: '每 6 小时', value: '0 */6 * * *' },
  { label: '每天 02:00', value: '0 2 * * *' },
  { label: '每天 08:00', value: '0 8 * * *' },
  { label: '每周一 08:00', value: '0 8 * * 1' },
]

function valuesFromSchedule(schedule?: ScanSchedule | null): ScanScheduleValues {
  if (!schedule) return defaultScanScheduleValues
  return {
    name: schedule.name,
    enabled: schedule.enabled,
    cron_expression: schedule.cron_expression,
    timezone: schedule.timezone,
    agent_ids: schedule.agent_ids,
    sampling_mode: schedule.sampling_mode,
    target_count: schedule.target_count,
    scheme: schedule.scheme,
    hostname: schedule.hostname,
    path: schedule.path,
    port: schedule.port,
    attempts: schedule.attempts,
    timeout_ms: schedule.timeout_ms,
    max_latency_ms: schedule.max_latency_ms,
    max_packet_loss: schedule.max_packet_loss,
    blacklist_minutes: schedule.blacklist_minutes,
    include_ipv6: schedule.include_ipv6,
    include_blocked: schedule.include_blocked,
  }
}

export function ScheduleSheet({
  open,
  onOpenChange,
  schedule,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  schedule?: ScanSchedule | null
}) {
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [confirmClose, setConfirmClose] = useState(false)
  const create = useCreateSchedule()
  const update = useUpdateSchedule()
  const pending = create.isPending || update.isPending
  const form = useForm<ScanScheduleValues>({ resolver: zodResolver(scanScheduleSchema), defaultValues: valuesFromSchedule(schedule) })
  const isDirty = form.formState.isDirty

  useEffect(() => {
    if (open) form.reset(valuesFromSchedule(schedule))
  }, [form, open, schedule])

  const expression = form.watch('cron_expression')
  const preset = useMemo(() => cronPresets.find((item) => item.value === expression)?.value ?? 'custom', [expression])

  function requestClose(nextOpen: boolean) {
    if (!nextOpen && isDirty && !pending) {
      setConfirmClose(true)
      return
    }
    if (!nextOpen) {
      form.reset(valuesFromSchedule(schedule))
      setAdvancedOpen(false)
    }
    onOpenChange(nextOpen)
  }

  async function submit(values: ScanScheduleValues) {
    if (schedule) await update.mutateAsync({ id: schedule.id, input: values })
    else await create.mutateAsync(values)
    form.reset(values)
    onOpenChange(false)
  }

  return (
    <>
      <Sheet open={open} onOpenChange={requestClose}>
        <SheetContent className="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl lg:max-w-2xl">
          <SheetHeader className="shrink-0 border-b px-6 py-5 pr-12">
            <SheetTitle>{schedule ? '编辑定时计划' : '新建定时计划'}</SheetTitle>
            <SheetDescription>中心端按 Cron 和时区创建扫描任务；浏览器关闭或服务器重启不会丢失计划。</SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(submit)} className="flex min-h-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 space-y-7 overflow-y-auto px-6 py-6">
              <section className="space-y-4">
                <div className="flex items-center gap-2 text-sm font-medium"><CalendarClock className="size-4" />执行计划</div>
                <FormField control={form.control} name="enabled" render={({ field }) => (
                  <FormItem className="flex items-center justify-between gap-4 rounded-lg border p-4 space-y-0">
                    <div><FormLabel>启用计划</FormLabel><FormDescription>停用后保留配置，但中心不会自动创建任务。</FormDescription></div>
                    <FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl>
                  </FormItem>
                )} />
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label>常用频率</Label>
                    <Select
                      items={Object.fromEntries([...cronPresets, { label: '自定义 Cron', value: 'custom' }].map((item) => [item.value, item.label]))}
                      value={preset}
                      onValueChange={(value) => { if (value !== 'custom') form.setValue('cron_expression', value, { shouldDirty: true, shouldValidate: true }) }}
                    >
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        {cronPresets.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}
                        <SelectItem value="custom">自定义 Cron</SelectItem>
                      </SelectContent>
                    </Select>
                    <p className="text-[0.8rem] text-muted-foreground">选择预设后仍可直接修改表达式。</p>
                  </div>
                  <FormField control={form.control} name="timezone" render={({ field }) => (
                    <FormItem>
                      <FormLabel>时区</FormLabel>
                      <FormControl><TimezoneCombobox value={field.value} onValueChange={field.onChange} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )} />
                </div>
                <FormField control={form.control} name="cron_expression" render={({ field }) => (
                  <FormItem>
                    <FormLabel>Cron 表达式</FormLabel>
                    <FormControl><Input className="font-mono" placeholder="0 */6 * * *" {...field} /></FormControl>
                    <FormDescription>标准 5 段格式：分钟、小时、日期、月份、星期；也支持 @hourly 和 @daily。</FormDescription>
                    <FormMessage />
                  </FormItem>
                )} />
              </section>
              <Separator />
              <ScanConfigurationFields advancedOpen={advancedOpen} onAdvancedOpenChange={setAdvancedOpen} />
              </div>
              <SheetFooter className="shrink-0 border-t bg-background px-6 py-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
                <Button type="button" variant="outline" onClick={() => requestClose(false)}>取消</Button>
                <Button type="submit" disabled={pending}>{pending ? <Loader2 className="animate-spin" /> : null}{schedule ? '保存修改' : '创建计划'}</Button>
              </SheetFooter>
            </form>
          </Form>
        </SheetContent>
      </Sheet>
      <AlertDialog open={confirmClose} onOpenChange={setConfirmClose}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>放弃未保存的计划配置？</AlertDialogTitle><AlertDialogDescription>关闭后，本次修改不会保存。</AlertDialogDescription></AlertDialogHeader>
          <AlertDialogFooter><AlertDialogCancel>继续编辑</AlertDialogCancel><AlertDialogAction onClick={() => { form.reset(valuesFromSchedule(schedule)); setConfirmClose(false); onOpenChange(false) }}>放弃修改</AlertDialogAction></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
