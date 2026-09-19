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

// The block collapses offcanvas: the sidebar leaves entirely and the content
// takes the width, rather than shrinking to a rail of icons.
test('the sidebar collapses and restores on ctrl+b', async ({ page }) => {
  await page.goto('/orders')
  const sidebar = page.locator('[data-slot="sidebar"]')
  const main = page.getByRole('main')
  await expect(sidebar).toHaveAttribute('data-state', 'expanded')
  const expanded = (await main.boundingBox())!.width

  await page.keyboard.press('Control+b')
  await expect(sidebar).toHaveAttribute('data-state', 'collapsed')
  // Offcanvas slides the sidebar out rather than unmounting it, so it stays
  // "visible" to Playwright; what changes is where it is.
  await expect
    .poll(async () => (await page.getByRole('link', { name: 'Orders' }).boundingBox())!.x)
    .toBeLessThan(0)
  await expect.poll(async () => (await main.boundingBox())!.width).toBeGreaterThan(expanded)

  await page.keyboard.press('Control+b')
  await expect(sidebar).toHaveAttribute('data-state', 'expanded')
  await expect
    .poll(async () => (await page.getByRole('link', { name: 'Orders' }).boundingBox())!.x)
    .toBeGreaterThanOrEqual(0)
})

test('keyboard focus is visible', async ({ page }) => {
  await page.goto('/kit')
  await page.getByRole('button', { name: 'Open dialog' }).focus()
  await page.screenshot({ path: 'screenshots/interaction_focus.png' })
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.screenshot({ path: 'screenshots/interaction_dialog.png' })
})
