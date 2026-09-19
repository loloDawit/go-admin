import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/ui/shadcn/button'
import { isApiError } from '../api/http'

export type BulkAction = {
  label: string
  onClick: () => void
  disabled?: boolean
  // Shown beside a disabled action so it reads as gated, not missing.
  reason?: string
}

export type BulkProgress = {
  done: number
  total: number
}

export type BulkOutcome<T> = {
  ok: T[]
  failed: { row: T; message: string }[]
}

// Sequential, not Promise.all: progress has to be reportable row by row, and
// the per-item endpoint is the one that exists — there is no batch endpoint
// to fire instead.
export async function runBulk<T>(
  rows: T[],
  action: (row: T) => Promise<unknown>,
  onProgress: (done: number) => void,
): Promise<BulkOutcome<T>> {
  const ok: T[] = []
  const failed: { row: T; message: string }[] = []
  for (const row of rows) {
    try {
      await action(row)
      ok.push(row)
    } catch (cause) {
      failed.push({ row, message: isApiError(cause) ? cause.message : 'Something went wrong.' })
    }
    onProgress(ok.length + failed.length)
  }
  return { ok, failed }
}

// Never claims a success it did not verify: a partial result names the
// failure by count in the headline, and by row in the description, rather
// than folding it into a single "done" toast.
export function reportBulk<T>(outcome: BulkOutcome<T>, verb: string, name: (row: T) => string): void {
  const { ok, failed } = outcome
  if (failed.length === 0) {
    toast.success(`${ok.length} ${verb}`)
    return
  }
  const first = failed[0]
  toast.error(`${ok.length} ${verb}, ${failed.length} failed`, {
    description: first ? `${name(first.row)}: ${first.message}` : undefined,
  })
}

export function BulkBar({
  count,
  actions,
  onClear,
  progress,
}: {
  count: number
  actions: BulkAction[]
  onClear: () => void
  progress?: BulkProgress
}) {
  const busy = progress !== undefined
  return (
    <div
      role="toolbar"
      aria-label="Bulk actions"
      className="flex flex-wrap items-center gap-4 border-b border-border bg-muted px-4 py-2"
    >
      <span className="text-label text-foreground font-medium" aria-live="polite">
        {progress ? `Working: ${progress.done} of ${progress.total}` : `${count} selected`}
      </span>
      {busy && <Loader2 className="size-3.5 animate-spin text-muted-foreground" aria-hidden />}
      <div className="flex flex-wrap items-center gap-3">
        {actions.map((action) => (
          <span key={action.label} className="inline-flex items-center gap-1.5">
            <Button
              variant="secondary"
              size="sm"
              disabled={busy || action.disabled}
              onClick={action.onClick}
            >
              {action.label}
            </Button>
            {action.disabled && action.reason && (
              <span className="text-caption text-muted-foreground">{action.reason}</span>
            )}
          </span>
        ))}
      </div>
      <Button variant="ghost" size="sm" disabled={busy} onClick={onClear} className="ml-auto">
        Clear
      </Button>
    </div>
  )
}
