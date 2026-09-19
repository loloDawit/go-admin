import { useMemo, useState } from 'react'
import { Area, AreaChart, CartesianGrid, XAxis } from 'recharts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/ui/shadcn/card'
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/ui/shadcn/chart'
import type { ChartConfig } from '@/ui/shadcn/chart'
import { ToggleGroup, ToggleGroupItem } from '@/ui/shadcn/toggle-group'
import { Skeleton } from '@/ui/shadcn/skeleton'
import { EmptyState } from '@/ui/EmptyState'
import { formatDate, formatMoney } from '@/api/format'
import type { RevenueDay } from '@/api/reports'

const RANGES = [
  { value: '7', label: 'Last 7 days', days: 7 },
  { value: '30', label: 'Last 30 days', days: 30 },
  { value: '90', label: 'Last 90 days', days: 90 },
] as const

type RangeValue = (typeof RANGES)[number]['value']

const chartConfig = {
  net: { label: 'Net revenue', color: 'var(--color-primary)' },
} satisfies ChartConfig

// The report has no shop-level currency: infer it from the data itself rather
// than guessing. Ties break alphabetically so the choice is deterministic.
function dominantCurrency(revenue: RevenueDay[]): string | undefined {
  const counts = new Map<string, number>()
  for (const day of revenue) counts.set(day.currency, (counts.get(day.currency) ?? 0) + 1)

  let best: string | undefined
  let bestCount = -1
  for (const [currency, count] of counts) {
    if (count > bestCount || (count === bestCount && currency < (best as string))) {
      best = currency
      bestCount = count
    }
  }
  return best
}

export function RevenueChart({
  revenue,
  status,
  errorDescription,
}: {
  revenue: RevenueDay[]
  status: 'loading' | 'ready' | 'error'
  errorDescription?: string
}) {
  const [range, setRange] = useState<RangeValue>('90')

  const currency = useMemo(() => dominantCurrency(revenue), [revenue])
  const currencyCount = useMemo(() => new Set(revenue.map((day) => day.currency)).size, [revenue])

  const series = useMemo(() => {
    if (!currency) return []
    const days = revenue
      .filter((day) => day.currency === currency)
      .sort((a, b) => a.day.localeCompare(b.day))
    const window = RANGES.find((option) => option.value === range)!.days
    return days.slice(-window)
  }, [revenue, currency, range])

  return (
    <Card>
      <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <CardTitle>Revenue</CardTitle>
          <CardDescription>
            {currency
              ? `Net revenue recognised each day, in ${currency}${
                  currencyCount > 1 ? ' — the shop’s most active currency' : ''
                }.`
              : 'Net revenue recognised each day.'}
          </CardDescription>
        </div>
        <ToggleGroup
          type="single"
          variant="outline"
          value={range}
          onValueChange={(value) => value && setRange(value as RangeValue)}
          className="flex-wrap justify-end"
        >
          {RANGES.map((option) => (
            <ToggleGroupItem key={option.value} value={option.value}>
              {option.label}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </CardHeader>
      <CardContent className="px-2 pt-4 sm:px-6 sm:pt-6" aria-busy={status === 'loading' || undefined}>
        {status === 'loading' && <Skeleton className="h-[250px] w-full" />}
        {status === 'error' && (
          <EmptyState
            title="Revenue is unavailable"
            description={errorDescription ?? 'The service did not respond. Try again in a moment.'}
          />
        )}
        {status === 'ready' && series.length === 0 && (
          <EmptyState
            title="No revenue recorded yet"
            description="A day appears here once an order placed that day is paid."
          />
        )}
        {status === 'ready' && series.length > 0 && currency && (
          <ChartContainer config={chartConfig} className="aspect-auto h-[250px] w-full">
            <AreaChart data={series} margin={{ left: 12, right: 12 }}>
              <defs>
                <linearGradient id="revenue-net-fill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--color-primary)" stopOpacity={0.8} />
                  <stop offset="95%" stopColor="var(--color-primary)" stopOpacity={0.1} />
                </linearGradient>
              </defs>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="day"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                minTickGap={32}
                tickFormatter={(value: string) => formatDate(`${value}T00:00:00Z`)}
              />
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    labelFormatter={(value) => formatDate(`${value}T00:00:00Z`)}
                    formatter={(value) => formatMoney(Number(value), currency)}
                  />
                }
              />
              <Area
                dataKey="netMinor"
                name="net"
                type="natural"
                fill="url(#revenue-net-fill)"
                stroke="var(--color-primary)"
                dot={{ r: 3, fill: 'var(--color-primary)', strokeWidth: 0 }}
                activeDot={{ r: 4 }}
              />
            </AreaChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  )
}
