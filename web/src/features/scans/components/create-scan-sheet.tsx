import { useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Loader2, Plus } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Form } from '@/components/ui/form'
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { useAgents } from '@/features/agents/hooks'
import { ScanConfigurationFields } from '@/features/scans/components/scan-configuration-fields'
import { useCreateScanJob } from '@/features/scans/hooks'
import { createScanJobSchema, defaultScanJobValues, type CreateScanJobValues } from '@/features/scans/schema'

export function CreateScanSheet() {
  const [open, setOpen] = useState(false)
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [confirmClose, setConfirmClose] = useState(false)
  const agents = useAgents({ refetchInterval: 5_000 })
  const createJob = useCreateScanJob()
  const form = useForm<CreateScanJobValues>({ resolver: zodResolver(createScanJobSchema), defaultValues: defaultScanJobValues })
  const isDirty = form.formState.isDirty

  function requestOpenChange(nextOpen: boolean) {
    if (!nextOpen && isDirty && !createJob.isPending) {
      setConfirmClose(true)
      return
    }
    if (!nextOpen) {
      form.reset(defaultScanJobValues)
      setAdvancedOpen(false)
    }
    setOpen(nextOpen)
  }

  async function submit(values: CreateScanJobValues) {
    await createJob.mutateAsync(values)
    form.reset(defaultScanJobValues)
    setOpen(false)
  }

  const onlineAgents = agents.data?.items.filter((agent) => agent.status === 'online') ?? []

  return (
    <>
      <Sheet open={open} onOpenChange={requestOpenChange}>
        <SheetTrigger render={<Button />}><Plus />创建扫描任务</SheetTrigger>
        <SheetContent className="flex w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-xl lg:max-w-2xl">
          <SheetHeader className="shrink-0 border-b px-6 py-5 pr-12">
            <SheetTitle>创建扫描任务</SheetTitle>
            <SheetDescription>从当前启用的 Cloudflare 前缀中采样目标，并将同一批 IP 分发给所选 Agent。</SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(submit)} className="flex min-h-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
                <ScanConfigurationFields advancedOpen={advancedOpen} onAdvancedOpenChange={setAdvancedOpen} />
              </div>
              <SheetFooter className="shrink-0 border-t bg-background px-6 py-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
                <Button type="button" variant="outline" onClick={() => requestOpenChange(false)}>取消</Button>
                <Button type="submit" disabled={createJob.isPending || onlineAgents.length === 0}>
                  {createJob.isPending ? <Loader2 className="animate-spin" /> : null}创建并下发
                </Button>
              </SheetFooter>
            </form>
          </Form>
        </SheetContent>
      </Sheet>
      <AlertDialog open={confirmClose} onOpenChange={setConfirmClose}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>放弃未保存的任务配置？</AlertDialogTitle><AlertDialogDescription>当前表单包含尚未提交的修改，关闭后这些内容会被清空。</AlertDialogDescription></AlertDialogHeader>
          <AlertDialogFooter><AlertDialogCancel>继续编辑</AlertDialogCancel><AlertDialogAction onClick={() => { form.reset(defaultScanJobValues); setConfirmClose(false); setOpen(false) }}>放弃修改</AlertDialogAction></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
