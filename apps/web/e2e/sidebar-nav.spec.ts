import { expect, test, type Page } from '@playwright/test'

// Intercepting the report keeps the badge's count deterministic and counts fulfillments,
// which proves getDashboard's in-flight dedupe rather than racing the real fetch.
async function mockDashboard(page: Page, counts: Record<string, number>) {
  let fulfilled = 0
  await page.route('**/api/v1/reports/dashboard', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ counts, recent: [], revenue: [] }),
    })
    fulfilled++
  })
  return () => fulfilled
}

function ordersMenuItem(page: Page) {
  return page
    .getByRole('link', { name: 'Orders', exact: true })
    .locator('xpath=ancestor::li[@data-slot="sidebar-menu-item"]')
}

test('the Orders badge states the same waiting total as the dashboard KPI', async ({ page }) => {
  await mockDashboard(page, { pending: 2, paid: 3, packed: 1, shipped: 9, delivered: 40 })
  await page.goto('/')
  const card = page.locator('[data-slot="card"]', { hasText: 'Orders to handle' })
  await expect(card.locator('[data-slot="card-title"]')).toHaveText('6')
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveText('6')
})

test('a zero waiting total renders no badge', async ({ page }) => {
  const fulfilledCount = await mockDashboard(page, { pending: 0, paid: 0, packed: 0 })
  await page.goto('/')
  const card = page.locator('[data-slot="card"]', { hasText: 'Orders to handle' })
  await expect(card.locator('[data-slot="card-title"]')).toHaveText('0')
  // The KPI card settling on '0' proves the sidebar's own fetch (sharing the same
  // in-flight request) has settled too, so an absent badge here means "zero", not "still loading".
  expect(fulfilledCount()).toBe(1)
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveCount(0)
})

test('the badge does not appear while the report is still loading', async ({ page }) => {
  await page.route('**/api/v1/reports/dashboard', async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 1500))
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ counts: { pending: 5 }, recent: [], revenue: [] }),
    })
  })
  await page.goto('/')
  await expect(ordersMenuItem(page)).toBeVisible()
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveCount(0)
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveText('5')
})

test('the Orders link keeps its plain accessible name once the badge renders', async ({ page }) => {
  await mockDashboard(page, { pending: 249 })
  await page.goto('/')
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveText('249')
  await expect(page.getByRole('link', { name: 'Orders', exact: true })).toHaveCount(1)
})

test('the report is fetched once per session, not once per navigation', async ({ page }) => {
  const fulfilledCount = await mockDashboard(page, { pending: 1 })
  await page.goto('/orders')
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveText('1')
  await page.getByRole('link', { name: 'Products', exact: true }).click()
  await expect(page.getByRole('heading', { level: 1, name: 'Products' })).toBeVisible()
  await page.getByRole('link', { name: 'Customers', exact: true }).click()
  await expect(page.getByRole('heading', { level: 1, name: 'Customers' })).toBeVisible()
  expect(fulfilledCount()).toBe(1)
})

test('Roles and Permissions nest under a parent instead of sitting flat', async ({ page }) => {
  await page.goto('/staff')
  const rolesLink = page.getByRole('link', { name: 'Roles', exact: true })
  const permissionsLink = page.getByRole('link', { name: 'Permissions', exact: true })
  await expect(rolesLink.locator('xpath=ancestor::ul[@data-slot="sidebar-menu-sub"]')).toHaveCount(1)
  await expect(permissionsLink.locator('xpath=ancestor::ul[@data-slot="sidebar-menu-sub"]')).toHaveCount(1)
  // Staff stays a direct sibling in the group, not swept into the same sub-list.
  await expect(
    page
      .getByRole('link', { name: 'Staff', exact: true })
      .locator('xpath=ancestor::ul[@data-slot="sidebar-menu-sub"]'),
  ).toHaveCount(0)
})

test('collapsing to the icon rail hides the nested sub-items', async ({ page }) => {
  await page.goto('/staff')
  const roles = page.getByRole('link', { name: 'Roles', exact: true })
  await expect(roles).toBeVisible()

  // The rail keeps its icons; a nested sub-item has nowhere to sit and hides.
  await page.keyboard.press('Control+b')
  await expect(roles).toBeHidden()

  await page.keyboard.press('Control+b')
  await expect(roles).toBeVisible()
})
