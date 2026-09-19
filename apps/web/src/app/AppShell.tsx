import { Outlet } from 'react-router-dom'
import { Toaster } from '@/ui/shadcn/sonner'
import { SidebarInset, SidebarProvider } from '@/ui/shadcn/sidebar'
import { TooltipProvider } from '@/ui/shadcn/tooltip'
import { AppSidebar } from './AppSidebar'
import { SiteHeader } from './SiteHeader'

export function AppShell() {
  return (
    <TooltipProvider>
      <SidebarProvider>
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
