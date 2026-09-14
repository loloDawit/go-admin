import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Alert, Button, TextField } from '../ui'
import styles from './Login.module.css'

export function Login() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [failed, setFailed] = useState(false)
  const [touched, setTouched] = useState(false)

  const emailError = touched && email.trim() === '' ? 'Enter the email you sign in with.' : undefined
  const passwordError = touched && password === '' ? 'Enter your password.' : undefined

  return (
    <div className={styles.page}>
      <div className={styles.panel}>
        <div className={styles.brand}>
          <h1 className={styles.brandName}>Northgate Supply</h1>
          <p className={styles.brandContext}>Staff sign-in</p>
        </div>

        {failed && (
          <Alert tone="danger" title="Those details did not match">
            Check the email and password, then try again.
          </Alert>
        )}

        <form
          className={styles.form}
          onSubmit={(event) => {
            event.preventDefault()
            setTouched(true)
            if (email.trim() === '' || password === '') return
            setSubmitting(true)
            setTimeout(() => {
              setSubmitting(false)
              if (password === 'wrong') {
                setFailed(true)
                return
              }
              navigate('/')
            }, 600)
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

        <p className={styles.footnote}>
          Sign-in is not connected yet. Any details take you to the dashboard.
        </p>
      </div>
    </div>
  )
}
