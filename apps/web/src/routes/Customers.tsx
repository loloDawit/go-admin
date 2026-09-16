import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Alert,
  Button,
  DataTable,
  Dialog,
  PageHeader,
  PageStack,
  Pagination,
  TextField,
} from '../ui'
import type { Column } from '../ui'
import { createCustomer, listCustomers } from '../api/customers'
import type { Customer } from '../api/customers'
import { isApiError } from '../api/client'
import { formatDate } from '../api/format'
import { useResource } from '../api/useResource'

const columns: Column<Customer>[] = [
  {
    key: 'name',
    header: 'Customer',
    cell: (customer) => <Link to={`/customers/${customer.id}`}>{customer.name}</Link>,
  },
  { key: 'email', header: 'Email', cell: (customer) => customer.email },
  { key: 'since', header: 'Customer since', cell: (customer) => formatDate(customer.createdAt) },
]

export function Customers() {
  const [page, setPage] = useState(1)
  const customers = useResource(`customers:${page}`, () => listCustomers(page))
  const [adding, setAdding] = useState(false)
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [emailError, setEmailError] = useState<string>()
  const [failure, setFailure] = useState<string>()
  const [saving, setSaving] = useState(false)
  const result = customers.data

  async function add() {
    setSaving(true)
    setEmailError(undefined)
    setFailure(undefined)
    try {
      await createCustomer(email.trim(), name.trim())
      setAdding(false)
      setName('')
      setEmail('')
      customers.reload()
    } catch (cause) {
      if (isApiError(cause) && cause.code === 'email_taken') {
        setEmailError('A customer already uses this email.')
      } else {
        setFailure(isApiError(cause) ? cause.message : 'The customer could not be added.')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <PageStack>
      <PageHeader
        title="Customers"
        description="Everyone an order can be placed for."
        actions={
          <Button variant="primary" onClick={() => setAdding(true)}>
            Add customer
          </Button>
        }
      />

      {failure && <Alert tone="danger" title={failure} />}

      <DataTable
        columns={columns}
        rows={result?.items ?? []}
        rowKey={(customer) => customer.id}
        status={customers.status}
        emptyTitle="No customers yet"
        emptyDescription="Add one before taking their first order."
        emptyAction={
          <Button variant="primary" onClick={() => setAdding(true)}>
            Add customer
          </Button>
        }
        errorDescription={customers.error?.message}
        onRetry={customers.reload}
      />

      {result && (
        <Pagination
          page={result.page}
          pageSize={result.pageSize}
          total={result.total}
          onChange={setPage}
        />
      )}

      <Dialog
        open={adding}
        title="Add a customer"
        onClose={() => setAdding(false)}
        footer={
          <>
            <Button onClick={() => setAdding(false)}>Cancel</Button>
            <Button
              variant="primary"
              loading={saving}
              disabled={name.trim() === '' || email.trim() === ''}
              onClick={() => void add()}
            >
              Add customer
            </Button>
          </>
        }
      >
        <TextField
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="Rita Alvarez"
        />
        <TextField
          label="Email"
          type="email"
          value={email}
          error={emailError}
          onChange={(event) => setEmail(event.target.value)}
          placeholder="rita@example.com"
        />
      </Dialog>
    </PageStack>
  )
}
