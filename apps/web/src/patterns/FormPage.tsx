import type { ReactNode } from 'react'
import { Alert, Button } from '../ui'

// A form bounds its own measure; a table does not. Fields past about 46rem
// stop reading as one column of work.
export function FormPage({
  failure,
  onSubmit,
  submitLabel,
  saving,
  onCancel,
  children,
}: {
  failure?: string
  onSubmit: () => void
  submitLabel: string
  saving?: boolean
  onCancel: () => void
  children: ReactNode
}) {
  return (
    <form
      className="flex max-w-[46rem] flex-col gap-5"
      onSubmit={(event) => {
        event.preventDefault()
        onSubmit()
      }}
    >
      {failure && <Alert tone="danger" title={failure} />}

      {children}

      <div className="flex gap-2 border-t border-border pt-4">
        <Button type="submit" variant="primary" loading={saving}>
          {submitLabel}
        </Button>
        <Button variant="ghost" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  )
}

// Fields wrap into as many columns as the measure allows, so a three-field row
// becomes one column on a phone without each screen saying how.
export function FieldGrid({ children }: { children: ReactNode }) {
  return (
    <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(13rem,1fr))]">
      {children}
    </div>
  )
}
