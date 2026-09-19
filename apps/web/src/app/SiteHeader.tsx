import { Monitor, Moon, Sun } from 'lucide-react'
import { Breadcrumb, BreadcrumbItem, BreadcrumbList, BreadcrumbPage } from '@/ui/shadcn/breadcrumb'
import { Button } from '@/ui/shadcn/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/ui/shadcn/dropdown-menu'
import { Separator } from '@/ui/shadcn/separator'
import { SidebarTrigger } from '@/ui/shadcn/sidebar'
import { useTheme, type Theme } from './ThemeProvider'

const THEME_ICON = { light: Sun, dark: Moon, system: Monitor } as const

function ThemeToggle() {
  const { theme, resolvedTheme, setTheme } = useTheme()
  const Icon = THEME_ICON[theme]

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="Theme">
          <Icon />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuRadioGroup value={theme} onValueChange={(value) => setTheme(value as Theme)}>
          <DropdownMenuRadioItem value="light">
            <Sun /> Light
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="dark">
            <Moon /> Dark
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="system">
            <Monitor /> System ({resolvedTheme})
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

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
      <div className="ml-auto">
        <ThemeToggle />
      </div>
    </header>
  )
}
