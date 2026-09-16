import { expect, test } from '@playwright/test'

// Owner's own id after a clean `make seed` is always 1: the seed command creates exactly one row.
const ROUTES = [
  '/',
  '/orders',
  '/orders/new',
  '/products',
  '/products/new',
  '/customers',
  '/staff',
  '/staff/1',
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
  await expect(page.getByText('No orders yet')).toBeVisible()
})

// The identity screens now require a real session; these two exercise the signed-out state, which
// the "authenticated" project's shared storageState would otherwise mask.
test.describe('signed out', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('renders /login', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('h1')).toBeVisible()
  })

  test('an unauthenticated visit to a shell route redirects to /login', async ({ page }) => {
    await page.goto('/staff')
    await expect(page).toHaveURL(/\/login$/)
  })

  test('login rejects empty details', async ({ page }) => {
    await page.goto('/login')
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page.getByText('Enter the email you sign in with.')).toBeVisible()
  })
})
