import { respond } from './client'
import type { Scenario } from './client'

export type StaffStatus = 'active' | 'invited' | 'suspended'

export type StaffMember = {
  id: string
  name: string
  email: string
  role: string
  status: StaffStatus
  lastSeenAt: string
}

export type Role = {
  id: string
  name: string
  description: string
  memberCount: number
  permissions: string[]
}

export type Permission = {
  key: string
  description: string
  roles: string[]
}

const STAFF: StaffMember[] = [
  {
    id: 's-1',
    name: 'Mara Lindqvist',
    email: 'mara@northgate.example',
    role: 'Owner',
    status: 'active',
    lastSeenAt: '2026-09-13T08:02:00Z',
  },
  {
    id: 's-2',
    name: 'Dana Okafor',
    email: 'dana@northgate.example',
    role: 'Fulfilment',
    status: 'active',
    lastSeenAt: '2026-09-13T07:35:00Z',
  },
  {
    id: 's-3',
    name: 'Ben Achterberg',
    email: 'ben@northgate.example',
    role: 'Catalog',
    status: 'invited',
    lastSeenAt: '—',
  },
  {
    id: 's-4',
    name: 'Sofia Reyes',
    email: 'sofia@northgate.example',
    role: 'Fulfilment',
    status: 'suspended',
    lastSeenAt: '2026-07-30T15:12:00Z',
  },
]

const ROLES: Role[] = [
  {
    id: 'r-owner',
    name: 'Owner',
    description: 'Full access, including staff and roles.',
    memberCount: 1,
    permissions: ['staff.write', 'roles.write', 'products.write', 'orders.write'],
  },
  {
    id: 'r-catalog',
    name: 'Catalog',
    description: 'Creates and edits products; reads orders.',
    memberCount: 1,
    permissions: ['products.write', 'orders.read'],
  },
  {
    id: 'r-fulfilment',
    name: 'Fulfilment',
    description: 'Advances order status and reads the catalog.',
    memberCount: 2,
    permissions: ['orders.write', 'products.read'],
  },
]

const PERMISSIONS: Permission[] = [
  { key: 'staff.write', description: 'Invite, edit and suspend staff', roles: ['Owner'] },
  { key: 'roles.write', description: 'Create roles and assign permissions', roles: ['Owner'] },
  { key: 'products.read', description: 'View the catalog', roles: ['Owner', 'Catalog', 'Fulfilment'] },
  { key: 'products.write', description: 'Create and edit products', roles: ['Owner', 'Catalog'] },
  { key: 'orders.read', description: 'View orders and history', roles: ['Owner', 'Catalog', 'Fulfilment'] },
  { key: 'orders.write', description: 'Advance and cancel orders', roles: ['Owner', 'Fulfilment'] },
]

export function listStaff(scenario: Scenario): Promise<StaffMember[]> {
  return respond(scenario, STAFF, [])
}

export function listRoles(scenario: Scenario): Promise<Role[]> {
  return respond(scenario, ROLES, [])
}

export function listPermissions(scenario: Scenario): Promise<Permission[]> {
  return respond(scenario, PERMISSIONS, [])
}

export const staffStatusLabels: Record<StaffStatus, string> = {
  active: 'Active',
  invited: 'Invited',
  suspended: 'Suspended',
}
