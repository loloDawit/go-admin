import { useState } from 'react'
import {
  Alert,
  Button,
  CheckboxField,
  DataTable,
  Dialog,
  PageHeader,
  PageStack,
  Section,
  SelectField,
  StateBlock,
  Status,
  TextField,
  TextareaField,
} from '../ui'
import type { Column } from '../ui'
import styles from './Kit.module.css'

type KitRow = { id: string; label: string; amount: string }

const columns: Column<KitRow>[] = [
  { key: 'label', header: 'Label', cell: (row) => row.label },
  { key: 'amount', header: 'Amount', numeric: true, cell: (row) => row.amount },
]

const rows: KitRow[] = [
  { id: '1', label: 'First row', amount: '£12.00' },
  { id: '2', label: 'Second row', amount: '£340.00' },
]

const TYPE_SCALE = [
  { token: '2xl / 24px', className: styles.size2xl },
  { token: 'xl / 20px', className: styles.sizeXl },
  { token: 'lg / 16px', className: styles.sizeLg },
  { token: 'base / 14px', className: styles.sizeBase },
  { token: 'md / 13px', className: styles.sizeMd },
  { token: 'sm / 12px', className: styles.sizeSm },
]

const SWATCHES = [
  { name: 'accent', className: styles.chipAccent },
  { name: 'success', className: styles.chipSuccess },
  { name: 'warning', className: styles.chipWarning },
  { name: 'danger', className: styles.chipDanger },
  { name: 'info', className: styles.chipInfo },
  { name: 'canvas', className: styles.chipCanvas },
  { name: 'surface', className: styles.chipSurface },
  { name: 'border', className: styles.chipBorder },
]

export function Kit() {
  const [dialogOpen, setDialogOpen] = useState(false)

  return (
    <PageStack>
      <PageHeader
        title="Interface kit"
        description="Every primitive in each of its states. Not part of the product navigation for staff."
      />

      <Section title="Colour">
        <div className={styles.swatches}>
          {SWATCHES.map((swatch) => (
            <div key={swatch.name} className={styles.swatch}>
              <span className={`${styles.chip} ${swatch.className}`} />
              {swatch.name}
            </div>
          ))}
        </div>
      </Section>

      <Section title="Type scale">
        <div className={styles.scale}>
          {TYPE_SCALE.map((step) => (
            <div key={step.token} className={styles.scaleRow}>
              <span className={styles.scaleLabel}>{step.token}</span>
              <span className={step.className}>Orders waiting on you</span>
            </div>
          ))}
        </div>
      </Section>

      <Section title="Buttons">
        <div className={styles.row}>
          <Button variant="primary">Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="danger">Danger</Button>
          <Button variant="ghost">Ghost</Button>
        </div>
        <div className={styles.row}>
          <Button variant="primary" loading>
            Saving
          </Button>
          <Button variant="primary" disabled>
            Disabled
          </Button>
          <Button variant="secondary" size="sm">
            Small
          </Button>
          <Button variant="secondary" size="lg">
            Large
          </Button>
        </div>
      </Section>

      <Section title="Form controls">
        <div className={styles.grid}>
          <TextField label="Default" placeholder="Placeholder" />
          <TextField label="With help" help="Explains the expected value." />
          <TextField label="With error" defaultValue="12" error="Enter a whole number above zero." />
          <TextField label="Disabled" defaultValue="Not editable" disabled />
          <SelectField label="Select" defaultValue="Kitchen">
            <option>Kitchen</option>
            <option>Textiles</option>
          </SelectField>
          <TextField label="Optional" optional />
        </div>
        <TextareaField label="Textarea" placeholder="Longer text" />
        <CheckboxField label="Checkbox" defaultChecked />
      </Section>

      <Section title="Feedback">
        <Alert tone="info" title="Informational">
          Something worth knowing before you continue.
        </Alert>
        <Alert tone="success" title="Product saved">
          The catalog now shows the new price.
        </Alert>
        <Alert tone="warning" title="Three products need restocking">
          The storefront keeps selling them until you change the count.
        </Alert>
        <Alert
          tone="danger"
          title="The catalog service did not respond"
          actions={<Button variant="secondary">Try again</Button>}
        >
          Nothing was changed. Try again in a moment.
        </Alert>
      </Section>

      <Section title="Status">
        <div className={styles.row}>
          <Status tone="neutral">Draft</Status>
          <Status tone="info">Paid</Status>
          <Status tone="success">Shipped</Status>
          <Status tone="warning">Awaiting payment</Status>
          <Status tone="danger">Refunded</Status>
        </div>
      </Section>

      <Section title="Table — ready">
        <DataTable columns={columns} rows={rows} rowKey={(row) => row.id} caption="Two rows" />
      </Section>

      <Section title="Table — loading">
        <DataTable columns={columns} rows={[]} rowKey={(row) => row.id} status="loading" />
      </Section>

      <Section title="Table — empty">
        <DataTable
          columns={columns}
          rows={[]}
          rowKey={(row) => row.id}
          emptyTitle="Nothing here yet"
          emptyDescription="Rows appear once the shop has activity."
          emptyAction={<Button variant="primary">Add the first one</Button>}
        />
      </Section>

      <Section title="Table — error">
        <DataTable
          columns={columns}
          rows={[]}
          rowKey={(row) => row.id}
          status="error"
          onRetry={() => undefined}
        />
      </Section>

      <Section title="Page states">
        <StateBlock title="Loading" description="Fetching the record." />
        <StateBlock
          tone="error"
          title="This record could not be loaded"
          description="The service did not respond. Try again in a moment."
          action={<Button variant="secondary">Try again</Button>}
        />
      </Section>

      <Section title="Dialog">
        <div className={styles.row}>
          <Button variant="secondary" onClick={() => setDialogOpen(true)}>
            Open dialog
          </Button>
        </div>
        <Dialog
          open={dialogOpen}
          title="Discard this draft?"
          description="The product has not been published, so customers never saw it."
          onClose={() => setDialogOpen(false)}
          footer={
            <>
              <Button variant="ghost" onClick={() => setDialogOpen(false)}>
                Keep editing
              </Button>
              <Button variant="danger" onClick={() => setDialogOpen(false)}>
                Discard draft
              </Button>
            </>
          }
        >
          <TextField label="Reason" optional placeholder="Duplicate of MUG-CLY-300" />
        </Dialog>
      </Section>
    </PageStack>
  )
}
