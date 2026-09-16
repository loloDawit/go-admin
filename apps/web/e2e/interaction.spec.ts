import { expect, test } from '@playwright/test'

test('navigation opens at narrow widths', async ({ page }) => {
  await page.setViewportSize({ width: 400, height: 900 })
  await page.goto('/orders')
  await expect(page.getByRole('navigation', { name: 'Sections' })).toBeHidden()
  await page.getByRole('button', { name: 'Menu' }).click()
  await expect(page.getByRole('navigation', { name: 'Sections' })).toBeVisible()
  await page.screenshot({ path: 'screenshots/interaction_narrow-nav.png', fullPage: true })
  await page.getByRole('link', { name: 'Products' }).click()
  await expect(page.getByRole('heading', { name: 'Products' })).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Sections' })).toBeHidden()
})

test('keyboard focus is visible', async ({ page }) => {
  await page.goto('/kit')
  await page.getByRole('button', { name: 'Open dialog' }).focus()
  await page.screenshot({ path: 'screenshots/interaction_focus.png' })
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.screenshot({ path: 'screenshots/interaction_dialog.png' })
})

