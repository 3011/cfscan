import { useState } from 'react'
import { CalendarClock, DatabaseZap, History, Laptop, LayoutDashboard, Moon, Palette, Plus, ShieldCheck, Sun } from 'lucide-react'
import { useTheme } from 'next-themes'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { ErrorState } from '@/components/shared/error-state'
import { PageHeader } from '@/components/shared/page-header'
import { PageSkeleton } from '@/components/shared/page-skeleton'
import { AutomationOverview } from '@/features/settings/components/automation-overview'
import { AutomationRunsTable } from '@/features/settings/components/automation-runs-table'
import { BlacklistRecheckPanel } from '@/features/settings/components/blacklist-recheck-panel'
import { ScheduleDialog } from '@/features/settings/components/schedule-dialog'
import { SchedulesTable } from '@/features/settings/components/schedules-table'
import { SourceSyncPanel } from '@/features/settings/components/source-sync-panel'
import { useScanSchedules } from '@/features/settings/hooks'
import type { ScanSchedule } from '@/features/settings/types'
import { cn } from '@/lib/utils'
import { useAuth } from '@/features/auth/auth-context'

const themes = [
  { value: 'light', label: '浅色', description: '始终使用浅色界面', icon: Sun },
  { value: 'dark', label: '深色', description: '始终使用深色界面', icon: Moon },
  { value: 'system', label: '跟随系统', description: '根据操作系统自动切换', icon: Laptop },
]

function AppearanceSettings() {
  const { theme = 'system', setTheme } = useTheme()
  return (
    <Card>
      <CardHeader><CardTitle>外观</CardTitle><CardDescription>主题设置保存在当前浏览器中，不影响其他用户。</CardDescription></CardHeader>
      <CardContent>
        <RadioGroup
          value={theme}
          onValueChange={setTheme}
          className="grid gap-3 md:grid-cols-3"
          aria-label="界面主题"
        >
          {themes.map((item) => {
            const Icon = item.icon
            const selected = theme === item.value
            const id = `theme-${item.value}`
            return (
              <label
                key={item.value}
                htmlFor={id}
                className={cn(
                  'relative flex cursor-pointer items-start gap-3 rounded-3xl bg-muted/35 p-4 text-left ring-1 ring-transparent transition-all hover:bg-muted/60 has-focus-visible:ring-3 has-focus-visible:ring-ring/30',
                  selected && 'bg-card shadow-sm ring-foreground/10',
                )}
              >
                <span className="rounded-2xl bg-background p-2 shadow-sm ring-1 ring-foreground/5"><Icon className="size-4" /></span>
                <span className="pr-6"><span className="block text-sm font-medium">{item.label}</span><span className="mt-1 block text-xs leading-5 text-muted-foreground">{item.description}</span></span>
                <RadioGroupItem id={id} value={item.value} className="absolute right-3 top-3" />
              </label>
            )
          })}
        </RadioGroup>
      </CardContent>
    </Card>
  )
}

function ScanSchedulesPanel({ onCreate, onEdit, canManage }: { onCreate: () => void; onEdit: (item: ScanSchedule) => void; canManage: boolean }) {
  const schedules = useScanSchedules()
  return (
    <Card>
      <CardHeader className="gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div><CardTitle className="flex items-center gap-2"><CalendarClock className="size-4" />扫描计划</CardTitle><CardDescription className="mt-1.5">中心端按 Cron 和时区创建扫描任务；手动扫描仍在“扫描任务”页面发起。</CardDescription></div>
        {canManage ? <Button onClick={onCreate}><Plus />新建计划</Button> : null}
      </CardHeader>
      <CardContent>{schedules.isPending ? <PageSkeleton rows={5} /> : schedules.isError ? <ErrorState error={schedules.error} onRetry={() => schedules.refetch()} /> : <SchedulesTable schedules={schedules.data.items} onEdit={onEdit} canManage={canManage} />}</CardContent>
    </Card>
  )
}

const settingsSections = [
  { value: 'overview', label: '自动化总览', icon: LayoutDashboard },
  { value: 'schedules', label: '扫描计划', icon: CalendarClock },
  { value: 'blacklist', label: '黑名单复查', icon: ShieldCheck },
  { value: 'sources', label: '数据源同步', icon: DatabaseZap },
  { value: 'runs', label: '执行记录', icon: History },
  { value: 'appearance', label: '外观', icon: Palette },
] as const

type SettingsSection = (typeof settingsSections)[number]['value']

export function SettingsPage() {
  const auth = useAuth()
  const [section, setSection] = useState<SettingsSection>('overview')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<ScanSchedule | null>(null)

  return (
    <div className="page-grid">
      <PageHeader title="设置" description="集中管理自动化策略、后台同步、执行记录和当前浏览器的外观。" />
      <Tabs
        value={section}
        onValueChange={(value) => setSection(value as SettingsSection)}
        className="gap-5"
      >
        <div className="no-scrollbar max-w-full overflow-x-auto pb-1">
          <TabsList className="w-max min-w-full justify-start sm:min-w-0">
            {settingsSections.map((item) => {
              const Icon = item.icon
              return (
                <TabsTrigger
                  key={item.value}
                  value={item.value}
                  className="flex-none gap-2 px-3"
                >
                  <Icon />
                  {item.label}
                </TabsTrigger>
              )
            })}
          </TabsList>
        </div>
        <div className="min-w-0">
          <TabsContent value="overview"><AutomationOverview /></TabsContent>
          <TabsContent value="schedules">
            <ScanSchedulesPanel
              canManage={auth.canManage}
              onCreate={() => { setEditing(null); setDialogOpen(true) }}
              onEdit={(item) => { setEditing(item); setDialogOpen(true) }}
            />
          </TabsContent>
          <TabsContent value="blacklist"><BlacklistRecheckPanel canManage={auth.canManage} /></TabsContent>
          <TabsContent value="sources"><SourceSyncPanel canManage={auth.canManage} /></TabsContent>
          <TabsContent value="runs"><AutomationRunsTable /></TabsContent>
          <TabsContent value="appearance"><AppearanceSettings /></TabsContent>
        </div>
      </Tabs>
      {auth.canManage ? (
        <ScheduleDialog
          open={dialogOpen}
          onOpenChange={(open) => { setDialogOpen(open); if (!open) setEditing(null) }}
          schedule={editing}
        />
      ) : null}
    </div>
  )
}
