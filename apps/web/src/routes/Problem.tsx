import { Link } from 'react-router-dom'
import { PageHeader, PageStack } from '../ui'

function Problem({ title, description }: { title: string; description: string }) {
  return (
    <PageStack>
      <PageHeader
        title={title}
        description={description}
        actions={<Link to="/">Back to the dashboard</Link>}
      />
    </PageStack>
  )
}

export function NotFound() {
  return (
    <Problem
      title="That page does not exist"
      description="The link may be out of date. Everything else is still where you left it."
    />
  )
}

export function Forbidden() {
  return (
    <Problem
      title="You do not have access to this"
      description="Your role does not include this permission. An owner can grant it on the Roles screen."
    />
  )
}

export function ServerError() {
  return (
    <Problem
      title="Something went wrong on our side"
      description="The request did not complete and nothing was changed. Try again in a moment."
    />
  )
}
