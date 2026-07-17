import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Loader2, Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { useCreateASN } from '@/features/sources/hooks'
import { createASNSourceSchema, defaultASNSourceValues, type CreateASNSourceValues } from '@/features/sources/schema'

export function CreateASNDialog() {
  const [open, setOpen] = useState(false)
  const create = useCreateASN()
  const form = useForm<CreateASNSourceValues>({ resolver: zodResolver(createASNSourceSchema), defaultValues: defaultASNSourceValues })

  useEffect(() => {
    if (!open) form.reset(defaultASNSourceValues)
  }, [form, open])

  async function submit(values: CreateASNSourceValues) {
    await create.mutateAsync(values)
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" />}><Plus />添加 ASN</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>添加 ASN 数据源</DialogTitle>
          <DialogDescription>仅添加你已确认属于 Cloudflare 或其网络组织的 ASN。新增后可以单独同步当前 BGP 宣告前缀。</DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form id="create-asn-form" onSubmit={form.handleSubmit(submit)} className="space-y-4">
            <FormField control={form.control} name="asn" render={({ field }) => (
              <FormItem>
                <FormLabel>ASN</FormLabel>
                <FormControl><Input type="number" min={1} value={field.value || ''} onChange={(event) => field.onChange(event.target.valueAsNumber)} onBlur={field.onBlur} name={field.name} ref={field.ref} placeholder="例如 13335" /></FormControl>
                <FormDescription>只填写数字，不需要输入 AS 前缀。</FormDescription>
                <FormMessage />
              </FormItem>
            )} />
            <FormField control={form.control} name="name" render={({ field }) => (
              <FormItem><FormLabel>名称</FormLabel><FormControl><Input placeholder="CLOUDFLARENET" {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="organization" render={({ field }) => (
              <FormItem><FormLabel>组织</FormLabel><FormControl><Input {...field} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="enabled" render={({ field }) => (
              <FormItem className="flex items-center justify-between rounded-lg border p-4 space-y-0">
                <div><FormLabel>立即启用</FormLabel><FormDescription>启用后，该 ASN 的活跃前缀会参与任务采样。</FormDescription></div>
                <FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl>
              </FormItem>
            )} />
          </form>
        </Form>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>取消</Button>
          <Button type="submit" form="create-asn-form" disabled={create.isPending}>{create.isPending ? <Loader2 className="animate-spin" /> : null}添加 ASN</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
