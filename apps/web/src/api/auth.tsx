import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import * as identity from './identity'
import type { Auth, Permission } from './identity'
import { isApiError, onPasswordChangeRequired, onUnauthorized } from './http'

type AuthStatus = 'loading' | 'anonymous' | 'must-change-password' | 'authenticated' | 'unreachable'

type AuthState = {
  status: AuthStatus
  // Set on login and on a successful /me; absent while status is must-change-password from a
  // bootstrap 403, which carries no body — only {code, message}.
  user: Auth | undefined
  signIn: (email: string, password: string) => Promise<void>
  signOut: () => Promise<void>
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>
  hasPermission: (permission: Permission) => boolean
  reloadBootstrap: () => void
}

const AuthContext = createContext<AuthState | undefined>(undefined)

// The gateway caches the signed principal for SESSION_CACHE_TTL: /me can still answer with the
// pre-change principal for a few seconds after a real password change lands. Poll past that.
async function meAfterPasswordChange(): Promise<Auth> {
  const deadline = Date.now() + 12_000
  for (;;) {
    try {
      return await identity.getMe()
    } catch (cause) {
      const stale = isApiError(cause) && cause.code === 'password_change_required'
      if (!stale || Date.now() >= deadline) throw cause
      await new Promise((resolve) => setTimeout(resolve, 1000))
    }
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')
  const [user, setUser] = useState<Auth | undefined>(undefined)
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    onUnauthorized(() => {
      setUser(undefined)
      setStatus('anonymous')
    })
    onPasswordChangeRequired(() => {
      setStatus('must-change-password')
    })
  }, [])

  useEffect(() => {
    let active = true
    setStatus((current) => (current === 'loading' ? current : 'loading'))
    identity
      .getMe()
      .then((auth) => {
        if (!active) return
        setUser(auth)
        setStatus(auth.mustChangePassword ? 'must-change-password' : 'authenticated')
      })
      .catch((cause: unknown) => {
        if (!active) return
        if (isApiError(cause) && cause.code === 'password_change_required') {
          setStatus('must-change-password')
          return
        }
        if (isApiError(cause) && cause.code === 'unauthenticated') {
          setStatus('anonymous')
          return
        }
        setStatus('unreachable')
      })
    return () => {
      active = false
    }
  }, [attempt])

  const value = useMemo<AuthState>(
    () => ({
      status,
      user,
      async signIn(email, password) {
        const auth = await identity.login(email, password)
        setUser(auth)
        setStatus(auth.mustChangePassword ? 'must-change-password' : 'authenticated')
      },
      async signOut() {
        try {
          await identity.logout()
        } catch {
          // The client-side state clears regardless: a session already gone server-side isn't a failure here.
        } finally {
          setUser(undefined)
          setStatus('anonymous')
        }
      },
      async changePassword(currentPassword, newPassword) {
        await identity.changeMyPassword(currentPassword, newPassword)
        const auth = await meAfterPasswordChange()
        setUser(auth)
        setStatus('authenticated')
      },
      hasPermission(permission) {
        return user?.permissions.includes(permission) ?? false
      },
      reloadBootstrap() {
        setAttempt((value) => value + 1)
      },
    }),
    [status, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used within an AuthProvider')
  return context
}
