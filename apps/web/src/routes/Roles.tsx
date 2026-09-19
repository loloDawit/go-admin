import { useState } from 'react'
import { Alert, Button, CheckboxField, Dialog, TextField } from '../ui'
import { ListPage } from '../patterns'
import type { Column } from '../ui'
import {
  createRole,
  deleteRole,
  listPermissions,
  listRoles,
  listStaff,
  permissionLabels,
  updateRole,
} from '../api/identity'
import type { Permission, Role } from '../api/identity'
import { useResource } from '../api/useResource'
import { useAuth } from '../api/auth'
import { isApiError } from '../api/http'

type Editing = { mode: 'create' } | { mode: 'edit'; role: Role }

export function Roles() {
  const roles = useResource('roles', listRoles)
  const permissions = useResource('permissions', listPermissions)
  const staff = useResource('staff-for-roles', listStaff)
  const auth = useAuth()
  const canEdit = auth.hasPermission('edit_roles')

  const [editing, setEditing] = useState<Editing>()
  const [name, setName] = useState('')
  const [selected, setSelected] = useState<Set<Permission>>(new Set())
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState<string>()

  const [deleteTarget, setDeleteTarget] = useState<Role>()
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState<string>()

  const memberCounts = new Map<string, number>()
  for (const member of staff.data ?? []) {
    memberCounts.set(member.roleId, (memberCounts.get(member.roleId) ?? 0) + 1)
  }

  const columns: Column<Role>[] = [
    { key: 'name', header: 'Role', width: '18rem', cell: (role) => role.name },
    {
      key: 'members',
      header: 'People',
      numeric: true,
      width: '8rem',
      cell: (role) => (staff.status === 'ready' ? String(memberCounts.get(role.id) ?? 0) : '—'),
    },
    {
      key: 'permissions',
      header: 'Permissions',
      width: '12rem',
      cell: (role) => `${role.permissions.length} of ${permissions.data?.length ?? role.permissions.length}`,
    },
    {
      key: 'actions',
      header: '',
      grow: true,
      cell: (role) =>
        canEdit && (
          <div className="flex justify-end gap-2">
            <Button variant="ghost" size="sm" onClick={() => openEdit(role)}>
              Edit
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setDeleteError(undefined)
                setDeleteTarget(role)
              }}
            >
              Delete
            </Button>
          </div>
        ),
    },
  ]

  function openCreate() {
    setName('')
    setSelected(new Set())
    setFormError(undefined)
    setEditing({ mode: 'create' })
  }

  function openEdit(role: Role) {
    setName(role.name)
    setSelected(new Set(role.permissions))
    setFormError(undefined)
    setEditing({ mode: 'edit', role })
  }

  function togglePermission(permission: Permission) {
    setSelected((current) => {
      const next = new Set(current)
      if (next.has(permission)) next.delete(permission)
      else next.add(permission)
      return next
    })
  }

  const addAction = canEdit ? (
    <Button variant="primary" onClick={openCreate}>
      Add role
    </Button>
  ) : undefined

  return (
    <>
      <ListPage
        title="Roles"
        description="A role is a named set of permissions. Staff hold exactly one."
        primaryAction={addAction}
        banner={
          deleteError ? (
            <Alert tone="danger" title="Could not delete this role">
              {deleteError}
            </Alert>
          ) : undefined
        }
        columns={columns}
        rows={roles.data ?? []}
        rowKey={(role) => role.id}
        status={roles.status}
        emptyTitle="No roles defined"
        emptyDescription="Without a role nobody can sign in to this back office."
        emptyAction={addAction}
        errorDescription={roles.error?.message}
        onRetry={roles.reload}
      />

      <Dialog
        open={Boolean(editing)}
        title={editing?.mode === 'create' ? 'Add a role' : 'Edit role'}
        onClose={() => setEditing(undefined)}
        footer={
          <>
            <Button variant="ghost" onClick={() => setEditing(undefined)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              loading={saving}
              onClick={() => {
                if (name.trim() === '') {
                  setFormError('Give the role a name.')
                  return
                }
                setFormError(undefined)
                setSaving(true)
                const body = { name, permissions: Array.from(selected) }
                const request =
                  editing?.mode === 'edit' ? updateRole(editing.role.id, body) : createRole(body)
                request
                  .then(() => {
                    setEditing(undefined)
                    roles.reload()
                  })
                  .catch((cause: unknown) => {
                    setFormError(isApiError(cause) ? cause.message : 'Something went wrong.')
                  })
                  .finally(() => setSaving(false))
              }}
            >
              {editing?.mode === 'create' ? 'Create role' : 'Save changes'}
            </Button>
          </>
        }
      >
        {formError && (
          <Alert tone="danger" title="Could not save this role">
            {formError}
          </Alert>
        )}
        <TextField label="Name" value={name} onChange={(event) => setName(event.target.value)} />
        <fieldset style={{ border: 0, padding: 0, margin: 0 }}>
          <legend style={{ font: 'inherit', padding: 0, marginBottom: 'var(--space-2)' }}>Permissions</legend>
          {(permissions.data ?? []).map((permission) => (
            <CheckboxField
              key={permission}
              label={permissionLabels[permission]}
              checked={selected.has(permission)}
              onChange={() => togglePermission(permission)}
            />
          ))}
        </fieldset>
      </Dialog>

      <Dialog
        open={Boolean(deleteTarget)}
        title="Delete this role?"
        description={deleteTarget ? `${deleteTarget.name} cannot be recovered once deleted.` : undefined}
        onClose={() => setDeleteTarget(undefined)}
        footer={
          <>
            <Button variant="ghost" onClick={() => setDeleteTarget(undefined)}>
              Cancel
            </Button>
            <Button
              variant="dangerSolid"
              loading={deleting}
              onClick={() => {
                if (!deleteTarget) return
                setDeleting(true)
                deleteRole(deleteTarget.id)
                  .then(() => {
                    setDeleteTarget(undefined)
                    roles.reload()
                  })
                  .catch((cause: unknown) => {
                    setDeleteTarget(undefined)
                    setDeleteError(
                      // The service currently answers "role_in_use", not the documented "conflict" —
                      // both are matched so this reads correctly however that drift gets resolved.
                      isApiError(cause) && ['conflict', 'role_in_use'].includes(cause.code)
                        ? 'People still hold this role. Move them to another role first.'
                        : isApiError(cause)
                          ? cause.message
                          : 'Something went wrong.',
                    )
                  })
                  .finally(() => setDeleting(false))
              }}
            >
              Delete
            </Button>
          </>
        }
      />
    </>
  )
}
