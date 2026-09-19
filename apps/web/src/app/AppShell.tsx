import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  CircleUser,
  Key,
  LayoutDashboard,
  Package,
  Settings as SettingsIcon,
  Shield,
  ShoppingCart,
  UserCog,
  Users,
  X,
  Menu as MenuIcon,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Button } from '@/ui/shadcn/button'
import { Toaster } from '@/ui/shadcn/sonner'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/shadcn/dropdown-menu'
import { useAuth } from '../api/auth'
import type { Permission } from '../api/identity'

type NavItem = { to: string; label: string; icon: LucideIcon; end?: boolean; require?: Permission }

const NAV_GROUPS: { label: string; items: NavItem[] }[] = [
  {
    label: 'Shop',
    items: [
      { to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true },
      { to: '/orders', label: 'Orders', icon: ShoppingCart },
      { to: '/products', label: 'Products', icon: Package },
      { to: '/customers', label: 'Customers', icon: Users },
    ],
  },
  {
    label: 'Access',
    items: [
      { to: '/staff', label: 'Staff', icon: UserCog, require: 'view_staff' },
      { to: '/roles', label: 'Roles', icon: Shield, require: 'view_roles' },
      { to: '/permissions', label: 'Permissions', icon: Key, require: 'view_roles' },
    ],
  },
  {
    label: 'Account',
    items: [
      { to: '/profile', label: 'Profile', icon: CircleUser },
      { to: '/settings', label: 'Settings', icon: SettingsIcon },
    ],
  },
]

const LINK_BASE =
  'relative flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-label transition-colors'
const LINK_IDLE = 'text-muted-foreground hover:bg-muted hover:text-foreground'
const LINK_ACTIVE =
  'bg-accent font-medium text-accent-foreground before:absolute before:left-0 before:h-4 before:w-0.5 before:rounded-full before:bg-primary'

export function AppShell() {
  const [navOpen, setNavOpen] = useState(false)
  const auth = useAuth()
  const navigate = useNavigate()

  return (
    <div className="flex min-h-dvh flex-col">
      <header className="sticky top-0 z-20 flex h-12 items-center justify-between gap-3 border-b border-border bg-card px-4">
        <div className="flex min-w-0 items-baseline gap-2">
          <span className="font-semibold">Northgate Supply</span>
          <span className="truncate text-caption text-subtle-foreground max-lg:hidden">
            Back office
          </span>
        </div>

        <div className="flex items-center gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" aria-label="Account">
                <CircleUser />
                <span className="max-sm:hidden">{auth.user?.email}</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuLabel className="font-normal">
                <span className="block text-caption text-muted-foreground">Signed in as</span>
                <span className="block truncate">{auth.user?.email}</span>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={() => navigate('/profile')}>Profile</DropdownMenuItem>
              <DropdownMenuItem onSelect={() => navigate('/settings')}>Settings</DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onSelect={() => {
                  void auth.signOut().then(() => navigate('/login', { replace: true }))
                }}
              >
                Sign out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <Button
            variant="secondary"
            size="sm"
            className="lg:hidden"
            aria-expanded={navOpen}
            aria-controls="app-nav"
            onClick={() => setNavOpen((open) => !open)}
          >
            {navOpen ? <X /> : <MenuIcon />}
            {navOpen ? 'Close' : 'Menu'}
          </Button>
        </div>
      </header>

      <div className="grid flex-1 grid-cols-1 lg:grid-cols-[13.5rem_1fr]">
        <nav
          id="app-nav"
          aria-label="Sections"
          className={`flex flex-col gap-5 border-border bg-card p-3 max-lg:border-b lg:sticky lg:top-12 lg:h-[calc(100dvh-3rem)] lg:overflow-y-auto lg:border-r ${
            navOpen ? '' : 'max-lg:hidden'
          }`}
        >
          {NAV_GROUPS.map((group) => {
            const items = group.items.filter(
              (item) => !item.require || auth.hasPermission(item.require),
            )
            if (items.length === 0) return null
            return (
              <div className="flex flex-col gap-0.5" key={group.label}>
                <span className="mb-1 px-2.5 text-caption text-subtle-foreground">
                  {group.label}
                </span>
                {items.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.end}
                    onClick={() => setNavOpen(false)}
                    className={({ isActive }) =>
                      `${LINK_BASE} ${isActive ? LINK_ACTIVE : LINK_IDLE}`
                    }
                  >
                    <item.icon className="size-4 shrink-0" strokeWidth={1.75} aria-hidden />
                    {item.label}
                  </NavLink>
                ))}
              </div>
            )
          })}
        </nav>

        <main className="min-w-0 bg-canvas p-6 max-lg:p-4">
          <Outlet />
        </main>
      </div>

      <Toaster position="bottom-right" closeButton richColors />
    </div>
  )
}
