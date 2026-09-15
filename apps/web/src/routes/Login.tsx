import { useEffect, useState } from 'react'
import type { Location } from 'react-router-dom'
import { useLocation, useNavigate } from 'react-router-dom'
import { Alert, Button, TextField } from '../ui'
import { useAuth } from '../api/auth'
import { isApiError } from '../api/http'
import styles from './Login.module.css'

export function Login() {
  const auth = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string>()
  const [touched, setTouched] = useState(false)

  useEffect(() => {
    if (auth.status === 'must-change-password') {
      navigate('/change-password', { replace: true })
      return
    }
    if (auth.status === 'authenticated') {
      const from = (location.state as { from?: Location } | null)?.from
      navigate(from?.pathname ?? '/', { replace: true })
    }
  }, [auth.status, location.state, navigate])

  const emailError = touched && email.trim() === '' ? 'Enter the email you sign in with.' : undefined
  const passwordError = touched && password === '' ? 'Enter your password.' : undefined

  return (
    <div className={styles.page}>
      <div className={styles.panel}>
        <div className={styles.brand}>
          <h1 className={styles.brandName}>Northgate Supply</h1>
          <p className={styles.brandContext}>Staff sign-in</p>
        </div>

        {error && (
          <Alert tone="danger" title="Sign-in failed">
            {error}
          </Alert>
        )}

        <form
          className={styles.form}
          onSubmit={(event) => {
            event.preventDefault()
            setTouched(true)
            if (email.trim() === '' || password === '') return
            setSubmitting(true)
            setError(undefined)
            auth
              .signIn(email, password)
              .catch((cause: unknown) => {
                setError(isApiError(cause) ? cause.message : 'Something went wrong.')
              })
              .finally(() => setSubmitting(false))
          }}
        >
          <TextField
            label="Email"
            type="email"
            autoComplete="username"
            value={email}
            error={emailError}
            onChange={(event) => setEmail(event.target.value)}
            onBlur={() => setTouched(true)}
          />
          <TextField
            label="Password"
            type="password"
            autoComplete="current-password"
            value={password}
            error={passwordError}
            onChange={(event) => setPassword(event.target.value)}
            onBlur={() => setTouched(true)}
          />
          <Button type="submit" variant="primary" block loading={submitting}>
            Sign in
          </Button>
        </form>
      </div>
    </div>
  )
}
