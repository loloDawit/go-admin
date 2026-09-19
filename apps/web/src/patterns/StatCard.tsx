import { TrendingDownIcon, TrendingUpIcon } from 'lucide-react'
import { Badge } from '../ui/shadcn/badge'
import { Card, CardAction, CardDescription, CardFooter, CardHeader, CardTitle } from '../ui/shadcn/card'

export type StatCardTrend = {
  direction: 'up' | 'down'
  label: string
}

export function StatCard({
  label,
  value,
  context,
  trend,
}: {
  label: string
  value: number | string | undefined
  context?: string
  trend?: StatCardTrend
}) {
  const TrendIcon = trend?.direction === 'down' ? TrendingDownIcon : TrendingUpIcon

  return (
    <Card className="@container/card">
      <CardHeader>
        <CardDescription>{label}</CardDescription>
        <CardTitle className="text-2xl font-semibold tabular-nums @[250px]/card:text-3xl">
          {value === undefined ? '—' : value}
        </CardTitle>
        {trend && (
          <CardAction>
            <Badge variant="outline">
              <TrendIcon />
              {trend.label}
            </Badge>
          </CardAction>
        )}
      </CardHeader>
      {context && (
        <CardFooter className="flex-col items-start gap-1.5 text-sm text-muted-foreground">
          {context}
        </CardFooter>
      )}
    </Card>
  )
}
