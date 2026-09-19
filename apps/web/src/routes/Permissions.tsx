import { ListPage } from '../patterns'
import { Badge } from '@/ui/shadcn/badge'
import type { Column } from '../ui'
import { listPermissions, listAllRoles, permissionLabels } from '../api/identity'
import type { Permission } from '../api/identity'
import { useResource } from '../api/useResource'

const SHOWN_HOLDERS = 4

export function Permissions() {
  const permissions = useResource('permissions', listPermissions)
  const roles = useResource('roles-for-permissions', listAllRoles)

  const rolesByPermission = new Map<Permission, string[]>()
  for (const role of roles.data ?? []) {
    for (const permission of role.permissions) {
      rolesByPermission.set(permission, [...(rolesByPermission.get(permission) ?? []), role.name])
    }
  }

  const columns: Column<Permission>[] = [
    { key: 'key', header: 'Permission', width: '16rem', cell: (permission) => permission },
    {
      key: 'description',
      header: 'What it allows',
      width: '26rem',
      wrap: true,
      cell: (permission) => permissionLabels[permission],
    },
    {
      key: 'roles',
      header: 'Held by',
      grow: true,
      wrap: true,
      // Named badges up to a point, then a count. A permission held by thirty
      // roles is a fact about the number, not a list anyone reads.
      cell: (permission) => {
        if (roles.status !== 'ready') return '—'
        const holders = rolesByPermission.get(permission) ?? []
        if (holders.length === 0) return <span className="text-subtle-foreground">No roles</span>
        return (
          <div className="flex flex-wrap items-center gap-1">
            {holders.slice(0, SHOWN_HOLDERS).map((name) => (
              <Badge key={name} variant="secondary" className="font-normal">
                {name}
              </Badge>
            ))}
            {holders.length > SHOWN_HOLDERS && (
              <span className="text-caption text-muted-foreground">
                +{holders.length - SHOWN_HOLDERS} more
              </span>
            )}
          </div>
        )
      },
    },
  ]

  return (
    <ListPage
      title="Permissions"
      description="The fixed vocabulary the services enforce. Roles are built from these."
      columns={columns}
      rows={permissions.data ?? []}
      rowKey={(permission) => permission}
      status={permissions.status}
      emptyTitle="No permissions published"
      emptyDescription="The identity service has not reported its vocabulary."
      errorDescription={permissions.error?.message}
      onRetry={permissions.reload}
    />
  )
}
