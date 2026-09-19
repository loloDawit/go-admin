import { expect, test } from '@playwright/test'
import { settled } from './select'

// A customer with no orders has a lifetime value of 0 in no currency, which the
// service reports as an empty currency code. Intl.NumberFormat throws a
// RangeError on one, which unmounted the whole page.
test('a customer with no orders opens instead of crashing', async ({ page }) => {
  const stamp = String(Date.now())
  const name = `Unordered ${stamp}`

  await page.goto('/customers')
  await page.getByRole('button', { name: 'Add customer' }).first().click()
  await page.getByRole('textbox', { name: 'Name' }).fill(name)
  await page.getByRole('textbox', { name: 'Email' }).fill(`${stamp}@example.com`)
  await page.getByRole('dialog').getByRole('button', { name: 'Add customer' }).click()

  await page.getByRole('link', { name }).click()
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible()
  await expect(page.getByText('Unexpected Application Error')).toHaveCount(0)
  await expect(page.getByText('No orders yet').first()).toBeVisible()
})

const VIEWPORTS = [
  { name: 'desktop', width: 1280, height: 900 },
  { name: 'narrow', width: 400, height: 900 },
]

for (const viewport of VIEWPORTS) {
  test.describe(viewport.name, () => {
    test.use({ viewport: { width: viewport.width, height: viewport.height } })

    test('captures a real customer detail', async ({ page }) => {
      const stamp = String(Date.now())
      const name = `Captured ${stamp}`

      await page.goto('/customers')
      await page.getByRole('button', { name: 'Add customer' }).first().click()
      await page.getByRole('textbox', { name: 'Name' }).fill(name)
      await page.getByRole('textbox', { name: 'Email' }).fill(`${stamp}@example.com`)
      await page.getByRole('dialog').getByRole('button', { name: 'Add customer' }).click()

      await page.getByRole('link', { name }).click()
      // level 1, because an accessible name matches by case-insensitive
      // substring: on an empty database the Orders page also carries the
      // heading "No orders yet", which answers to "Orders".
      await expect(page.getByRole('heading', { level: 1, name })).toBeVisible()
      await settled(page)
      await page.screenshot({
        path: `screenshots/${viewport.name}_customer-detail.png`,
        fullPage: true,
      })
    })
  })
}

// Both render outside the shell, so a direct visit is their real presentation.
test.describe('signed out', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  for (const viewport of VIEWPORTS) {
    test(`captures login and change-password at ${viewport.name}`, async ({ page }) => {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })

      await page.goto('/login')
      await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
      await page.screenshot({ path: `screenshots/${viewport.name}_login.png`, fullPage: true })

      await page.goto('/change-password')
      await expect(page.locator('h1')).toBeVisible()
      await page.screenshot({
        path: `screenshots/${viewport.name}_change-password.png`,
        fullPage: true,
      })
    })
  }
})
