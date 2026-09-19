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
      <SidebarProvider defaultOpen={storedSidebarOpen()}>
        <AppSidebar />
        <SidebarInset>
          <SiteHeader />
          <div className="bg-canvas min-w-0 flex-1 p-6 max-lg:p-4">
            <Outlet />
          </div>
        </SidebarInset>
      </SidebarProvider>
      <Toaster position="bottom-right" closeButton richColors />
    </TooltipProvider>
  )
}
