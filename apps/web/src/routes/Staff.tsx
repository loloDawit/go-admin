import { useState } from 'react'
import {
  Button,
  DataTable,
  Dialog,
  PageHeader,
  PageStack,
  SelectField,
  Status,
  TextField,
} from '../ui'
import type { Column } from '../ui'
import { listStaff, staffStatusLabels } from '../api/identity'
import type { StaffMember } from '../api/identity'
import { formatDateTime } from '../api/format'
import { useResource } from '../api/useResource'
import { staffStatusTones } from '../app/statusTones'

const columns: Column<StaffMember>[] = [
  { key: 'name', header: 'Name', cell: (member) => member.name },
  { key: 'email', header: 'Email', cell: (member) => member.email },
  { key: 'role', header: 'Role', cell: (member) => member.role },
  {
    key: 'status',
    header: 'Status',
    cell: (member) => <Status tone={staffStatusTones[member.status]}>{staffStatusLabels[member.status]}</Status>,
  },
  {
    key: 'seen',
    header: 'Last seen',
    cell: (member) => formatDateTime(member.lastSeenAt),
  },
]

export function Staff() {
  const staff = useResource('staff', listStaff)
  const [inviteOpen, setInviteOpen] = useState(false)

  return (
    <PageStack>
      <PageHeader
        title="Staff"
        description="People who can sign in to this back office."
        actions={
          <Button variant="primary" onClick={() => setInviteOpen(true)}>
            Invite colleague
          </Button>
        }
      />

      <DataTable
        columns={columns}
        rows={staff.data ?? []}
        rowKey={(member) => member.id}
        status={staff.status}
        emptyTitle="No colleagues yet"
        emptyDescription="Invite someone and they will appear here once they accept."
        emptyAction={
          <Button variant="primary" onClick={() => setInviteOpen(true)}>
            Invite colleague
          </Button>
        }
        errorDescription={staff.error?.message}
        onRetry={staff.reload}
      />

      <Dialog
        open={inviteOpen}
        title="Invite a colleague"
        description="They receive an email and choose their own password."
        onClose={() => setInviteOpen(false)}
        footer={
          <>
            <Button variant="ghost" onClick={() => setInviteOpen(false)}>
              Cancel
            </Button>
            <Button variant="primary" onClick={() => setInviteOpen(false)}>
              Send invitation
            </Button>
          </>
        }
      >
        <TextField label="Email" type="email" placeholder="name@northgate.example" />
        <SelectField label="Role" defaultValue="Fulfilment" help="Roles decide what they can change.">
          <option>Owner</option>
          <option>Catalog</option>
          <option>Fulfilment</option>
        </SelectField>
      </Dialog>
    </PageStack>
  )
}
