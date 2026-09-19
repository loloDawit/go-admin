import { expect, test } from '@playwright/test'
import { settled } from './select'

// The catalog has hundreds of seeded products, so the default 20-row page
// always leaves more than one page to page through.
// The controls are the subject, so the data is mocked: a seeded database may
// hold fewer than one page of products, and then nothing about paging can be
// exercised at all.
test('page X of Y is stated, and first/last jump to the ends', async ({ page }) => {
  const TOTAL = 95
  await page.route('**/api/v1/products*', async (route) => {
    const url = new URL(route.request().url())
    const size = Number(url.searchParams.get('pageSize') ?? 20)
    const current = Number(url.searchParams.get('page') ?? 1)
    const start = (current - 1) * size
    const items = Array.from({ length: Math.max(0, Math.min(size, TOTAL - start)) }, (_, i) => ({
      id: String(start + i + 1),
      sku: `SKU-${start + i + 1}`,
      title: `Product ${start + i + 1}`,
      description: '',
      status: 'active',
      priceMinor: 1000,
      currency: 'USD',
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    }))
    await route.fulfill({ json: { items, page: current, pageSize: size, total: TOTAL } })
  })

  await page.goto('/products')
  await settled(page)

  const lastPage = Math.ceil(TOTAL / 20)
  const pageLabel = page.getByText(/^Page \d+ of \d+$/)
  await expect(pageLabel).toHaveText(`Page 1 of ${lastPage}`)
  await expect(page.getByRole('button', { name: 'First page' })).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Previous' })).toBeDisabled()

  await page.getByRole('button', { name: 'Next' }).click()
  await settled(page)
  await expect(page).toHaveURL(/page=2/)
  await expect(pageLabel).toHaveText(`Page 2 of ${lastPage}`)

  await page.getByRole('button', { name: 'Last page' }).click()
  await settled(page)
  await expect(page).toHaveURL(new RegExp(`page=${lastPage}`))
  await expect(pageLabel).toHaveText(`Page ${lastPage} of ${lastPage}`)
  await expect(page.getByRole('button', { name: 'Next' })).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Last page' })).toBeDisabled()

  await page.getByRole('button', { name: 'First page' }).click()
  await settled(page)
  await expect(page).toHaveURL(/products$|page=1/)
  await expect(pageLabel).toHaveText(`Page 1 of ${lastPage}`)
})

test('a hidden column disappears from the table and stays hidden across a reload', async ({ page }) => {
  await page.goto('/products')
  await settled(page)

  await expect(page.getByRole('columnheader', { name: 'SKU', exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Columns' }).click()
  await page.getByRole('menuitemcheckbox', { name: 'SKU' }).click()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('columnheader', { name: 'SKU', exact: true })).toHaveCount(0)

  await page.reload()
  await settled(page)
  await expect(page.getByRole('columnheader', { name: 'SKU', exact: true })).toHaveCount(0)

  await page.getByRole('button', { name: 'Columns' }).click()
  await page.getByRole('menuitemcheckbox', { name: 'SKU' }).click()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('columnheader', { name: 'SKU', exact: true })).toBeVisible()
})

test('column visibility is kept per screen, not shared across tables', async ({ page }) => {
  await page.goto('/products')
  await settled(page)
  await page.getByRole('button', { name: 'Columns' }).click()
  await page.getByRole('menuitemcheckbox', { name: 'SKU' }).click()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('columnheader', { name: 'SKU', exact: true })).toHaveCount(0)

  await page.goto('/roles')
  await settled(page)
  await expect(page.getByRole('columnheader', { name: 'Role', exact: true })).toBeVisible()
})

test('the table header stays visible after scrolling its own rows', async ({ page }) => {
  await page.goto('/products')
  await settled(page)

  const scrollContainer = page.locator('[data-slot="table-container"]').locator('..')
  const header = page.getByRole('columnheader', { name: 'Product', exact: true })

  const before = await header.boundingBox()
  await scrollContainer.evaluate((el) => {
    el.scrollTop = 300
  })
  await expect
    .poll(() => scrollContainer.evaluate((el) => el.scrollTop))
    .toBeGreaterThan(0)
  const after = await header.boundingBox()

  expect(before).not.toBeNull()
  expect(after).not.toBeNull()
  expect(Math.abs((after?.y ?? 0) - (before?.y ?? 0))).toBeLessThan(2)
})
