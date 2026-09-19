import { useState } from 'react'
import type { ComponentProps } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import {
  CircleUser,
  Key,
  LayoutDashboard,
  LogOut,
  Package,
  MoreVertical,
  Settings as SettingsIcon,
  Store,
  Shield,
  ShoppingCart,
  UserCog,
  Users,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useAuth } from '../api/auth'
import type { Permission } from '../api/identity'
import type { OrderStatus } from '../api/orders'
import { getDashboard } from '../api/reports'
import { useResource } from '../api/useResource'
import { Avatar, AvatarFallback } from '@/ui/shadcn/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/ui/shadcn/dropdown-menu'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from '@/ui/shadcn/sidebar'

type NavItem = {
  to?: string
  label: string
  icon: LucideIcon
  end?: boolean
  require?: Permission
  children?: NavItem[]
}

export const NAV_GROUPS: { label: string; items: NavItem[] }[] = [
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
      {
        label: 'Roles & permissions',
        icon: Shield,
        children: [
          { to: '/roles', label: 'Roles', icon: Shield, require: 'view_roles' },
          { to: '/permissions', label: 'Permissions', icon: Key, require: 'view_roles' },
        ],
      },
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

// Same total Dashboard.tsx shows as "Orders to handle": placed, paid or packed.
const WAITING_STATUSES: OrderStatus[] = ['pending', 'paid', 'packed']

const LINK_CLASS =
  'flex w-full items-center gap-2 aria-[current=page]:bg-sidebar-accent aria-[current=page]:font-medium aria-[current=page]:text-sidebar-accent-foreground'

function initials(email: string | undefined) {
  return email ? email.slice(0, 2).toUpperCase() : '?'
}

// A parent with no visible children disappears too, same as a flat item without permission.
function visibleItem(
  item: NavItem,
  hasPermission: (permission: Permission) => boolean,
): NavItem | null {
  if (item.children) {
    const children = item.children.filter((child) => !child.require || hasPermission(child.require))
    return children.length > 0 ? { ...item, children } : null
  }
  return !item.require || hasPermission(item.require) ? item : null
}

function NavParentItem({
  item,
  items,
  onNavigate,
}: {
  item: NavItem
  items: NavItem[]
  onNavigate: () => void
}) {
  const [open, setOpen] = useState(true)
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        tooltip={item.label}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <item.icon aria-hidden />
        <span>{item.label}</span>
      </SidebarMenuButton>
      {open && (
        <SidebarMenuSub>
          {items.map((child) => (
            <SidebarMenuSubItem key={child.to}>
              <SidebarMenuSubButton asChild>
                <NavLink to={child.to!} className={LINK_CLASS} onClick={onNavigate}>
                  <child.icon aria-hidden />
                  <span>{child.label}</span>
                </NavLink>
              </SidebarMenuSubButton>
            </SidebarMenuSubItem>
          ))}
        </SidebarMenuSub>
      )}
    </SidebarMenuItem>
  )
}

export function AppSidebar(props: ComponentProps<typeof Sidebar>) {
  const auth = useAuth()
  const navigate = useNavigate()
  const { setOpenMobile, isMobile } = useSidebar()
  // AppShell mounts this once; it never refetches per navigation.
  const report = useResource('sidebar-dashboard', getDashboard)
  const counts = report.data?.counts
  const waiting = counts
    ? WAITING_STATUSES.reduce((sum, status) => sum + (counts[status] ?? 0), 0)
    : undefined

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild className="data-[slot=sidebar-menu-button]:p-1.5!">
              <NavLink to="/" onClick={() => setOpenMobile(false)}>
                <Store className="size-5!" aria-hidden />
                <span className="text-base font-semibold">Northgate Supply</span>
              </NavLink>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent role="navigation" aria-label="Sections">
        {NAV_GROUPS.map((group) => {
          const items = group.items
            .map((item) => visibleItem(item, auth.hasPermission))
            .filter((item): item is NavItem => item !== null)
          if (items.length === 0) return null
          return (
            <SidebarGroup key={group.label}>
              <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
              <SidebarMenu>
                {items.map((item) =>
                  item.children ? (
                    <NavParentItem
                      key={item.label}
                      item={item}
                      items={item.children}
                      onNavigate={() => setOpenMobile(false)}
                    />
                  ) : (
                    <SidebarMenuItem key={item.to}>
                      <SidebarMenuButton asChild tooltip={item.label}>
                        <NavLink
                          to={item.to!}
                          end={item.end}
                          className={LINK_CLASS}
                          onClick={() => setOpenMobile(false)}
                        >
                          <item.icon aria-hidden />
                          <span>{item.label}</span>
                        </NavLink>
                      </SidebarMenuButton>
                      {item.to === '/orders' && waiting !== undefined && waiting > 0 && (
                        <SidebarMenuBadge>{waiting}</SidebarMenuBadge>
                      )}
                    </SidebarMenuItem>
                  ),
                )}
              </SidebarMenu>
            </SidebarGroup>
          )
        })}
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton
                  size="lg"
                  aria-label="Account"
                  className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                >
                  <Avatar className="h-8 w-8 rounded-lg grayscale">
                    <AvatarFallback className="rounded-lg">
                      {initials(auth.user?.email)}
                    </AvatarFallback>
                  </Avatar>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{auth.user?.email}</span>
                    {/* The block shows a name above the email; /me returns only
                        an email, so there is no name to show. */}
                    <span className="text-muted-foreground truncate text-xs">
                      {auth.user?.permissions.length ?? 0} permissions
                    </span>
                  </div>
                  <MoreVertical className="ml-auto size-4" aria-hidden />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
                side={isMobile ? 'bottom' : 'right'}
                align="end"
                sideOffset={4}
              >
                <DropdownMenuLabel className="font-normal">
                  <span className="text-caption text-muted-foreground block">Signed in as</span>
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
                  <LogOut />
                  Sign out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  )
}
