import type { CSSProperties } from 'react'
import { Outlet } from 'react-router-dom'
import { Toaster } from '@/ui/shadcn/sonner'
import { SidebarInset, SidebarProvider } from '@/ui/shadcn/sidebar'
import { TooltipProvider } from '@/ui/shadcn/tooltip'
import { AppSidebar } from './AppSidebar'
import { SiteHeader } from './SiteHeader'

// The primitive writes this cookie on every toggle but never reads it back:
// dashboard-01 reads it server-side in Next.js, which has no equivalent here.
// Read synchronously so a collapsed rail does not flash open on first paint.
function storedSidebarOpen(): boolean {
  const match = /(?:^|;\s*)sidebar_state=(true|false)/.exec(document.cookie)
  return match ? match[1] === 'true' : true
}

export function AppShell() {
  return (
    <TooltipProvider>
      <SidebarProvider
        defaultOpen={storedSidebarOpen()}
        style={
          {
            '--sidebar-width': 'calc(var(--spacing) * 72)',
            '--header-height': 'calc(var(--spacing) * 12)',
          } as CSSProperties
        }
      >
        <AppSidebar variant="inset" />
        <SidebarInset className="min-w-0">
          <SiteHeader />
          <div className="flex min-w-0 flex-1 flex-col">
            <div className="@container/main flex flex-1 flex-col gap-2">
              <div className="flex flex-col gap-4 px-4 py-4 md:gap-6 md:py-6 lg:px-6">
                <Outlet />
              </div>
            </div>
          </div>
        </SidebarInset>
      </SidebarProvider>
      <Toaster position="bottom-right" closeButton richColors />
    </TooltipProvider>
  )
}
