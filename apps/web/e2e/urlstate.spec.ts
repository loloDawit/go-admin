import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

function unique(): string {
  return String(Date.now())
}

async function activeProduct(page: Page, title: string): Promise<void> {
  await page.goto('/products/new')
  await page.getByRole('textbox', { name: 'SKU' }).fill(`URL-${unique()}`)
  await page.getByRole('textbox', { name: 'Title' }).fill(title)
  await page.getByRole('textbox', { name: 'Price' }).fill('3.00')
  await page.getByRole('button', { name: 'Create product' }).click()
  await page.getByRole('button', { name: 'Activate' }).click()
  await expect(page.getByText('Active', { exact: true })).toBeVisible()
}

test('a searched product list survives a reload', async ({ page }) => {
  const title = `Recoverable ${unique()}`
  await activeProduct(page, title)

  await page.goto('/products')
  await page.getByRole('searchbox', { name: 'Search' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await expect(page).toHaveURL(/[?&]q=/)

  await page.reload()
  await expect(page.getByRole('searchbox', { name: 'Search' })).toHaveValue(title)
  await expect(page.getByRole('link', { name: title })).toBeVisible()
})

test('a filtered list is a link someone else can open', async ({ page, context }) => {
  await page.goto('/products')
  await page.getByLabel('Status').selectOption('archived')
  await expect(page).toHaveURL(/[?&]status=archived/)

  const shared = page.url()
  const other = await context.newPage()
  await other.goto(shared)
  await expect(other.getByLabel('Status')).toHaveValue('archived')
  await other.close()
})

test('a default never appears in the query string', async ({ page }) => {
  await page.goto('/products')
  await page.getByLabel('Status').selectOption('active')
  await expect(page).toHaveURL(/\/products\?status=active$/)

  await page.getByLabel('Status').selectOption('')
  await expect(page).toHaveURL(/\/products$/)
})

test('back undoes the last filter change', async ({ page }) => {
  await page.goto('/products')
  await page.getByLabel('Status').selectOption('active')
  await expect(page).toHaveURL(/status=active/)

  await page.goBack()
  await expect(page).toHaveURL(/\/products$/)
  await expect(page.getByLabel('Status')).toHaveValue('')
})

// A status the client did not generate must never reach the service, which
// answers 422 for an unknown enum value and would turn a mistyped link into an
// error page.
test('an unrecognised status in the URL is dropped, not forwarded', async ({ page }) => {
  await page.goto('/products?status=banana')
  await expect(page.locator('[aria-busy="true"]')).toHaveCount(0)
  await expect(page.getByLabel('Status')).toHaveValue('')
  await expect(page.getByText('could not be loaded')).toHaveCount(0)
})

test('an order status filter is in the URL and survives a reload', async ({ page }) => {
  await page.goto('/orders')
  await page.getByLabel('Status').selectOption('packed')
  await expect(page).toHaveURL(/\/orders\?status=packed$/)

  await page.reload()
  await expect(page.getByLabel('Status')).toHaveValue('packed')
})

test('an unrecognised order status in the URL is dropped', async ({ page }) => {
  await page.goto('/orders?status=elsewhere')
  await expect(page.locator('[aria-busy="true"]')).toHaveCount(0)
  await expect(page.getByLabel('Status')).toHaveValue('')
  await expect(page.getByText('could not be loaded')).toHaveCount(0)
})

// Asserting the URL alone would pass against a screen that ignores it. Previous
// is disabled on page one and enabled beyond it, so it observes the page in effect.
test('the customer list page is in the URL', async ({ page }) => {
  await page.goto('/customers')
  await expect(page.getByRole('button', { name: 'Previous' })).toBeDisabled()

  await page.goto('/customers?page=2')
  await expect(page.locator('[aria-busy="true"]')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Previous' })).toBeEnabled()
})
