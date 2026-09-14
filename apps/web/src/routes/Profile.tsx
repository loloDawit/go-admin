import { useState } from 'react'
import { Alert, Button, PageHeader, PageStack, Section, TextField } from '../ui'
import styles from './ProductForm.module.css'

export function Profile() {
  const [saved, setSaved] = useState(false)

  return (
    <PageStack>
      <PageHeader title="Profile" description="Your own details and sign-in." />

      {saved && (
        <Alert tone="success" title="Profile saved">
          The change is local until the identity service is connected.
        </Alert>
      )}

      <form
        className={styles.form}
        onSubmit={(event) => {
          event.preventDefault()
          setSaved(true)
        }}
      >
        <Section title="Details">
          <div className={styles.grid}>
            <TextField label="Name" defaultValue="Mara Lindqvist" />
            <TextField label="Email" type="email" defaultValue="mara@northgate.example" />
          </div>
        </Section>

        <Section title="Password" description="Changing it signs you out of other devices.">
          <div className={styles.grid}>
            <TextField label="Current password" type="password" autoComplete="current-password" />
            <TextField
              label="New password"
              type="password"
              autoComplete="new-password"
              help="At least 12 characters."
            />
          </div>
        </Section>

        <div className={styles.actions}>
          <Button type="submit" variant="primary">
            Save changes
          </Button>
        </div>
      </form>
    </PageStack>
  )
}
