import type { ReactNode } from 'react'
import { EmptyState } from './EmptyState'

export function StateBlock({
  tone = 'neutral',
  title,
  description,
  action,
}: {
  tone?: 'neutral' | 'error'
  title: string
  description?: string
  action?: ReactNode
  centered?: boolean
}) {
  const block = <EmptyState title={title} description={description} action={action} />
  // A failed state has to be announced, not just drawn.
  return tone === 'error' ? <div role="alert">{block}</div> : block
}
