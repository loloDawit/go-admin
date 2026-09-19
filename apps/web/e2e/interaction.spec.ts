import { expect, test } from '@playwright/test'

test('navigation opens at narrow widths', async ({ page }) => {
  await page.setViewportSize({ width: 400, height: 900 })
  await page.goto('/orders')
  await expect(page.getByRole('navigation', { name: 'Sections' })).toBeHidden()
  await page.getByRole('button', { name: 'Toggle Sidebar' }).click()
  await expect(page.getByRole('navigation', { name: 'Sections' })).toBeVisible()
  await page.screenshot({ path: 'screenshots/interaction_narrow-nav.png', fullPage: true })
  await page.getByRole('link', { name: 'Products' }).click()
  await expect(page.getByRole('heading', { level: 1, name: 'Products' })).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Sections' })).toBeHidden()
})

// Collapsing leaves an icon rail rather than removing the sidebar: the labels
// and the account details go, the icons and the tooltips stay.
test('the sidebar collapses to an icon rail and restores on ctrl+b', async ({ page }) => {
  await page.goto('/orders')
  const sidebar = page.locator('[data-slot="sidebar"]')
  const ordersLink = page.getByRole('link', { name: 'Orders' })
  await expect(sidebar).toHaveAttribute('data-state', 'expanded')
  await expect(ordersLink).toBeVisible()

  await page.keyboard.press('Control+b')
  await expect(sidebar).toHaveAttribute('data-state', 'collapsed')
  await ordersLink.hover()
  await expect(page.getByRole('tooltip', { name: 'Orders' })).toBeVisible()

  await page.keyboard.press('Control+b')
  await expect(sidebar).toHaveAttribute('data-state', 'expanded')
  await expect(page.getByRole('tooltip')).toBeHidden()
})

test('keyboard focus is visible', async ({ page }) => {
  await page.goto('/kit')
  await page.getByRole('button', { name: 'Open dialog' }).focus()
  await page.screenshot({ path: 'screenshots/interaction_focus.png' })
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.screenshot({ path: 'screenshots/interaction_dialog.png' })
})
