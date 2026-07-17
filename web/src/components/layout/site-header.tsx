import { useLocation } from 'react-router-dom'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { Breadcrumb, BreadcrumbItem, BreadcrumbList, BreadcrumbPage } from '@/components/ui/breadcrumb'
import { Badge } from '@/components/ui/badge'
import { ThemeToggle } from '@/components/layout/theme-toggle'

const labels: Record<string, string> = {
  '/': '运行总览',
  '/jobs': '扫描任务',
  '/results': '结果排行',
  '/blacklist': '黑名单',
  '/sources': 'IP 数据源',
  '/agents': 'Agent 节点',
  '/settings': '设置',
  '/users': '账号与权限',
}

export function SiteHeader() {
  const location = useLocation()
  const current = labels[location.pathname] ?? 'Cloudflare IP Scanner'
  return (
    <header className="sticky top-0 z-30 flex h-12 shrink-0 items-center gap-2 border-b border-border/60 bg-background/88 px-3 backdrop-blur-xl md:px-4">
      <SidebarTrigger className="-ml-1" />
      <Breadcrumb className="min-w-0">
        <BreadcrumbList>
          <BreadcrumbItem className="min-w-0">
            <BreadcrumbPage className="truncate font-medium">{current}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="ml-auto flex items-center gap-1.5">
        <Badge variant="secondary" className="hidden gap-1.5 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 sm:flex">
          <span className="size-1.5 rounded-full bg-emerald-500" aria-hidden="true" />
          Live
        </Badge>
        <ThemeToggle />
      </div>
    </header>
  )
}
