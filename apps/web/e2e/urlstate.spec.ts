import { expect, test } from '@playwright/test'
import { chooseOption, expectChosen, settled } from './select'
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
  await chooseOption(page, 'Status', 'Archived')
  await expect(page).toHaveURL(/[?&]status=archived/)

  const shared = page.url()
  const other = await context.newPage()
  await other.goto(shared)
  await expect(other.getByRole('combobox', { name: 'Status' })).toHaveText('Archived')
  await other.close()
})

test('a default never appears in the query string', async ({ page }) => {
  await page.goto('/products')
  await chooseOption(page, 'Status', 'Active')
  await expect(page).toHaveURL(/\/products\?status=active$/)

  await chooseOption(page, 'Status', 'Draft and active')
  await expect(page).toHaveURL(/\/products$/)
})

test('back undoes the last filter change', async ({ page }) => {
  await page.goto('/products')
  await chooseOption(page, 'Status', 'Active')
  await expect(page).toHaveURL(/status=active/)

  await page.goBack()
  await expect(page).toHaveURL(/\/products$/)
  await expectChosen(page, 'Status', 'Draft and active')
})

// A status the client did not generate must never reach the service, which
// answers 422 for an unknown enum value and would turn a mistyped link into an
// error page.
test('an unrecognised status in the URL is dropped, not forwarded', async ({ page }) => {
  await page.goto('/products?status=banana')
  await settled(page)
  await expectChosen(page, 'Status', 'Draft and active')
  await expect(page.getByText('could not be loaded')).toHaveCount(0)
})

test('an order status filter is in the URL and survives a reload', async ({ page }) => {
  await page.goto('/orders')
  await chooseOption(page, 'Status', 'Packed')
  await expect(page).toHaveURL(/\/orders\?status=packed$/)

  await page.reload()
  await expectChosen(page, 'Status', 'Packed')
})

test('an unrecognised order status in the URL is dropped', async ({ page }) => {
  await page.goto('/orders?status=elsewhere')
  await settled(page)
  await expectChosen(page, 'Status', 'All statuses')
  await expect(page.getByText('could not be loaded')).toHaveCount(0)
})

// Asserting the URL alone would pass against a screen that ignores it. Previous
// is disabled on page one and enabled beyond it, so it observes the page in effect.
test('the customer list page is in the URL', async ({ page }) => {
  const stamp = unique()
  await page.goto('/customers')
  await page.getByRole('button', { name: 'Add customer' }).first().click()
  await page.getByRole('textbox', { name: 'Name' }).fill(`Paged ${stamp}`)
  await page.getByRole('textbox', { name: 'Email' }).fill(`${stamp}@example.com`)
  await page.getByRole('dialog').getByRole('button', { name: 'Add customer' }).click()
  await expect(page.getByRole('link', { name: `Paged ${stamp}` })).toBeVisible()

  await expect(page.getByRole('button', { name: 'Previous' })).toBeDisabled()

  await page.goto('/customers?page=2')
  await settled(page)
  await expect(page.getByRole('button', { name: 'Previous' })).toBeEnabled()
})

test('a product column sorts, says so, and cycles back off', async ({ page }) => {
  await page.goto('/products')
  const header = page.getByRole('columnheader', { name: 'Product' })

  await header.getByRole('button').click()
  await expect(page).toHaveURL(/\/products\?sort=title$/)
  await expect(header).toHaveAttribute('aria-sort', 'ascending')

  await header.getByRole('button').click()
  await expect(page).toHaveURL(/\/products\?sort=-title$/)
  await expect(header).toHaveAttribute('aria-sort', 'descending')

  await header.getByRole('button').click()
  await expect(page).toHaveURL(/\/products$/)
  await expect(header).toHaveAttribute('aria-sort', 'none')
})

test('a sorted list is recoverable from its URL', async ({ page }) => {
  await page.goto('/products?sort=-price')
  await settled(page)
  await expect(page.getByRole('columnheader', { name: 'Price' })).toHaveAttribute(
    'aria-sort',
    'descending',
  )
})

test('an unrecognised sort in the URL is dropped', async ({ page }) => {
  await page.goto('/products?sort=whatever')
  await settled(page)
  await expect(page.getByText('could not be loaded')).toHaveCount(0)
})

test('orders sort by the columns their service allowlists', async ({ page }) => {
  await page.goto('/orders')
  await page.getByRole('columnheader', { name: 'Total' }).getByRole('button').click()
  await expect(page).toHaveURL(/\/orders\?sort=total_minor$/)

  await page.reload()
  await expect(page.getByRole('columnheader', { name: 'Total' })).toHaveAttribute(
    'aria-sort',
    'ascending',
  )
})

// A column the service will not sort by must not offer to.
test('a column with no server-side sort is not a button', async ({ page }) => {
  await page.goto('/products')
  await expect(page.getByRole('columnheader', { name: 'Status' }).getByRole('button')).toHaveCount(
    0,
  )
})

// Read the titles only once the skeleton is gone: six empty placeholder rows
// reverse to themselves, so this assertion cannot fail against them.
async function productTitles(page: Page): Promise<string[]> {
  await settled(page)
  return page.locator('tbody tr td:first-child a').allInnerTexts()
}

// Page one descending is the last page ascending, not page one reversed, so the
// two pages are compared by their first row rather than element by element.
test('sorting reorders the rows, not just the header', async ({ page }) => {
  await page.goto('/products?sort=title')
  const ascending = await productTitles(page)

  await page.goto('/products?sort=-title')
  const descending = await productTitles(page)

  expect(ascending.length).toBeGreaterThan(1)
  expect(descending[0]).not.toEqual(ascending[0])
})

// The search endpoint ranks by relevance and takes no sort. Leaving the headers
// clickable there would be a control that silently does nothing.
test('a searched list does not offer a sort it cannot apply', async ({ page }) => {
  await page.goto('/products?q=a&sort=title')
  await expect(page).toHaveURL(/\?q=a&sort=title$/)
  await expect(page.getByRole('columnheader', { name: 'Product' }).getByRole('button')).toHaveCount(
    0,
  )
  await expect(page.getByText('ranked by how well they match')).toBeVisible()
})

// The error state must not throw the filters away: reloading into an error and
// losing the query would make a shared link unrecoverable.
test('a list error keeps the filters that produced it', async ({ page }) => {
  await page.goto('/products?status=active&mock=error')
  await expect(page.getByText('could not be loaded')).toBeVisible()
  await expectChosen(page, 'Status', 'Active')
  await expect(page).toHaveURL(/status=active/)
})

// The order number is ~90px of text. Letting it absorb every spare pixel at a
// workstation width puts a row's first cell and its total a monitor apart.
test('no column absorbs the slack a wide viewport leaves', async ({ page }) => {
  await page.setViewportSize({ width: 2000, height: 1200 })
  await page.goto('/orders')
  await expect(page.getByRole('table')).toBeVisible()
  await settled(page)

  const cells = page.getByRole('row').nth(1).getByRole('cell')
  const first = await cells.first().boundingBox()
  const last = await cells.last().boundingBox()
  if (!first || !last) throw new Error('row cells not found')

  // The grow column is a deliberate choice, not whichever column happens to be
  // first: an order number is fixed-width text and must never absorb the slack.
  expect(first.width).toBeLessThan(420)
})
