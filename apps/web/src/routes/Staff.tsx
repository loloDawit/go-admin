import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { Alert, Button, CheckboxField, Dialog, Pagination, SelectField, Status, TextField } from '../ui'
import { positiveInt, toSearchParams } from '../api/listQuery'
import { ListPage, SectionCard } from '../patterns'
import type { Column } from '../ui'
import { createStaff, listAllRoles, listStaff } from '../api/identity'
import type { Staff as StaffMember } from '../api/identity'
import { useResource } from '../api/useResource'
import { useAuth } from '../api/auth'
import { isApiError } from '../api/http'
import { staffActiveTones } from '../app/statusTones'

type Step = 'form' | 'password'

export function Staff() {
  const [params, setParams] = useSearchParams()
  const page = positiveInt(params, 'page') ?? 1
  const pageSize = positiveInt(params, 'pageSize')
  const staff = useResource(`staff:${page}:${pageSize ?? ''}`, () => listStaff(page, pageSize))
  const result = staff.data
  const roles = useResource('roles', listAllRoles)
  const auth = useAuth()
  const canEdit = auth.hasPermission('edit_staff')

  const [open, setOpen] = useState(false)
  const [step, setStep] = useState<Step>('form')
  const [email, setEmail] = useState('')
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [roleId, setRoleId] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string>()
  const [generatedPassword, setGeneratedPassword] = useState('')
  const [acknowledged, setAcknowledged] = useState(false)

  const roleNames = new Map((roles.data ?? []).map((role) => [role.id, role.name]))

  const columns: Column<StaffMember>[] = [
    {
      key: 'name',
      header: 'Name',
      width: '18rem',
      cell: (member) => (
        <Link to={`/staff/${member.id}`}>
          {member.firstName} {member.lastName}
        </Link>
      ),
    },
    { key: 'email', header: 'Email', grow: true, cell: (member) => member.email },
    {
      key: 'role',
      header: 'Role',
      width: '14rem',
      cell: (member) => roleNames.get(member.roleId) ?? member.roleId,
    },
    {
      key: 'status',
      header: 'Status',
      width: '9rem',
      cell: (member) => (
        <Status tone={staffActiveTones[member.isActive ? 'active' : 'inactive']}>
          {member.isActive ? 'Active' : 'Deactivated'}
        </Status>
      ),
    },
  ]

  function openDialog() {
    setEmail('')
    setFirstName('')
    setLastName('')
    // No default: picking a role, including the most privileged one, must be a deliberate choice.
    setRoleId('')
    setFormError(undefined)
    setAcknowledged(false)
    setStep('form')
    setOpen(true)
  }

  // The native close event re-enters here after Done; the reload belongs on the button instead.
  function closeDialog() {
    setOpen(false)
  }

  function dismissPasswordReveal() {
    setStep('form')
    staff.reload()
  }

  const addAction = canEdit ? (
    <Button variant="primary" onClick={openDialog}>
      Add colleague
    </Button>
  ) : undefined

  return (
    <>
      <ListPage
        title="Staff"
        description="People who can sign in to this back office."
        primaryAction={addAction}
        banner={
          step === 'password' ? (
            // Not in the dialog: this password cannot be shown again, and a
            // dialog can be dismissed by a stray Escape. Above the table so it
            // is never scrolled out of view.
            <SectionCard title="Account created">
              <Alert tone="warning" title="Shown once, never retrievable again">
                {email} can sign in with the password below. Copy it now — it cannot be displayed
                a second time.
              </Alert>
              <TextField
                label="Generated password"
                readOnly
                value={generatedPassword}
                className="font-mono"
                onFocus={(event) => event.currentTarget.select()}
              />
              <CheckboxField
                label="I have saved this password"
                checked={acknowledged}
                onChange={(event) => setAcknowledged(event.target.checked)}
              />
              <div>
                <Button variant="primary" disabled={!acknowledged} onClick={dismissPasswordReveal}>
                  Done
                </Button>
              </div>
            </SectionCard>
          ) : undefined
        }
        columns={columns}
        rows={result?.items ?? []}
        rowKey={(member) => member.id}
        rowHref={(member) => `/staff/${member.id}`}
        status={staff.status}
        emptyTitle="No colleagues yet"
        emptyDescription="Add someone and they will appear here."
        emptyAction={addAction}
        errorDescription={staff.error?.message}
        onRetry={staff.reload}
        pagination={
          result && (
            <Pagination
              page={result.page}
              pageSize={result.pageSize}
              total={result.total}
              onChange={(next) =>
                setParams(toSearchParams({ page: next === 1 ? undefined : next, pageSize }))
              }
              onPageSizeChange={(size) => setParams(toSearchParams({ pageSize: size }))}
            />
          )
        }
      />

      <Dialog
        open={open && step === 'form'}
        title="Add a colleague"
        description="A password is generated for them; they choose their own on first sign-in."
        onClose={closeDialog}
        footer={
          <>
            <Button variant="ghost" onClick={closeDialog}>
              Cancel
            </Button>
            <Button
              variant="primary"
              loading={submitting}
              onClick={() => {
                if (
                  email.trim() === '' ||
                  firstName.trim() === '' ||
                  lastName.trim() === '' ||
                  roleId === ''
                ) {
                  setFormError('Fill in every field.')
                  return
                }
                setFormError(undefined)
                setSubmitting(true)
                createStaff({ email, firstName, lastName, roleId })
                  .then((result) => {
                    toast.success('Account created')
                    setGeneratedPassword(result.password)
                    setStep('password')
                    setOpen(false)
                  })
                  .catch((cause: unknown) => {
                    setFormError(isApiError(cause) ? cause.message : 'Something went wrong.')
                  })
                  .finally(() => setSubmitting(false))
              }}
            >
              Create account
            </Button>
          </>
        }
      >
        {formError && (
          <Alert tone="danger" title="Could not create the account">
            {formError}
          </Alert>
        )}
        <TextField
          label="Email"
          type="email"
          placeholder="name@northgate.example"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
        <TextField
          label="First name"
          value={firstName}
          onChange={(event) => setFirstName(event.target.value)}
        />
        <TextField
          label="Last name"
          value={lastName}
          onChange={(event) => setLastName(event.target.value)}
        />
        <SelectField
          label="Role"
          value={roleId}
          onChange={(event) => setRoleId(event.target.value)}
          help="Decides what they can see and change."
        >
          <option value="" disabled>
            Choose a role
          </option>
          {(roles.data ?? []).map((role) => (
            <option key={role.id} value={role.id}>
              {role.name}
            </option>
          ))}
        </SelectField>
      </Dialog>
    </>
  )
}
