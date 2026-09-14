import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Alert, Button, TextField } from '../ui'
import { useAuth } from '../api/auth'
import { isApiError } from '../api/http'
import styles from './ChangePassword.module.css'

export function ChangePassword() {
  const auth = useAuth()
  const navigate = useNavigate()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string>()

  useEffect(() => {
    if (auth.status === 'anonymous') navigate('/login', { replace: true })
  }, [auth.status, navigate])

  if (auth.status !== 'must-change-password' && auth.status !== 'authenticated') return null

  const required = auth.status === 'must-change-password'

  return (
    <div className={styles.page}>
      <div className={styles.panel}>
        <div className={styles.heading}>
          <h1 className={styles.title}>Change your password</h1>
          <p className={styles.description}>
            {required
              ? 'Set a password of your own before continuing.'
              : 'Choose a new password for this account.'}
          </p>
        </div>

        {error && (
          <Alert tone="danger" title="Could not change your password">
            {error}
          </Alert>
        )}

        <form
          className={styles.form}
          onSubmit={(event) => {
            event.preventDefault()
            setSubmitting(true)
            setError(undefined)
            auth
              .changePassword(currentPassword, newPassword)
              .then(() => navigate('/', { replace: true }))
              .catch((cause: unknown) => {
                setError(isApiError(cause) ? cause.message : 'Something went wrong.')
              })
              .finally(() => setSubmitting(false))
          }}
        >
          <div className={styles.grid}>
            <TextField
              label="Current password"
              type="password"
              autoComplete="current-password"
              value={currentPassword}
              onChange={(event) => setCurrentPassword(event.target.value)}
              required
            />
            <TextField
              label="New password"
              type="password"
              autoComplete="new-password"
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
              help="At least 12 characters."
              required
            />
          </div>

          <div className={styles.actions}>
            <Button type="submit" variant="primary" loading={submitting}>
              Change password
            </Button>
            {!required && (
              <Button variant="ghost" onClick={() => navigate(-1)}>
                Cancel
              </Button>
            )}
            <Button
              variant="ghost"
              onClick={() => {
                void auth.signOut().then(() => navigate('/login', { replace: true }))
              }}
            >
              Sign out instead
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}
