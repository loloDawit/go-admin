import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import {
  Alert,
  Button,
  CheckboxField,
  DefinitionList,
  Dialog,
  PageHeader,
  PageStack,
  SelectField,
  StateBlock,
  Status,
  TextField,
} from '../ui'
import { deactivateStaff, getStaff, listAllRoles, updateStaff } from '../api/identity'
import { useResource } from '../api/useResource'
import { useAuth } from '../api/auth'
import { isApiError } from '../api/http'
import { staffActiveTones } from '../app/statusTones'

export function StaffDetail() {
  const { staffId = '' } = useParams()
  const auth = useAuth()
  const member = useResource(`staff:${staffId}`, () => getStaff(staffId))
  const roles = useResource('roles', listAllRoles)
  const isSelf = auth.user?.staffId === staffId
  const canEdit = auth.hasPermission('edit_staff')

  const [editOpen, setEditOpen] = useState(false)
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [roleId, setRoleId] = useState('')
  const [isActive, setIsActive] = useState(true)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState<string>()

  const [deactivateOpen, setDeactivateOpen] = useState(false)
  const [deactivating, setDeactivating] = useState(false)
  const [deactivateError, setDeactivateError] = useState<string>()

  if (member.status === 'loading') {
    return <StateBlock title="Loading colleague" description="Fetching their details." />
  }

  if (member.status === 'error') {
    return (
      <StateBlock
        tone="error"
        title="This record could not be loaded"
        description={member.error?.message}
        action={
          <Button variant="secondary" onClick={member.reload}>
            Try again
          </Button>
        }
      />
    )
  }

  if (!member.data) {
    return (
      <StateBlock
        title="No such colleague"
        description="They may have been removed."
        action={<Link to="/staff">Back to staff</Link>}
      />
    )
  }

  const current = member.data
  const roleNames = new Map((roles.data ?? []).map((role) => [role.id, role.name]))

  function openEdit() {
    setFirstName(current.firstName)
    setLastName(current.lastName)
    setEmail(current.email)
    setRoleId(current.roleId)
    setIsActive(current.isActive)
    setFormError(undefined)
    setEditOpen(true)
  }

  return (
    <PageStack>
      <PageHeader
        breadcrumb={<Link to="/staff">Staff</Link>}
        title={`${current.firstName} ${current.lastName}`}
        description={current.email}
        actions={
          canEdit && (
            <>
              <Button variant="secondary" onClick={openEdit}>
                Edit
              </Button>
              {current.isActive && (
                <Button
                  variant="danger"
                  onClick={() => {
                    setDeactivateError(undefined)
                    setDeactivateOpen(true)
                  }}
                >
                  Deactivate
                </Button>
              )}
            </>
          )
        }
      />

      {deactivateError && (
        <Alert tone="danger" title="Could not deactivate this account">
          {deactivateError}
        </Alert>
      )}

      <DefinitionList
        items={[
          { term: 'Role', value: roleNames.get(current.roleId) ?? current.roleId },
          {
            term: 'Status',
            value: (
              <Status tone={staffActiveTones[current.isActive ? 'active' : 'inactive']}>
                {current.isActive ? 'Active' : 'Deactivated'}
              </Status>
            ),
          },
          {
            term: 'Password',
            value: current.mustChangePassword ? 'Must be changed at next sign-in' : 'Set by the user',
          },
        ]}
      />

      <Dialog
        open={editOpen}
        title="Edit colleague"
        description={isSelf ? 'You can change your own name and email here.' : undefined}
        onClose={() => setEditOpen(false)}
        footer={
          <>
            <Button variant="ghost" onClick={() => setEditOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              loading={saving}
              onClick={() => {
                setFormError(undefined)
                setSaving(true)
                const body = isSelf
                  ? { firstName, lastName, email }
                  : { firstName, lastName, email, roleId, isActive }
                updateStaff(staffId, body)
                  .then(() => {
                    toast.success('Changes saved')
                    setEditOpen(false)
                    member.reload()
                  })
                  .catch((cause: unknown) => {
                    setFormError(isApiError(cause) ? cause.message : 'Something went wrong.')
                  })
                  .finally(() => setSaving(false))
              }}
            >
              Save changes
            </Button>
          </>
        }
      >
        {formError && (
          <Alert tone="danger" title="Could not save changes">
            {formError}
          </Alert>
        )}
        <TextField label="First name" value={firstName} onChange={(event) => setFirstName(event.target.value)} />
        <TextField label="Last name" value={lastName} onChange={(event) => setLastName(event.target.value)} />
        <TextField
          label="Email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
        {!isSelf && (
          <>
            <SelectField label="Role" value={roleId} onChange={(event) => setRoleId(event.target.value)}>
              {(roles.data ?? []).map((role) => (
                <option key={role.id} value={role.id}>
                  {role.name}
                </option>
              ))}
            </SelectField>
            <CheckboxField
              label="Active"
              checked={isActive}
              onChange={(event) => setIsActive(event.target.checked)}
            />
          </>
        )}
      </Dialog>

      <Dialog
        open={deactivateOpen}
        title="Deactivate this account?"
        description={`${current.firstName} ${current.lastName} will no longer be able to sign in.`}
        onClose={() => setDeactivateOpen(false)}
        footer={
          <>
            <Button variant="ghost" onClick={() => setDeactivateOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="dangerSolid"
              loading={deactivating}
              onClick={() => {
                setDeactivating(true)
                deactivateStaff(staffId)
                  .then(() => {
                    toast.success('Account deactivated')
                    setDeactivateOpen(false)
                    member.reload()
                  })
                  .catch((cause: unknown) => {
                    setDeactivateOpen(false)
                    setDeactivateError(
                      isApiError(cause) && cause.code === 'last_admin'
                        ? 'At least one active person must be able to manage staff. Give someone else that role first.'
                        : isApiError(cause)
                          ? cause.message
                          : 'Something went wrong.',
                    )
                  })
                  .finally(() => setDeactivating(false))
              }}
            >
              Deactivate
            </Button>
          </>
        }
      />
    </PageStack>
  )
}
