import { useId } from 'react'
import type {
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from 'react'
import { Checkbox } from '@/ui/shadcn/checkbox'
import { Input } from '@/ui/shadcn/input'
import { Label } from '@/ui/shadcn/label'
import { Textarea } from '@/ui/shadcn/textarea'

type FieldShell = {
  label: string
  help?: string
  error?: string
  optional?: boolean
}

type ControlRender = (props: {
  id: string
  'aria-describedby'?: string
  'aria-invalid'?: true
}) => ReactNode

function Field({ label, help, error, optional, render }: FieldShell & { render: ControlRender }) {
  const id = useId()
  const helpId = `${id}-help`
  const errorId = `${id}-error`
  const describedBy = [error ? errorId : '', help ? helpId : ''].filter(Boolean).join(' ')

  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id}>
        {label}
        {optional && <span className="font-normal text-subtle-foreground"> (optional)</span>}
      </Label>
      {render({
        id,
        'aria-describedby': describedBy || undefined,
        'aria-invalid': error ? true : undefined,
      })}
      {help && !error && (
        <p className="text-caption text-muted-foreground" id={helpId}>
          {help}
        </p>
      )}
      {error && (
        <p className="text-caption text-destructive" id={errorId}>
          {error}
        </p>
      )}
    </div>
  )
}

export function TextField({
  label,
  help,
  error,
  optional,
  ...rest
}: FieldShell & InputHTMLAttributes<HTMLInputElement>) {
  return (
    <Field
      label={label}
      help={help}
      error={error}
      optional={optional}
      render={(props) => <Input {...props} {...rest} />}
    />
  )
}

// A native select, deliberately: this is a form control bound to a value, and
// the browser's own picker is better on a phone than any listbox we would
// build. The filter bars use the Radix listbox, which is a different job.
export function SelectField({
  label,
  help,
  error,
  optional,
  children,
  className,
  ...rest
}: FieldShell & SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <Field
      label={label}
      help={help}
      error={error}
      optional={optional}
      render={(props) => (
        <select
          {...props}
          {...rest}
          className={`h-8 w-full rounded-md border border-input bg-card px-2.5 text-body shadow-xs outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 ${className ?? ''}`}
        >
          {children}
        </select>
      )}
    />
  )
}

export function TextareaField({
  label,
  help,
  error,
  optional,
  ...rest
}: FieldShell & TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <Field
      label={label}
      help={help}
      error={error}
      optional={optional}
      render={(props) => <Textarea rows={4} {...props} {...rest} />}
    />
  )
}

export function CheckboxField({
  label,
  checked,
  onChange,
  disabled,
}: { label: string } & InputHTMLAttributes<HTMLInputElement>) {
  const id = useId()
  return (
    <div className="flex items-center gap-2">
      <Checkbox
        id={id}
        checked={checked}
        disabled={disabled}
        onCheckedChange={(next) =>
          onChange?.({
            target: { checked: next === true },
          } as React.ChangeEvent<HTMLInputElement>)
        }
      />
      <Label htmlFor={id}>{label}</Label>
    </div>
  )
}
