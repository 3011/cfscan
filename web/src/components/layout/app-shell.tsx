import { Outlet } from 'react-router-dom'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { AppSidebar } from '@/components/layout/app-sidebar'
import { SiteHeader } from '@/components/layout/site-header'

export function AppShell() {
  return (
    <SidebarProvider
      className="bg-sidebar"
      style={{
        '--sidebar-width': '15.5rem',
        '--sidebar-width-icon': '3rem',
      } as React.CSSProperties}
    >
      <AppSidebar />
      <SidebarInset className="min-w-0 overflow-x-hidden ring-1 ring-foreground/5 dark:ring-foreground/10">
        <SiteHeader />
        <main className="flex flex-1 flex-col bg-background">
          <div className="mx-auto flex min-w-0 w-full max-w-[1600px] flex-1 flex-col p-4 md:p-5 xl:p-6">
            <Outlet />
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  )
}
