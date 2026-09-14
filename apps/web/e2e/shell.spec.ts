import { expect, test } from '@playwright/test'

const ROUTES = [
  '/login',
  '/',
  '/orders',
  '/orders/o-5105',
  '/products',
  '/products/p-1042',
  '/products/p-1042/edit',
  '/customers',
  '/customers/c-3',
  '/staff',
  '/roles',
  '/permissions',
  '/profile',
  '/settings',
  '/kit',
  '/403',
  '/404-does-not-exist',
]

const VIEWPORTS = [
  { name: 'desktop', width: 1280, height: 900 },
  { name: 'narrow', width: 400, height: 900 },
]

for (const viewport of VIEWPORTS) {
  test.describe(viewport.name, () => {
    test.use({ viewport: { width: viewport.width, height: viewport.height } })

    for (const route of ROUTES) {
      test(`renders ${route}`, async ({ page }) => {
        const failures: string[] = []
        page.on('console', (message) => {
          if (message.type() === 'error') failures.push(message.text())
        })
        await page.goto(route)
        await expect(page.locator('h1')).toBeVisible()
        if (route !== '/kit') {
          await expect(page.locator('[aria-busy="true"]')).toHaveCount(0)
        }
        await page.screenshot({
          path: `screenshots/${viewport.name}${route.replace(/\//g, '_')}.png`,
          fullPage: true,
        })
        expect(failures).toEqual([])
      })
    }
  })
}

test('list states are reachable', async ({ page }) => {
  await page.goto('/orders?mock=error')
  await expect(page.getByRole('alert')).toContainText('could not be loaded')

  await page.goto('/orders?mock=empty')
  await expect(page.getByText('No orders in this period')).toBeVisible()
})

test('order status dialog opens and closes', async ({ page }) => {
  await page.goto('/orders/o-5105')
  await page.getByRole('button', { name: 'Update status' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
})

test('login rejects empty details', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByText('Enter the email you sign in with.')).toBeVisible()
})
