import { useNavigate } from 'react-router-dom'
import { FileQuestion, ServerCrash, ShieldOff } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Button, EmptyState } from '../ui'

function Problem({
  icon,
  title,
  description,
}: {
  icon: LucideIcon
  title: string
  description: string
}) {
  const navigate = useNavigate()
  return (
    <div className="grid min-h-[60vh] place-items-center">
      <div className="rounded-lg border border-border bg-card">
        <EmptyState
          icon={icon}
          as="h1"
          title={title}
          description={description}
          action={<Button variant="primary" onClick={() => navigate('/')}>Back to the dashboard</Button>}
        />
      </div>
    </div>
  )
}

export function NotFound() {
  return (
    <Problem
      icon={FileQuestion}
      title="That page does not exist"
      description="The link may be out of date. Everything else is still where you left it."
    />
  )
}

export function Forbidden() {
  return (
    <Problem
      icon={ShieldOff}
      title="You do not have access to this"
      description="Your role does not include this permission. An owner can grant it on the Roles screen."
    />
  )
}

export function ServerError() {
  return (
    <Problem
      icon={ServerCrash}
      title="Something went wrong on our side"
      description="The request did not complete and nothing was changed. Try again in a moment."
    />
  )
}
