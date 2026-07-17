import { Activity, Ban, Bot, ChevronsUpDown, Cloud, CloudCog, Gauge, ListChecks, LogOut, Settings, UserRound, Users } from 'lucide-react'
import { NavLink, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/features/auth/auth-context'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/components/ui/sidebar'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'

const groups = [
  {
    label: '工作台',
    items: [{ to: '/', label: '运行总览', icon: Gauge }],
  },
  {
    label: '扫描运营',
    items: [
      { to: '/jobs', label: '扫描任务', icon: ListChecks },
      { to: '/results', label: '结果排行', icon: Activity },
      { to: '/blacklist', label: '黑名单', icon: Ban },
    ],
  },
  {
    label: '资源管理',
    items: [
      { to: '/sources', label: 'IP 数据源', icon: CloudCog },
      { to: '/agents', label: 'Agent 节点', icon: Bot },
    ],
  },
  {
    label: '系统',
    items: [
      { to: '/settings', label: '设置', icon: Settings },
      { to: '/users', label: '账号与权限', icon: Users, adminOnly: true },
    ],
  },
]

export function AppSidebar() {
  const location = useLocation()
  const navigate = useNavigate()
  const auth = useAuth()
  return (
    <Sidebar collapsible="icon" variant="inset">
      <SidebarHeader className="pb-1 group-data-[collapsible=icon]:pb-3">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" className="h-11" render={<NavLink to="/" />}>
              <span className="flex size-8 shrink-0 items-center justify-center rounded-2xl bg-sidebar-primary text-sidebar-primary-foreground shadow-sm">
                <Cloud className="size-4" />
              </span>
              <span className="grid min-w-0 flex-1 text-left leading-tight">
                <span className="truncate font-heading text-sm font-medium">CF Scanner</span>
                <span className="truncate text-[11px] text-sidebar-foreground/55">Network Quality</span>
              </span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent className="gap-0.5 group-data-[collapsible=icon]:gap-0.5 group-data-[collapsible=icon]:pt-2">
        {groups.map((group) => (
          <SidebarGroup key={group.label} className="py-1.5 group-data-[collapsible=icon]:py-0">
            <SidebarGroupLabel className="h-7 px-2.5 text-[11px] tracking-wide group-data-[collapsible=icon]:hidden">{group.label}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.filter((item) => !('adminOnly' in item && item.adminOnly) || auth.canManage).map((item) => {
                  const active = item.to === '/' ? location.pathname === '/' : location.pathname.startsWith(item.to)
                  return (
                    <SidebarMenuItem key={item.to}>
                      <SidebarMenuButton isActive={active} tooltip={item.label} render={<NavLink to={item.to} />}>
                        <item.icon />
                        <span>{item.label}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter className="pt-1">
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger render={<SidebarMenuButton size="lg" className="h-11 data-popup-open:bg-sidebar-accent" />}>
                <span className="flex size-8 shrink-0 items-center justify-center rounded-2xl bg-sidebar-accent text-sidebar-accent-foreground">
                  <UserRound className="size-4" />
                </span>
                <span className="grid min-w-0 flex-1 text-left leading-tight">
                  <span className="truncate text-sm font-medium">{auth.user?.display_name}</span>
                  <span className="truncate text-[11px] text-sidebar-foreground/55">@{auth.user?.username} · {auth.canManage ? '管理员' : '查看者'}</span>
                </span>
                <ChevronsUpDown className="ml-auto size-3.5 text-sidebar-foreground/50" />
              </DropdownMenuTrigger>
              <DropdownMenuContent side="right" align="end" className="w-60">
                <DropdownMenuLabel>
                  <span className="block truncate text-sm text-foreground">{auth.user?.display_name}</span>
                  <span className="mt-1 block truncate font-normal text-muted-foreground">@{auth.user?.username}</span>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                {auth.canManage ? <DropdownMenuItem onClick={() => navigate('/users')}><Users />账号与权限</DropdownMenuItem> : null}
                <DropdownMenuItem disabled={auth.logoutPending} onClick={async () => { await auth.logout(); navigate('/login', { replace: true }) }}><LogOut />退出登录</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
