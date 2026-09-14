import { DataTable, PageHeader, PageStack } from '../ui'
import type { Column } from '../ui'
import { listPermissions, listRoles, permissionLabels } from '../api/identity'
import type { Permission } from '../api/identity'
import { useResource } from '../api/useResource'

export function Permissions() {
  const permissions = useResource('permissions', listPermissions)
  const roles = useResource('roles-for-permissions', listRoles)

  const rolesByPermission = new Map<Permission, string[]>()
  for (const role of roles.data ?? []) {
    for (const permission of role.permissions) {
      rolesByPermission.set(permission, [...(rolesByPermission.get(permission) ?? []), role.name])
    }
  }

  const columns: Column<Permission>[] = [
    { key: 'key', header: 'Permission', cell: (permission) => permission },
    { key: 'description', header: 'What it allows', cell: (permission) => permissionLabels[permission] },
    {
      key: 'roles',
      header: 'Held by',
      cell: (permission) => {
        const holders = rolesByPermission.get(permission) ?? []
        return roles.status === 'ready' ? holders.join(', ') || '—' : '—'
      },
    },
  ]

  return (
    <PageStack>
      <PageHeader
        title="Permissions"
        description="The fixed vocabulary the services enforce. Roles are built from these."
      />
      <DataTable
        columns={columns}
        rows={permissions.data ?? []}
        rowKey={(permission) => permission}
        status={permissions.status}
        emptyTitle="No permissions published"
        emptyDescription="The identity service has not reported its vocabulary."
        errorDescription={permissions.error?.message}
        onRetry={permissions.reload}
      />
    </PageStack>
  )
}
