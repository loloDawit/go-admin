import { expect, test } from '@playwright/test'
import { settled } from './select'
import type { Page } from '@playwright/test'

async function selectCustomer(page: Page, name: string, stamp: string): Promise<void> {
  await page.getByRole('textbox', { name: 'Customer email' }).fill(`${stamp}@example.com`)
  await page.getByRole('textbox', { name: 'Customer email' }).press('Enter')
  await expect(page.getByText(`Ordering for ${name}`)).toBeVisible()
}

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

// The email is the caller's stamp, not a fresh one: selectCustomer looks the
// customer up by it, so the two have to agree.
async function customer(page: Page, name: string, stamp: string) {
  await page.goto('/customers')
  await page.getByRole('button', { name: 'Add customer' }).first().click()
  await page.getByRole('textbox', { name: 'Name' }).fill(name)
  await page.getByRole('textbox', { name: 'Email' }).fill(`${stamp}@example.com`)
  await page.getByRole('dialog').getByRole('button', { name: 'Add customer' }).click()
  await expect(page.getByRole('link', { name })).toBeVisible()
}

// The milestone path end to end: a product has to be activated before it can
// be sold, and every transition has to be readable afterwards.
test('a product is sold, the order advances, and the history records it', async ({ page }) => {
  const stamp = unique()
  const title = `Sellable ${stamp}`
  const buyer = `Buyer ${stamp}`

  await activeProduct(page, title, '12.50')
  await customer(page, buyer, stamp)

  await page.goto('/orders/new')
  await selectCustomer(page, buyer, stamp)
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
  const stamp = unique()
  const title = `Cancellable ${stamp}`
  const buyer = `Regretful ${stamp}`

  await activeProduct(page, title, '4.00')
  await customer(page, buyer, stamp)

  await page.goto('/orders/new')
  await selectCustomer(page, buyer, stamp)
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
  const stamp = unique()
  const title = `Unactivated ${stamp}`

  await page.goto('/products/new')
  await page.getByRole('textbox', { name: 'SKU' }).fill(`DRAFT-${stamp}`)
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
    const stamp = unique()
    const title = `Narrow ${stamp}`
    const buyer = `Narrow buyer ${stamp}`

    await activeProduct(page, title, '7.25')
    await customer(page, buyer, stamp)

    await page.goto('/orders/new')
    await expect(page.getByRole('heading', { name: 'New order' })).toBeVisible()
    await selectCustomer(page, buyer, stamp)
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

  await customer(page, buyer, stamp)

  await page.goto('/orders/new')
  await selectCustomer(page, buyer, stamp)
  for (const title of [usd, eur]) {
    await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
    await page.getByRole('button', { name: 'Search' }).click()
    await page.getByRole('button', { name: 'Add' }).click()
  }

  await expect(page.getByText('Every line must be in one currency')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Place order' })).toBeDisabled()
  await expect(page.getByText('Total')).toHaveCount(0)
})

// The dashboard made four requests to render three numbers and could not show
// revenue at all. One request, and revenue per day and currency.
test('the dashboard reports revenue once an order is paid', async ({ page }) => {
  const stamp = unique()
  const title = `Revenue ${stamp}`
  const buyer = `Revenue buyer ${stamp}`

  await activeProduct(page, title, '40.00')
  await customer(page, buyer, stamp)

  await page.goto('/orders/new')
  await selectCustomer(page, buyer, stamp)
  await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await page.getByRole('button', { name: 'Add' }).click()
  await page.getByRole('button', { name: 'Place order' }).click()
  await expect(page.getByRole('heading', { name: /^ORD-/ })).toBeVisible()
  await page.getByRole('button', { name: 'Mark paid' }).click()
  await expect(page.getByRole('button', { name: 'Mark packed' })).toBeVisible()

  let requests = 0
  page.on('request', (r) => {
    if (r.url().includes('/api/v1/')) requests++
  })

  await page.goto('/')
  await expect(page.getByRole('heading', { level: 2, name: 'Revenue' })).toBeVisible()
  // The projection is asynchronous: the worker has to publish and apply before
  // the figure moves, so this polls rather than asserting once.
  await expect
    .poll(() => page.getByRole('cell', { name: '$40.00' }).first().isVisible(), {
      timeout: 20000,
    })
    .toBe(true)

  expect(requests).toBeLessThanOrEqual(2)
})

// A list of order numbers against nothing but a status is not something staff
// can scan: the row has to say who the order is for.
test('the orders list names the customer', async ({ page }) => {
  const stamp = unique()
  const buyer = `Listed buyer ${stamp}`
  const title = `Listed ${stamp}`

  await activeProduct(page, title, '9.00')
  await customer(page, buyer, stamp)

  await page.goto('/orders/new')
  await selectCustomer(page, buyer, stamp)
  await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await page.getByRole('button', { name: 'Add' }).click()
  await page.getByRole('button', { name: 'Place order' }).click()
  await expect(page.getByRole('heading', { name: /^ORD-/ })).toBeVisible()

  await page.goto('/orders')
  await settled(page)
  await expect(page.getByRole('columnheader', { name: 'Customer' })).toBeVisible()
  await expect(page.getByRole('cell', { name: buyer }).first()).toBeVisible()
})

// An action that appears to do nothing is an action staff repeat. The
// confirmation names what happened in the same words the button used.
test('advancing an order confirms what happened', async ({ page }) => {
  const stamp = unique()
  const buyer = `Toast buyer ${stamp}`
  const title = `Toast ${stamp}`

  await activeProduct(page, title, '5.00')
  await customer(page, buyer, stamp)

  await page.goto('/orders/new')
  await selectCustomer(page, buyer, stamp)
  await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await page.getByRole('button', { name: 'Add' }).click()
  await page.getByRole('button', { name: 'Place order' }).click()
  await expect(page.getByRole('heading', { name: /^ORD-/ })).toBeVisible()

  await page.getByRole('button', { name: 'Mark paid' }).click()
  await expect(page.getByText('Order paid')).toBeVisible()
})
