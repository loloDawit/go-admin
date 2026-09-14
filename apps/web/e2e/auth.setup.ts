import { test as setup, expect } from '@playwright/test'

const authFile = 'playwright/.auth/owner.json'

setup('authenticate as the seeded owner', async ({ page }) => {
  await page.goto('/login')
  await page.getByRole('textbox', { name: 'Email' }).fill('owner@example.com')
  await page.getByRole('textbox', { name: 'Password' }).fill('dev_only_owner_password')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL('/')
  await page.context().storageState({ path: authFile })
})
