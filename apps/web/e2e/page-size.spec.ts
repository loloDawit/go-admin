import { expect, test } from '@playwright/test'
import { settled } from './select'

// A rows-per-page control the server ignores is worse than none. It has to
// reach the request and survive the next filter change, like page does.
test('rows per page reaches the server and survives a reload', async ({ page }) => {
  await page.goto('/products')
  await settled(page)
  await expect(page.locator('tbody tr')).toHaveCount(20)

  await page.getByRole('combobox', { name: 'Rows per page' }).click()
  await page.getByRole('option', { name: '10', exact: true }).click()
  await settled(page)
  await expect(page.locator('tbody tr')).toHaveCount(10)
  await expect(page).toHaveURL(/pageSize=10/)

  await page.reload()
  await settled(page)
  await expect(page.locator('tbody tr')).toHaveCount(10)
})
