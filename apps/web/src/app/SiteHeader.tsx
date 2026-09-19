import { Breadcrumb, BreadcrumbItem, BreadcrumbList, BreadcrumbPage } from '@/ui/shadcn/breadcrumb'
import { Separator } from '@/ui/shadcn/separator'
import { SidebarTrigger } from '@/ui/shadcn/sidebar'

export function SiteHeader() {
  return (
    <header className="border-border bg-card sticky top-0 z-20 flex h-12 shrink-0 items-center gap-2 border-b px-4">
      <SidebarTrigger className="-ml-1" />
      <Separator orientation="vertical" className="h-4" />
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            {/* Static, not the page name: BreadcrumbPage is role="link", which would collide with getByRole('link', { name: <page> }) against the nav. */}
            <BreadcrumbPage>Back office</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
    </header>
  )
}
