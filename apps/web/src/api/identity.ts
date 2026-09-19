import { api } from './http'
import type { components } from './generated/identity'

export type Permission = components['schemas']['Permission']
export type Auth = components['schemas']['Auth']
export type Staff = components['schemas']['Staff']
export type Role = components['schemas']['Role']
export type CreateStaffRequest = components['schemas']['CreateStaffRequest']
export type UpdateStaffRequest = components['schemas']['UpdateStaffRequest']
export type CreateRoleRequest = components['schemas']['CreateRoleRequest']
export type UpdateRoleRequest = components['schemas']['UpdateRoleRequest']

export const permissionLabels: Record<Permission, string> = {
  view_staff: 'View staff',
  edit_staff: 'Invite, edit and deactivate staff',
  view_roles: 'View roles and permissions',
  edit_roles: 'Create, edit and delete roles',
  view_products: 'View the catalog',
  edit_products: 'Create and edit products',
  view_orders: 'View orders and history',
  edit_orders: 'Advance and cancel orders',
}

export function login(email: string, password: string): Promise<Auth> {
  return api.post<Auth>('/api/v1/login', { email, password }, { suppressUnauthorized: true })
}

export function logout(): Promise<void> {
  return api.post<void>('/api/v1/logout')
}

export function getMe(): Promise<Auth> {
  return api.get<Auth>('/api/v1/me')
}

export function changeMyPassword(currentPassword: string, newPassword: string): Promise<void> {
  return api.post<void>('/api/v1/me/password', { currentPassword, newPassword })
}

export type StaffPage = components['schemas']['StaffPage']

export function listStaff(page = 1, pageSize?: number): Promise<StaffPage> {
  const params = new URLSearchParams({ page: String(page) })
  if (pageSize) params.set('pageSize', String(pageSize))
  return api.get<StaffPage>(`/api/v1/staff?${params}`)
}

export function getStaff(id: string): Promise<Staff> {
  return api.get<Staff>(`/api/v1/staff/${id}`)
}

export function createStaff(body: CreateStaffRequest): Promise<{ staff: Staff; password: string }> {
  return api.post<{ staff: Staff; password: string }>('/api/v1/staff', body)
}

export function updateStaff(id: string, body: UpdateStaffRequest): Promise<Staff> {
  return api.patch<Staff>(`/api/v1/staff/${id}`, body)
}

export function deactivateStaff(id: string): Promise<Staff> {
  return api.post<Staff>(`/api/v1/staff/${id}/deactivate`)
}

export type RolePage = components['schemas']['RolePage']

export function listRoles(page = 1, pageSize?: number): Promise<RolePage> {
  const params = new URLSearchParams({ page: String(page) })
  if (pageSize) params.set('pageSize', String(pageSize))
  return api.get<RolePage>(`/api/v1/roles?${params}`)
}

// The role picker and the permissions screen need every role, not a page of
// them. Roles are a bounded administrative vocabulary, so asking for the
// service's maximum page is honest here in a way it would not be for staff.
export async function listAllRoles(): Promise<Role[]> {
  const { items } = await api.get<RolePage>('/api/v1/roles?pageSize=100')
  return items
}

export function getRole(id: string): Promise<Role> {
  return api.get<Role>(`/api/v1/roles/${id}`)
}

export function createRole(body: CreateRoleRequest): Promise<Role> {
  return api.post<Role>('/api/v1/roles', body)
}

export function updateRole(id: string, body: UpdateRoleRequest): Promise<Role> {
  return api.patch<Role>(`/api/v1/roles/${id}`, body)
}

export function deleteRole(id: string): Promise<void> {
  return api.delete<void>(`/api/v1/roles/${id}`)
}

export async function listPermissions(): Promise<Permission[]> {
  const { permissions } = await api.get<{ permissions: Permission[] }>('/api/v1/permissions')
  return permissions
}
