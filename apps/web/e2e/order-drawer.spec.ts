import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

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
            id: 'order-quick-look',
            number: 'ORD-QUICKLOOK',
            customerId: 'cust-1',
            customerName: 'Quick Look Customer',
            status: 'paid',
            totalMinor: 4500,
            currency: 'USD',
            placedAt: new Date().toISOString(),
          },
        ],
        page: 1,
        pageSize: 20,
        total: 1,
      }),
    })
  })
}

test('the row itself still opens the full order', async ({ page }) => {
  await mockOrders(page)
  await page.goto('/orders')
  await page.getByRole('cell', { name: 'ORD-QUICKLOOK', exact: true }).click()
  await expect(page).toHaveURL(/\/orders\/order-quick-look$/)
})

test('quick look previews a row without leaving the list, then Escape restores it', async ({ page }) => {
  await mockOrders(page)
  await page.goto('/orders')

  await page.getByRole('button', { name: 'Actions for ORD-QUICKLOOK', exact: true }).click()
  await page.getByRole('menuitem', { name: 'Quick look', exact: true }).click()

  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('ORD-QUICKLOOK')).toBeVisible()
  await expect(page).toHaveURL(/\/orders$/)

  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()

  // the list underneath stayed live through the preview
  await page.getByRole('tab', { name: 'All', exact: true }).click()
})

test('quick look links through to the full order', async ({ page }) => {
  await mockOrders(page)
  await page.goto('/orders')

  await page.getByRole('button', { name: 'Actions for ORD-QUICKLOOK', exact: true }).click()
  await page.getByRole('menuitem', { name: 'Quick look', exact: true }).click()
  await page.getByRole('dialog').getByRole('link', { name: 'Open order', exact: true }).click()

  await expect(page).toHaveURL(/\/orders\/order-quick-look$/)
})
