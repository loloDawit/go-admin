import { useState } from 'react'
import {
  Alert,
  Button,
  CheckboxField,
  PageHeader,
  PageStack,
  Section,
  SelectField,
  TextField,
} from '../ui'
import styles from './ProductForm.module.css'

export function Settings() {
  const [saved, setSaved] = useState(false)

  return (
    <PageStack>
      <PageHeader title="Settings" description="How the shop behaves for everyone." />

      {saved && (
        <Alert tone="success" title="Settings saved">
          The change is local until the services are connected.
        </Alert>
      )}

      <form
        className={styles.form}
        onSubmit={(event) => {
          event.preventDefault()
          setSaved(true)
        }}
      >
        <Section title="Shop">
          <div className={styles.grid}>
            <TextField label="Shop name" defaultValue="Northgate Supply" />
            <SelectField label="Currency" defaultValue="GBP">
              <option>GBP</option>
              <option>EUR</option>
              <option>USD</option>
            </SelectField>
            <SelectField label="Time zone" defaultValue="Europe/London">
              <option>Europe/London</option>
              <option>Europe/Berlin</option>
              <option>UTC</option>
            </SelectField>
          </div>
        </Section>

        <Section title="Fulfilment">
          <div className={styles.grid}>
            <TextField label="Low stock threshold" inputMode="numeric" defaultValue="3" />
            <TextField label="Dispatch cut-off" defaultValue="15:00" />
          </div>
          <CheckboxField label="Email the team when an order has waited over a day" defaultChecked />
        </Section>

        <Section title="Danger zone" description="These cannot be undone from this screen.">
          <Alert tone="danger" title="Closing the shop hides the storefront">
            Customers see a holding page and no new orders arrive.
          </Alert>
          <div>
            <Button variant="danger">Close the shop</Button>
          </div>
        </Section>

        <div className={styles.actions}>
          <Button type="submit" variant="primary">
            Save settings
          </Button>
        </div>
      </form>
    </PageStack>
  )
}
