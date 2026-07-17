import { ChevronDown, Server, Settings2 } from 'lucide-react'
import { useFormContext } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { StatusBadge } from '@/components/shared/status-badge'
import { useAgents } from '@/features/agents/hooks'
import { useOverview } from '@/features/dashboard/hooks'
import type { CreateScanJobValues } from '@/features/scans/schema'

export function ScanConfigurationFields({
  advancedOpen,
  onAdvancedOpenChange,
}: {
  advancedOpen: boolean
  onAdvancedOpenChange: (open: boolean) => void
}) {
  const form = useFormContext<CreateScanJobValues>()
  const agents = useAgents({ refetchInterval: 5_000 })
  const overview = useOverview()
  const samplingMode = form.watch('sampling_mode')

  return (
    <>
      <section className="space-y-4">
        <div className="flex items-center gap-2 text-sm font-medium"><Server className="size-4" />任务与探测目标</div>
        <FormField control={form.control} name="name" render={({ field }) => (
          <FormItem>
            <FormLabel>任务名称</FormLabel>
            <FormControl><Input placeholder="例如：亚洲节点日常扫描" {...field} /></FormControl>
            <FormDescription>定时计划生成的每一轮任务也会使用这个名称。</FormDescription>
            <FormMessage />
          </FormItem>
        )} />
        <div className="grid gap-4 sm:grid-cols-2">
          <FormField control={form.control} name="sampling_mode" render={({ field }) => (
            <FormItem>
              <FormLabel>扫描范围</FormLabel>
              <Select items={{ count: '按数量采样', one_per_prefix: '每个前缀取 1 个 IP' }} value={field.value} onValueChange={field.onChange}>
                <FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl>
                <SelectContent>
                  <SelectItem value="count">按数量采样</SelectItem>
                  <SelectItem value="one_per_prefix">每个前缀取 1 个 IP</SelectItem>
                </SelectContent>
              </Select>
              <FormDescription>{field.value === 'one_per_prefix' ? `覆盖全部启用 CIDR${overview.data ? `（当前来源共 ${overview.data.prefixes_total} 条，IPv6 开关会影响实际数量）` : ''}。` : '从全部启用前缀中轮询抽取指定数量。'}</FormDescription>
              <FormMessage />
            </FormItem>
          )} />
          {samplingMode === 'count' ? (
            <FormField control={form.control} name="target_count" render={({ field }) => (
              <FormItem>
                <FormLabel>采样 IP 数量</FormLabel>
                <FormControl><Input type="number" min={1} max={10_000} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} onBlur={field.onBlur} name={field.name} ref={field.ref} /></FormControl>
                <FormDescription>每个所选 Agent 都会扫描这一批目标。</FormDescription>
                <FormMessage />
              </FormItem>
            )} />
          ) : (
            <div className="rounded-lg border bg-muted/40 p-4 text-sm">
              <p className="font-medium">自动按前缀数量生成目标</p>
              <p className="mt-1 text-muted-foreground">对官方地址段与所有启用 ASN 前缀的去重并集，各选择一个唯一 IP。</p>
            </div>
          )}
          <FormField control={form.control} name="attempts" render={({ field }) => (
            <FormItem>
              <FormLabel>每个 IP 尝试次数</FormLabel>
              <FormControl><Input type="number" min={1} max={10} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} onBlur={field.onBlur} name={field.name} ref={field.ref} /></FormControl>
              <FormDescription>用于计算成功率和丢包率。</FormDescription>
              <FormMessage />
            </FormItem>
          )} />
        </div>
        <div className="grid gap-4 sm:grid-cols-[140px_1fr_110px]">
          <FormField control={form.control} name="scheme" render={({ field }) => (
            <FormItem>
              <FormLabel>协议</FormLabel>
              <Select items={{ https: 'HTTPS', http: 'HTTP' }} value={field.value} onValueChange={(value: 'http' | 'https') => {
                field.onChange(value)
                form.setValue('port', value === 'https' ? 443 : 80, { shouldDirty: true })
              }}>
                <FormControl><SelectTrigger><SelectValue /></SelectTrigger></FormControl>
                <SelectContent><SelectItem value="https">HTTPS</SelectItem><SelectItem value="http">HTTP</SelectItem></SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )} />
          <FormField control={form.control} name="hostname" render={({ field }) => (
            <FormItem>
              <FormLabel>测试域名</FormLabel>
              <FormControl><Input placeholder="cloudflare.com" {...field} /></FormControl>
              <FormMessage />
            </FormItem>
          )} />
          <FormField control={form.control} name="port" render={({ field }) => (
            <FormItem>
              <FormLabel>端口</FormLabel>
              <FormControl><Input type="number" min={1} max={65_535} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} onBlur={field.onBlur} name={field.name} ref={field.ref} /></FormControl>
              <FormMessage />
            </FormItem>
          )} />
        </div>
        <FormField control={form.control} name="path" render={({ field }) => (
          <FormItem>
            <FormLabel>请求路径</FormLabel>
            <FormControl><Input className="font-mono" placeholder="/cdn-cgi/trace" {...field} /></FormControl>
            <FormDescription>默认路径可读取 `colo=`，并保留响应头中的 CF-RAY。</FormDescription>
            <FormMessage />
          </FormItem>
        )} />
      </section>

      <Separator />

      <section className="space-y-4">
        <div>
          <p className="text-sm font-medium">执行 Agent</p>
          <p className="mt-1 text-sm text-muted-foreground">不选择时在执行时自动使用所有在线 Agent；离线节点不会接收任务。</p>
        </div>
        <FormField control={form.control} name="agent_ids" render={({ field }) => (
          <FormItem>
            <div className="grid gap-2 sm:grid-cols-2">
              {(agents.data?.items ?? []).map((agent) => {
                const checked = field.value.includes(agent.id)
                const disabled = agent.status !== 'online'
                return (
                  <label key={agent.id} className="flex cursor-pointer items-start gap-3 rounded-lg border p-3 has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-muted/50 data-[disabled=true]:cursor-not-allowed data-[disabled=true]:opacity-60" data-disabled={disabled || undefined}>
                    <Checkbox
                      checked={checked}
                      disabled={disabled}
                      onCheckedChange={(value) => field.onChange(value ? [...field.value, agent.id] : field.value.filter((id) => id !== agent.id))}
                      aria-label={`选择 ${agent.name}`}
                    />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-center justify-between gap-2"><strong className="truncate text-sm">{agent.name}</strong><StatusBadge status={agent.status} /></span>
                      <span className="mt-1 block text-xs text-muted-foreground">{agent.continent} / {agent.region} · 并发 {agent.concurrency}</span>
                    </span>
                  </label>
                )
              })}
            </div>
            {!agents.isPending && !agents.data?.items.length ? <p className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">当前没有已注册 Agent，任务执行时将无法下发。</p> : null}
            <FormMessage />
          </FormItem>
        )} />
      </section>

      <Separator />

      <Collapsible open={advancedOpen} onOpenChange={onAdvancedOpenChange}>
        <CollapsibleTrigger render={<Button type="button" variant="ghost" className="w-full justify-between px-0 hover:bg-transparent" />}>
            <span className="flex items-center gap-2"><Settings2 />阈值与高级选项</span>
            <ChevronDown className={`transition-transform ${advancedOpen ? 'rotate-180' : ''}`} />
          </CollapsibleTrigger>
        <CollapsibleContent className="space-y-5 pt-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField control={form.control} name="timeout_ms" render={({ field }) => (
              <FormItem><FormLabel>单次超时（毫秒）</FormLabel><FormControl><Input type="number" min={500} max={30_000} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} onBlur={field.onBlur} name={field.name} ref={field.ref} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="max_latency_ms" render={({ field }) => (
              <FormItem><FormLabel>延迟阈值（毫秒）</FormLabel><FormControl><Input type="number" min={1} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} onBlur={field.onBlur} name={field.name} ref={field.ref} /></FormControl><FormDescription>超过后进入当前 Agent 的黑名单。</FormDescription><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="max_packet_loss" render={({ field }) => (
              <FormItem><FormLabel>丢包阈值（%）</FormLabel><FormControl><Input type="number" min={0} max={100} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} onBlur={field.onBlur} name={field.name} ref={field.ref} /></FormControl><FormMessage /></FormItem>
            )} />
            <FormField control={form.control} name="blacklist_minutes" render={({ field }) => (
              <FormItem><FormLabel>黑名单等待时间（分钟）</FormLabel><FormControl><Input type="number" min={1} max={10_080} value={field.value} onChange={(event) => field.onChange(event.target.valueAsNumber)} onBlur={field.onBlur} name={field.name} ref={field.ref} /></FormControl><FormMessage /></FormItem>
            )} />
          </div>
          <div className="space-y-3 rounded-lg border p-4">
            <FormField control={form.control} name="include_ipv6" render={({ field }) => (
              <FormItem className="flex items-center justify-between gap-4 space-y-0">
                <div><FormLabel>包含 IPv6 采样</FormLabel><FormDescription>IPv6 只按 CIDR 采样，不会展开整个地址空间。</FormDescription></div>
                <FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl>
              </FormItem>
            )} />
            <Separator />
            <FormField control={form.control} name="include_blocked" render={({ field }) => (
              <FormItem className="flex items-center justify-between gap-4 space-y-0">
                <div><FormLabel>包含当前黑名单 IP</FormLabel><FormDescription>通常只用于人工复核，不建议日常计划开启。</FormDescription></div>
                <FormControl><Switch checked={field.value} onCheckedChange={field.onChange} /></FormControl>
              </FormItem>
            )} />
          </div>
        </CollapsibleContent>
      </Collapsible>
    </>
  )
}
