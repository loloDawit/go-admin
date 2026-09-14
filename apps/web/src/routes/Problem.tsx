import { Link } from 'react-router-dom'
import { PageStack, StateBlock } from '../ui'

export function NotFound() {
  return (
    <PageStack>
      <StateBlock
        title="That page does not exist"
        description="The link may be out of date. Everything else is still where you left it."
        action={<Link to="/">Back to the dashboard</Link>}
      />
    </PageStack>
  )
}

export function Forbidden() {
  return (
    <PageStack>
      <StateBlock
        tone="error"
        title="You do not have access to this"
        description="Your role does not include this permission. An owner can change that on the Roles screen."
        action={<Link to="/">Back to the dashboard</Link>}
      />
    </PageStack>
  )
}

export function ServerError() {
  return (
    <PageStack>
      <StateBlock
        tone="error"
        title="Something went wrong on our side"
        description="The request did not complete. Nothing was changed. Try again in a moment."
        action={<Link to="/">Back to the dashboard</Link>}
      />
    </PageStack>
  )
}
