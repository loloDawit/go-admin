import { DataTable, PageHeader, PageStack } from '../ui'
import type { Column } from '../ui'
import { listPermissions } from '../api/identity'
import type { Permission } from '../api/identity'
import { useResource } from '../api/useResource'
import { useScenario } from '../app/useScenario'

const columns: Column<Permission>[] = [
  { key: 'key', header: 'Permission', cell: (permission) => permission.key },
  { key: 'description', header: 'What it allows', cell: (permission) => permission.description },
  { key: 'roles', header: 'Held by', cell: (permission) => permission.roles.join(', ') },
]

export function Permissions() {
  const scenario = useScenario()
  const permissions = useResource(() => listPermissions(scenario), [scenario])

  return (
    <PageStack>
      <PageHeader
        title="Permissions"
        description="The fixed vocabulary the services enforce. Roles are built from these."
      />
      <DataTable
        columns={columns}
        rows={permissions.data ?? []}
        rowKey={(permission) => permission.key}
        status={permissions.status}
        emptyTitle="No permissions published"
        emptyDescription="The identity service has not reported its vocabulary."
        errorDescription={permissions.error?.message}
        onRetry={permissions.reload}
      />
    </PageStack>
  )
}
