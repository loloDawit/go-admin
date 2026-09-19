import { expect, test } from '@playwright/test'

// A stat card's figure is driven by the card's own width via a CSS container
// query (@[250px]/card:text-3xl), not the viewport — this reads real computed
// style rather than asserting a class name, so it fails if the query regresses
// to a fixed size.
test('the KPI value scales with its own card width, not a fixed size', async ({ page }) => {
  await page.goto('/')
  const card = page.locator('[data-slot="card"]', { hasText: 'Orders to handle' })
  const title = card.locator('[data-slot="card-title"]')
  await expect(title).toBeVisible()

  await card.evaluate((el) => {
    ;(el as HTMLElement).style.width = '200px'
  })
  const narrow = await title.evaluate((el) => parseFloat(getComputedStyle(el).fontSize))

  await card.evaluate((el) => {
    ;(el as HTMLElement).style.width = '400px'
  })
  const wide = await title.evaluate((el) => parseFloat(getComputedStyle(el).fontSize))

  expect(wide).toBeGreaterThan(narrow)
})

test('a KPI still under load shows an em dash, not a stale or fabricated figure', async ({
  page,
}) => {
  await page.route('**/api/v1/reports/dashboard', async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 3000))
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ counts: {}, recent: [], revenue: [] }),
    })
  })
  await page.goto('/')
  const card = page.locator('[data-slot="card"]', { hasText: 'Orders to handle' })
  await expect(card.locator('[data-slot="card-title"]')).toHaveText('—')
})

// The report's revenue days are a period total (placed/paid/net per day); the
// three KPIs shown are a current backlog count (a stock, not a flow), so no
// prior-period figure exists to compare against — the card must render with
// no trend badge rather than inventing one.
test('a KPI with no comparable prior period carries no trend badge', async ({ page }) => {
  await page.goto('/')
  const card = page.locator('[data-slot="card"]', { hasText: 'Orders to handle' })
  await expect(card.locator('[data-slot="card-title"]')).not.toHaveText('—')
  await expect(card.locator('[data-slot="badge"]')).toHaveCount(0)
})
