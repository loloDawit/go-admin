import { expect, test, type Page } from '@playwright/test'

// Intercepting the report keeps the badge's count deterministic — the seeded
// backend's own order mix is not something this spec controls or should assert on.
async function mockDashboard(page: Page, counts: Record<string, number>) {
  await page.route('**/api/v1/reports/dashboard', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ counts, recent: [], revenue: [] }),
    })
  })
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
  await mockDashboard(page, { pending: 0, paid: 0, packed: 0 })
  await page.goto('/')
  await expect(page.locator('[data-slot="card"]', { hasText: 'Orders to handle' }).locator('[data-slot="card-title"]')).toHaveText('0')
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
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveCount(0)
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveText('5')
})

test('the Orders link keeps its plain accessible name once the badge renders', async ({ page }) => {
  await mockDashboard(page, { pending: 249 })
  await page.goto('/')
  await expect(ordersMenuItem(page).locator('[data-slot="sidebar-menu-badge"]')).toHaveText('249')
  await expect(page.getByRole('link', { name: 'Orders', exact: true })).toHaveCount(1)
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

test('collapsing the rail to icons hides the nested sub-items entirely', async ({ page }) => {
  await page.goto('/staff')
  await expect(page.getByRole('link', { name: 'Roles', exact: true })).toBeVisible()
  await page.keyboard.press('Control+b')
  await expect(page.getByRole('link', { name: 'Roles', exact: true })).toBeHidden()
  await page.keyboard.press('Control+b')
  await expect(page.getByRole('link', { name: 'Roles', exact: true })).toBeVisible()
})
