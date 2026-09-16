import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

function unique(): string {
  return String(Date.now())
}

async function activeProduct(page: Page, title: string, price: string) {
  await page.goto('/products/new')
  await page.getByRole('textbox', { name: 'SKU' }).fill(`ORD-${unique()}`)
  await page.getByRole('textbox', { name: 'Title' }).fill(title)
  await page.getByRole('textbox', { name: 'Price' }).fill(price)
  await page.getByRole('button', { name: 'Create product' }).click()
  await page.getByRole('button', { name: 'Activate' }).click()
  await expect(page.getByText('Active', { exact: true })).toBeVisible()
}

async function customer(page: Page, name: string) {
  await page.goto('/customers')
  await page.getByRole('button', { name: 'Add customer' }).first().click()
  await page.getByRole('textbox', { name: 'Name' }).fill(name)
  await page.getByRole('textbox', { name: 'Email' }).fill(`${unique()}@example.com`)
  await page.getByRole('dialog').getByRole('button', { name: 'Add customer' }).click()
  await expect(page.getByRole('link', { name })).toBeVisible()
}

// The milestone path end to end: a product has to be activated before it can
// be sold, and every transition has to be readable afterwards.
test('a product is sold, the order advances, and the history records it', async ({ page }) => {
  const id = unique()
  const title = `Sellable ${id}`
  const buyer = `Buyer ${id}`

  await activeProduct(page, title, '12.50')
  await customer(page, buyer)

  await page.goto('/orders/new')
  await page.getByLabel('Customer').selectOption({ label: buyer })
  await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await page.getByRole('button', { name: 'Add' }).click()
  await expect(page.getByText('$12.50').first()).toBeVisible()

  await page.getByLabel(`Quantity of ${title}`).fill('2')
  await expect(page.getByText('$25.00').first()).toBeVisible()
  await page.screenshot({ path: 'screenshots/order_create.png', fullPage: true })

  await page.getByRole('button', { name: 'Place order' }).click()
  await expect(page.getByRole('heading', { name: /^ORD-/ })).toBeVisible()
  await expect(page.getByText('Awaiting payment')).toBeVisible()
  await expect(page.getByText('Order placed')).toBeVisible()

  await page.getByRole('button', { name: 'Mark paid' }).click()
  await expect(page.getByRole('button', { name: 'Mark packed' })).toBeVisible()
  await expect(page.getByText('Awaiting payment → Paid')).toBeVisible()

  await page.getByRole('button', { name: 'Mark packed' }).click()
  await expect(page.getByText('Paid → Packed')).toBeVisible()
  await page.screenshot({ path: 'screenshots/order_detail.png', fullPage: true })
})

test('a cancelled order records the reason and moves no further', async ({ page }) => {
  const id = unique()
  const title = `Cancellable ${id}`
  const buyer = `Regretful ${id}`

  await activeProduct(page, title, '4.00')
  await customer(page, buyer)

  await page.goto('/orders/new')
  await page.getByLabel('Customer').selectOption({ label: buyer })
  await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await page.getByRole('button', { name: 'Add' }).click()
  await page.getByRole('button', { name: 'Place order' }).click()
  await expect(page.getByRole('heading', { name: /^ORD-/ })).toBeVisible()

  await page.getByRole('button', { name: 'Cancel order' }).click()
  await page.getByRole('textbox', { name: 'Reason' }).fill('Customer changed their mind')
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel order' }).click()

  await expect(page.getByText('Cancelled', { exact: true })).toBeVisible()
  await expect(page.getByText('Customer changed their mind')).toBeVisible()
  await expect(page.getByRole('button', { name: /^Mark / })).toHaveCount(0)
})

test('a draft product cannot be ordered', async ({ page }) => {
  const id = unique()
  const title = `Unactivated ${id}`

  await page.goto('/products/new')
  await page.getByRole('textbox', { name: 'SKU' }).fill(`DRAFT-${id}`)
  await page.getByRole('textbox', { name: 'Title' }).fill(title)
  await page.getByRole('button', { name: 'Create product' }).click()
  await expect(page.getByText('Draft', { exact: true })).toBeVisible()

  await page.goto('/orders/new')
  await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await expect(page.getByText('Nothing active matches')).toBeVisible()
})

test.describe('narrow', () => {
  test.use({ viewport: { width: 400, height: 900 } })

  test('the order screens hold at 400px', async ({ page }) => {
    const id = unique()
    const title = `Narrow ${id}`
    const buyer = `Narrow buyer ${id}`

    await activeProduct(page, title, '7.25')
    await customer(page, buyer)

    await page.goto('/orders/new')
    await expect(page.getByRole('heading', { name: 'New order' })).toBeVisible()
    await page.getByLabel('Customer').selectOption({ label: buyer })
    await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
    await page.getByRole('button', { name: 'Search' }).click()
    await page.getByRole('button', { name: 'Add' }).click()
    await page.screenshot({ path: 'screenshots/narrow_order_create.png', fullPage: true })

    await page.getByRole('button', { name: 'Place order' }).click()
    await expect(page.getByRole('heading', { name: /^ORD-/ })).toBeVisible()
    await page.screenshot({ path: 'screenshots/narrow_order_detail.png', fullPage: true })
  })
})

// The service refuses an order whose lines quote different currencies. The screen
// has to refuse it first: a total summed across currencies means nothing.
test('an order cannot mix currencies', async ({ page, request }) => {
  const stamp = unique()
  const usd = `USD line ${stamp}`
  const eur = `EUR line ${stamp}`
  const buyer = `Mixed ${stamp}`

  await activeProduct(page, usd, '10.00')

  const created = await request.post('/api/v1/products', {
    data: { sku: `EUR-${stamp}`, title: eur, priceMinor: 900, currency: 'EUR' },
  })
  expect(created.ok()).toBeTruthy()
  const { id } = (await created.json()) as { id: string }
  expect((await request.post(`/api/v1/products/${id}/activate`)).ok()).toBeTruthy()

  await customer(page, buyer)

  await page.goto('/orders/new')
  await page.getByLabel('Customer').selectOption({ label: buyer })
  for (const title of [usd, eur]) {
    await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
    await page.getByRole('button', { name: 'Search' }).click()
    await page.getByRole('button', { name: 'Add' }).click()
  }

  await expect(page.getByText('Every line must be in one currency')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Place order' })).toBeDisabled()
  await expect(page.getByText('Total')).toHaveCount(0)
})
