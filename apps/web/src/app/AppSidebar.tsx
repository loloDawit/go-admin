import type { ComponentProps } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import {
  CircleUser,
  Key,
  LayoutDashboard,
  LogOut,
  Package,
  Settings as SettingsIcon,
  Shield,
  ShoppingCart,
  UserCog,
  Users,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useAuth } from '../api/auth'
import type { Permission } from '../api/identity'
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
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/ui/shadcn/sidebar'

type NavItem = { to: string; label: string; icon: LucideIcon; end?: boolean; require?: Permission }

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

const LINK_CLASS =
  'flex w-full items-center gap-2 aria-[current=page]:bg-sidebar-accent aria-[current=page]:font-medium aria-[current=page]:text-sidebar-accent-foreground'

function initials(email: string | undefined) {
  return email ? email.slice(0, 2).toUpperCase() : '?'
}

export function AppSidebar(props: ComponentProps<typeof Sidebar>) {
  const auth = useAuth()
  const navigate = useNavigate()
  const { setOpenMobile } = useSidebar()

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <div className="flex items-baseline gap-2 px-2 py-1.5 group-data-[collapsible=icon]:px-0 group-data-[collapsible=icon]:text-center">
          <span className="truncate font-semibold group-data-[collapsible=icon]:hidden">
            Northgate Supply
          </span>
        </div>
      </SidebarHeader>

      <SidebarContent role="navigation" aria-label="Sections">
        {NAV_GROUPS.map((group) => {
          const items = group.items.filter((item) => !item.require || auth.hasPermission(item.require))
          if (items.length === 0) return null
          return (
            <SidebarGroup key={group.label}>
              <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
              <SidebarMenu>
                {items.map((item) => (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton asChild tooltip={item.label}>
                      <NavLink to={item.to} end={item.end} className={LINK_CLASS} onClick={() => setOpenMobile(false)}>
                        <item.icon aria-hidden />
                        <span>{item.label}</span>
                      </NavLink>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
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
                <SidebarMenuButton size="lg" aria-label="Account">
                  <Avatar size="sm">
                    <AvatarFallback>{initials(auth.user?.email)}</AvatarFallback>
                  </Avatar>
                  <span className="truncate">{auth.user?.email}</span>
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent side="right" align="end" className="w-56">
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
