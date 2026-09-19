export function StatCard({
  label,
  value,
  context,
}: {
  label: string
  value: number | string | undefined
  context?: string
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <p className="text-caption text-muted-foreground">{label}</p>
      <p className="mt-1 text-display font-semibold tabular-nums">
        {value === undefined ? '—' : value}
      </p>
      {context && <p className="mt-1 text-caption text-muted-foreground">{context}</p>}
    </div>
  )
}
