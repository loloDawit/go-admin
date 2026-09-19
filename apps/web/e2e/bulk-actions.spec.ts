import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

function uniquePrefix(): string {
  return `BULK-${Date.now()}`
}

async function createProduct(page: Page, sku: string, title: string, price: string): Promise<string> {
  await page.goto('/products/new')
  await page.getByRole('textbox', { name: 'SKU' }).fill(sku)
  await page.getByRole('textbox', { name: 'Title' }).fill(title)
  await page.getByRole('textbox', { name: 'Price' }).fill(price)
  await page.getByRole('button', { name: 'Create product' }).click()
  await expect(page.getByRole('heading', { name: title })).toBeVisible()
  const id = page.url().split('/').pop()
  if (!id) throw new Error('product id missing from the detail URL')
  return id
}

async function searchProducts(page: Page, q: string) {
  await page.goto('/products')
  await page.getByRole('searchbox', { name: 'Search' }).fill(q)
  await page.getByRole('button', { name: 'Search' }).click()
}

test('selecting two products and archiving them leaves both archived', async ({ page }) => {
  const prefix = uniquePrefix()
  const titleA = `${prefix} Alpha`
  const titleB = `${prefix} Bravo`
  const idA = await createProduct(page, `${prefix}-A`, titleA, '10.00')
  const idB = await createProduct(page, `${prefix}-B`, titleB, '12.00')

  await searchProducts(page, prefix)
  await expect(page.getByRole('link', { name: titleA })).toBeVisible()
  await expect(page.getByRole('link', { name: titleB })).toBeVisible()

  await page.getByRole('row', { name: titleA }).getByRole('checkbox').click()
  await page.getByRole('row', { name: titleB }).getByRole('checkbox').click()
  await expect(page.getByText('2 of 2 selected', { exact: true })).toBeVisible()

  await page.setViewportSize({ width: 1930, height: 1000 })
  await page.screenshot({ path: 'screenshots/bulk_light_selection.png' })

  await page.getByRole('button', { name: 'Archive', exact: true }).click()
  await expect(page.getByText('2 archived', { exact: true })).toBeVisible()

  // Archived products drop out of the default "Draft and active" filter, so
  // the honest verification is the resource itself, not the list that just
  // made them disappear.
  const resA = await page.request.get(`/api/v1/products/${idA}`)
  const resB = await page.request.get(`/api/v1/products/${idB}`)
  expect((await resA.json()).status).toBe('archived')
  expect((await resB.json()).status).toBe('archived')
})

test('a partial bulk failure names the failure and the successful row is really archived', async ({
  page,
}) => {
  const prefix = uniquePrefix()
  const titleA = `${prefix} Alpha`
  const titleB = `${prefix} Bravo`
  const idA = await createProduct(page, `${prefix}-A`, titleA, '10.00')
  const idB = await createProduct(page, `${prefix}-B`, titleB, '12.00')

  await searchProducts(page, prefix)

  await page.route('**/api/v1/products/*/archive', async (route) => {
    if (route.request().url().includes(idB)) {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ code: 'internal', message: 'boom' }),
      })
      return
    }
    await route.continue()
  })

  await page.getByRole('row', { name: titleA }).getByRole('checkbox').click()
  await page.getByRole('row', { name: titleB }).getByRole('checkbox').click()
  await page.getByRole('button', { name: 'Archive', exact: true }).click()

  await expect(page.getByText('1 archived, 1 failed', { exact: true })).toBeVisible()

  const resA = await page.request.get(`/api/v1/products/${idA}`)
  const resB = await page.request.get(`/api/v1/products/${idB}`)
  expect((await resA.json()).status).toBe('archived')
  expect((await resB.json()).status).toBe('draft')
})

test('changing a filter clears the selection instead of carrying it silently', async ({ page }) => {
  const prefix = uniquePrefix()
  const title = `${prefix} Solo`
  await createProduct(page, `${prefix}-S`, title, '5.00')
  await searchProducts(page, prefix)

  await page.getByRole('row', { name: title }).getByRole('checkbox').click()
  await expect(page.getByText('1 of 1 selected', { exact: true })).toBeVisible()

  await page.goto(`/products?status=active&q=${encodeURIComponent(prefix)}`)
  await expect(page.getByText('selected')).toHaveCount(0)
})

test('clicking a row checkbox toggles selection without navigating to the detail page', async ({
  page,
}) => {
  const prefix = uniquePrefix()
  const title = `${prefix} Guard`
  await createProduct(page, `${prefix}-G`, title, '5.00')
  await searchProducts(page, prefix)

  const checkbox = page.getByRole('row', { name: title }).getByRole('checkbox')
  await checkbox.click()
  await expect(checkbox).toBeChecked()
  await expect(page).toHaveURL(/\/products\?/)
})

test.describe('dark mode', () => {
  test('a selection and the bulk bar render in dark', async ({ page }) => {
    const prefix = uniquePrefix()
    const title = `${prefix} Night`
    await createProduct(page, `${prefix}-N`, title, '5.00')

    await page.addInitScript(() => {
      window.localStorage.setItem('go-admin-theme', 'dark')
    })
    await searchProducts(page, prefix)
    await page.setViewportSize({ width: 1930, height: 1000 })
    await page.getByRole('row', { name: title }).getByRole('checkbox').click()
    await expect(page.getByText('1 of 1 selected', { exact: true })).toBeVisible()
    await page.screenshot({ path: 'screenshots/bulk_dark_selection.png' })
  })
})

test.describe('orders', () => {
  async function mockOrders(page: Page) {
    await page.route('**/api/v1/orders*', async (route) => {
      if (route.request().method() !== 'GET') {
        await route.continue()
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [
            {
              id: 'order-pending',
              number: 'ORD-PENDING',
              customerId: 'cust-1',
              customerName: 'Pending Customer',
              status: 'pending',
              totalMinor: 1000,
              currency: 'USD',
              placedAt: new Date().toISOString(),
            },
            {
              id: 'order-paid',
              number: 'ORD-PAID',
              customerId: 'cust-2',
              customerName: 'Paid Customer',
              status: 'paid',
              totalMinor: 2000,
              currency: 'USD',
              placedAt: new Date().toISOString(),
            },
          ],
          page: 1,
          pageSize: 20,
          total: 2,
        }),
      })
    })
  }

  test('mark packed is offered only when every selected order is paid', async ({ page }) => {
    await mockOrders(page)
    await page.goto('/orders')

    await page.getByRole('row', { name: 'ORD-PENDING' }).getByRole('checkbox').click()
    await expect(page.getByRole('button', { name: 'Mark packed', exact: true })).toBeDisabled()

    await page.getByRole('row', { name: 'ORD-PAID' }).getByRole('checkbox').click()
    await expect(page.getByRole('button', { name: 'Mark packed', exact: true })).toBeDisabled()

    await page.getByRole('row', { name: 'ORD-PENDING' }).getByRole('checkbox').click()
    await expect(page.getByRole('button', { name: 'Mark packed', exact: true })).toBeEnabled()
  })
})
