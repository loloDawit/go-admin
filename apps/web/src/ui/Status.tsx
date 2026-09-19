import { StatusBadge } from './StatusBadge'
import type { StatusTone } from './tone'

export function Status({ tone, children }: { tone: StatusTone; children: string }) {
  return <StatusBadge tone={tone}>{children}</StatusBadge>
}
