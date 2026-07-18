import { useMemo, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { Check, Copy, Loader2, Plus, ServerCog, TerminalSquare } from 'lucide-react'
import { toast } from 'sonner'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAgentEnrollmentConfig, useCreatePreauthorizedEnrollment } from '@/features/agents/hooks'
import type { CreatePreauthorizedEnrollmentResponse } from '@/features/agents/types'

const continents = ['Africa', 'Asia', 'Europe', 'North America', 'Oceania', 'South America'] as const
const preauthorizedSchema = z.object({
  name: z.string().trim().min(1, '请输入节点名称').max(128),
  region: z.string().trim().min(1, '请输入地区').max(128),
  continent: z.string().min(1, '请选择大洲'),
  concurrency: z.number().int().min(1).max(4096),
  ttl_minutes: z.number().int().min(5).max(1440),
})
type PreauthorizedValues = z.infer<typeof preauthorizedSchema>

function CommandBlock({ value }: { value: string }) {
  return (
    <div className="relative rounded-2xl bg-muted p-4 pr-12">
      <pre className="overflow-x-auto text-xs leading-6"><code>{value}</code></pre>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="absolute right-2 top-2"
        aria-label="复制命令"
        onClick={async () => { await navigator.clipboard.writeText(value); toast.success('命令已复制') }}
      ><Copy /></Button>
    </div>
  )
}

export function AddAgentSheet() {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'device' | 'preauthorized'>('device')
  const [generated, setGenerated] = useState<CreatePreauthorizedEnrollmentResponse | null>(null)
  const config = useAgentEnrollmentConfig()
  const create = useCreatePreauthorizedEnrollment()
  const form = useForm<PreauthorizedValues>({
    resolver: zodResolver(preauthorizedSchema),
    defaultValues: { name: '', region: '', continent: 'Asia', concurrency: 64, ttl_minutes: 30 },
  })

  const agentURL = config.data?.public_url || window.location.origin
  const image = config.data?.agent_image || 'ghcr.io/3011/cfscan-agent:latest'
  const deviceCommands = useMemo(() => ({
    binary: `cfscan-agent connect --server ${agentURL}`,
    docker: `docker run -d \\\n  --name cfscan-agent \\\n  --restart unless-stopped \\\n  -e CFSCAN_AGENT_IDENTITY_FILE=/var/lib/cfscan-agent/identity.json \\\n  -v cfscan-agent-data:/var/lib/cfscan-agent \\\n  ${image} connect --server ${agentURL}`,
  }), [agentURL, image])
  const joinCommands = generated ? {
    binary: `cfscan-agent join --server ${agentURL} --token ${generated.pairing_token}`,
    docker: `docker run -d \\\n  --name cfscan-agent \\\n  --restart unless-stopped \\\n  -e CFSCAN_AGENT_IDENTITY_FILE=/var/lib/cfscan-agent/identity.json \\\n  -v cfscan-agent-data:/var/lib/cfscan-agent \\\n  ${image} join --server ${agentURL} --token ${generated.pairing_token}`,
  } : null

  async function submit(values: PreauthorizedValues) {
    setGenerated(await create.mutateAsync(values))
  }

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen)
    if (!nextOpen) {
      setMode('device')
      setGenerated(null)
      form.reset()
    }
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetTrigger render={<Button />}><Plus />添加 Agent</SheetTrigger>
      <SheetContent className="sm:max-w-xl">
        <SheetHeader>
          <SheetTitle>连接 Agent</SheetTitle>
          <SheetDescription>普通部署通过浏览器批准；无人值守部署使用一次性预授权密钥。</SheetDescription>
        </SheetHeader>
        <div className="flex-1 overflow-y-auto px-6 pb-6">
          <RadioGroup value={mode} onValueChange={(value) => { setMode(value as typeof mode); setGenerated(null) }} className="grid gap-3 sm:grid-cols-2">
            <Label className="items-start gap-3 rounded-2xl border p-4 has-data-checked:border-primary has-data-checked:bg-primary/5">
              <RadioGroupItem value="device" className="mt-0.5" />
              <span className="space-y-1"><span className="flex items-center gap-2 font-medium"><TerminalSquare />从 Agent 发起 <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[10px] text-primary">推荐</span></span><span className="block text-xs font-normal leading-5 text-muted-foreground">运行无秘密命令，再通过 Web 确认设备信息。</span></span>
            </Label>
            <Label className="items-start gap-3 rounded-2xl border p-4 has-data-checked:border-primary has-data-checked:bg-primary/5">
              <RadioGroupItem value="preauthorized" className="mt-0.5" />
              <span className="space-y-1"><span className="flex items-center gap-2 font-medium"><ServerCog />自动化部署</span><span className="block text-xs font-normal leading-5 text-muted-foreground">预先批准 Agent，适合脚本和无人值守安装。</span></span>
            </Label>
          </RadioGroup>

          {mode === 'device' ? (
            <div className="mt-6 space-y-5">
              <ol className="space-y-3 text-sm text-muted-foreground">
                <li><span className="mr-2 font-medium text-foreground">1.</span>在目标机器运行命令。</li>
                <li><span className="mr-2 font-medium text-foreground">2.</span>打开终端中显示的配对 URL。</li>
                <li><span className="mr-2 font-medium text-foreground">3.</span>确认设备信息并批准，Agent 会自动保存身份并开始运行。</li>
              </ol>
              <Tabs defaultValue="binary">
                <TabsList><TabsTrigger value="binary">二进制</TabsTrigger><TabsTrigger value="docker">Docker</TabsTrigger></TabsList>
                <TabsContent value="binary" className="mt-3"><CommandBlock value={deviceCommands.binary} /></TabsContent>
                <TabsContent value="docker" className="mt-3 space-y-2"><CommandBlock value={deviceCommands.docker} /><p className="text-xs text-muted-foreground">使用 <code>docker logs -f cfscan-agent</code> 查看配对 URL。</p></TabsContent>
              </Tabs>
            </div>
          ) : generated && joinCommands ? (
            <div className="mt-6 space-y-5">
              <div className="flex items-start gap-3 rounded-2xl border bg-muted/30 p-4"><span className="flex size-8 items-center justify-center rounded-full bg-primary text-primary-foreground"><Check /></span><div><p className="font-medium">一次性部署命令已生成</p><p className="mt-1 text-xs text-muted-foreground">仅可使用一次，将在 {Math.round(generated.expires_in / 60)} 分钟后失效。</p></div></div>
              <Tabs defaultValue="binary">
                <TabsList><TabsTrigger value="binary">二进制</TabsTrigger><TabsTrigger value="docker">Docker</TabsTrigger></TabsList>
                <TabsContent value="binary" className="mt-3"><CommandBlock value={joinCommands.binary} /></TabsContent>
                <TabsContent value="docker" className="mt-3"><CommandBlock value={joinCommands.docker} /></TabsContent>
              </Tabs>
              <p className="text-xs leading-5 text-muted-foreground">一次性密钥会出现在命令中。正式自动化环境也可通过 <code>--token-file</code> 或 <code>--token-stdin</code> 传入，避免进入 Shell 历史。</p>
              <Button type="button" variant="outline" onClick={() => setGenerated(null)}>重新生成</Button>
            </div>
          ) : (
            <Form {...form}>
              <form id="preauthorize-agent-form" className="mt-6 space-y-4" onSubmit={form.handleSubmit(submit)}>
                <FormField control={form.control} name="name" render={({ field }) => <FormItem><FormLabel>节点名称</FormLabel><FormControl><Input placeholder="edge-node-01" {...field} /></FormControl><FormMessage /></FormItem>} />
                <FormField control={form.control} name="region" render={({ field }) => <FormItem><FormLabel>地区</FormLabel><FormControl><Input placeholder="Guangzhou" {...field} /></FormControl><FormMessage /></FormItem>} />
                <FormField control={form.control} name="continent" render={({ field }) => <FormItem><FormLabel>大洲</FormLabel><Select items={Object.fromEntries(continents.map((item) => [item, item]))} value={field.value} onValueChange={field.onChange}><FormControl><SelectTrigger className="w-full"><SelectValue /></SelectTrigger></FormControl><SelectContent>{continents.map((item) => <SelectItem key={item} value={item}>{item}</SelectItem>)}</SelectContent></Select><FormMessage /></FormItem>} />
                <div className="grid gap-4 sm:grid-cols-2">
                  <FormField control={form.control} name="concurrency" render={({ field }) => <FormItem><FormLabel>最大并发数</FormLabel><FormControl><Input type="number" min={1} max={4096} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} /></FormControl><FormMessage /></FormItem>} />
                  <FormField control={form.control} name="ttl_minutes" render={({ field }) => <FormItem><FormLabel>有效期（分钟）</FormLabel><FormControl><Input type="number" min={5} max={1440} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} /></FormControl><FormDescription>默认 30 分钟，使用一次后立即失效。</FormDescription><FormMessage /></FormItem>} />
                </div>
              </form>
            </Form>
          )}
        </div>
        <SheetFooter className="border-t bg-popover">
          {mode === 'preauthorized' && !generated ? <Button type="submit" form="preauthorize-agent-form" disabled={create.isPending}>{create.isPending ? <Loader2 className="animate-spin" /> : null}生成部署命令</Button> : null}
          <Button type="button" variant="outline" onClick={() => setOpen(false)}>关闭</Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
