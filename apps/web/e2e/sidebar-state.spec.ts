import { expect, test } from '@playwright/test'

const sidebar = '[data-slot="sidebar"]'

// The primitive writes a sidebar_state cookie on every toggle, but nothing read
// it back: dashboard-01 reads it server-side in Next.js, which has no
// equivalent here, so a collapsed rail sprang open on every navigation.
test('a collapsed rail stays collapsed across a reload', async ({ page }) => {
  await page.goto('/orders')
  await expect(page.locator(sidebar)).toHaveAttribute('data-state', 'expanded')

  await page.getByRole('button', { name: 'Toggle Sidebar' }).click()
  await expect(page.locator(sidebar)).toHaveAttribute('data-state', 'collapsed')

  await page.reload()
  await expect(page.locator(sidebar)).toHaveAttribute('data-state', 'collapsed')

  await page.goto('/products')
  await expect(page.locator(sidebar)).toHaveAttribute('data-state', 'collapsed')

  await page.getByRole('button', { name: 'Toggle Sidebar' }).click()
  await page.reload()
  await expect(page.locator(sidebar)).toHaveAttribute('data-state', 'expanded')
})
