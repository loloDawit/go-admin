import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../api/auth'
import { Button, StateBlock } from '../ui'

export function RequireAuth() {
  const auth = useAuth()
  const location = useLocation()

  if (auth.status === 'loading') return null

  if (auth.status === 'unreachable') {
    return (
      <div className="grid min-h-dvh place-items-center bg-canvas p-6">
        <StateBlock
          tone="error"
          centered
          title="The service did not respond"
          description="Check your connection, then try again."
          action={
            <Button variant="secondary" onClick={auth.reloadBootstrap}>
              Try again
            </Button>
          }
        />
      </div>
    )
  }

  if (auth.status === 'anonymous') {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  if (auth.status === 'must-change-password') {
    return <Navigate to="/change-password" replace />
  }

  return <Outlet />
}
