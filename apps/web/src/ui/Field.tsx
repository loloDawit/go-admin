import { useId } from 'react'
import type {
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from 'react'
import styles from './Field.module.css'

type FieldShell = {
  label: string
  help?: string
  error?: string
  optional?: boolean
}

type ControlRender = (props: {
  id: string
  className: string
  'aria-describedby'?: string
  'aria-invalid'?: true
}) => ReactNode

function Field({ label, help, error, optional, render }: FieldShell & { render: ControlRender }) {
  const id = useId()
  const helpId = `${id}-help`
  const errorId = `${id}-error`
  const describedBy = [error ? errorId : '', help ? helpId : ''].filter(Boolean).join(' ')

  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={id}>
        {label}
        {optional && <span className={styles.optional}> (optional)</span>}
      </label>
      {render({
        id,
        className: `${styles.control} ${error ? styles.invalid : ''}`.trim(),
        'aria-describedby': describedBy || undefined,
        'aria-invalid': error ? true : undefined,
      })}
      {help && !error && (
        <p className={styles.help} id={helpId}>
          {help}
        </p>
      )}
      {error && (
        <p className={styles.error} id={errorId}>
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
      render={(props) => <input {...props} {...rest} />}
    />
  )
}

export function SelectField({
  label,
  help,
  error,
  optional,
  children,
  ...rest
}: FieldShell & SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <Field
      label={label}
      help={help}
      error={error}
      optional={optional}
      render={(props) => (
        <select {...props} className={`${props.className} ${styles.select}`} {...rest}>
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
      render={(props) => (
        <textarea {...props} className={`${props.className} ${styles.textarea}`} {...rest} />
      )}
    />
  )
}

export function CheckboxField({
  label,
  ...rest
}: { label: string } & InputHTMLAttributes<HTMLInputElement>) {
  const id = useId()
  return (
    <div className={`${styles.field} ${styles.checkboxField}`}>
      <input type="checkbox" id={id} className={styles.checkbox} {...rest} />
      <label className={styles.checkboxLabel} htmlFor={id}>
        {label}
      </label>
    </div>
  )
}
