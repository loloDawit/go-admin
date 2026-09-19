import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'
import { settled } from './select'

function day(n: number): string {
  const date = new Date(Date.UTC(2026, 0, 1))
  date.setUTCDate(date.getUTCDate() + n)
  return date.toISOString().slice(0, 10)
}

function revenueDays(count: number, currency = 'USD') {
  return Array.from({ length: count }, (_, i) => ({
    day: day(i),
    currency,
    placedCount: 1,
    paidCount: 1,
    recognisedMinor: 10000 + i * 137,
    refundedMinor: 0,
    netMinor: 10000 + i * 137,
  }))
}

async function mockDashboard(page: Page, revenue: unknown[]) {
  await page.route('**/api/v1/reports/dashboard', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ counts: {}, recent: [], revenue }),
    })
  })
}

function chart(page: Page) {
  return page.locator('[data-slot="card"]', { hasText: 'Revenue' })
}

test('the chart renders one point per revenue day, and the range toggle changes the count', async ({
  page,
}) => {
  await mockDashboard(page, revenueDays(95))
  await page.goto('/')
  await settled(page)

  // widest range (90) out of 95 available days is the default
  await expect(chart(page).locator('.recharts-dot')).toHaveCount(90)

  await chart(page).getByRole('radio', { name: 'Last 7 days' }).click()
  await expect(chart(page).locator('.recharts-dot')).toHaveCount(7)

  await chart(page).getByRole('radio', { name: 'Last 30 days' }).click()
  await expect(chart(page).locator('.recharts-dot')).toHaveCount(30)
})

test('a single revenue day still shows a visible point, not a zero-width line', async ({ page }) => {
  await mockDashboard(page, revenueDays(1))
  await page.goto('/')
  await settled(page)

  await expect(chart(page).locator('.recharts-dot')).toHaveCount(1)
})

test('no revenue at all shows an empty state, not a blank chart', async ({ page }) => {
  await mockDashboard(page, [])
  await page.goto('/')
  await settled(page)

  await expect(chart(page).getByText('No revenue recorded yet')).toBeVisible()
  await expect(chart(page).locator('.recharts-dot')).toHaveCount(0)
})

test('a failed report shows an error state on the chart, not a stale or blank one', async ({ page }) => {
  await page.route('**/api/v1/reports/dashboard', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ code: 'internal', message: 'boom' }),
    })
  })
  await page.goto('/')
  await settled(page)

  await expect(chart(page).getByText('Revenue is unavailable')).toBeVisible()
})

test("hovering the chart shows the day's revenue formatted as money, not a raw minor-unit integer", async ({
  page,
}) => {
  await mockDashboard(page, revenueDays(7))
  await page.goto('/')
  await settled(page)

  const plot = chart(page).locator('.recharts-surface').first()
  const box = await plot.boundingBox()
  if (!box) throw new Error('chart did not render a plot area')
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)

  await expect(chart(page).getByText(/^\$\d[\d,]*\.\d{2}$/)).toBeVisible()
})
