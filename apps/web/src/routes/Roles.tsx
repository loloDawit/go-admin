import { DataTable, PageHeader, PageStack } from '../ui'
import type { Column } from '../ui'
import { listRoles } from '../api/identity'
import type { Role } from '../api/identity'
import { useResource } from '../api/useResource'

const columns: Column<Role>[] = [
  { key: 'name', header: 'Role', cell: (role) => role.name },
  { key: 'description', header: 'What it allows', cell: (role) => role.description },
  { key: 'members', header: 'People', numeric: true, cell: (role) => role.memberCount },
  {
    key: 'permissions',
    header: 'Permissions',
    cell: (role) => role.permissions.join(', '),
  },
]

export function Roles() {
  const roles = useResource('roles', listRoles)

  return (
    <PageStack>
      <PageHeader
        title="Roles"
        description="A role is a named set of permissions. Staff hold exactly one."
      />
      <DataTable
        columns={columns}
        rows={roles.data ?? []}
        rowKey={(role) => role.id}
        status={roles.status}
        emptyTitle="No roles defined"
        emptyDescription="Without a role nobody can sign in to this back office."
        errorDescription={roles.error?.message}
        onRetry={roles.reload}
      />
    </PageStack>
  )
}
