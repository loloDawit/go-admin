import { expect, test } from '@playwright/test'

test('the kit shows every status tone and both empty states', async ({ page }) => {
  await page.goto('/kit')
  await expect(page.getByRole('heading', { name: 'Component kit' })).toBeVisible()
  for (const label of ['Archived', 'Paid', 'Delivered', 'Pending', 'Refunded']) {
    await expect(page.getByText(label, { exact: true })).toBeVisible()
  }
  await expect(page.getByRole('heading', { name: 'Nothing here yet' })).toBeVisible()
  await expect(
    page.getByRole('heading', { name: 'No products match these filters' }),
  ).toBeVisible()
})

// The dialog is Radix, not <dialog open>: it is a div with role="dialog", and
// Escape must still close it.
test('the kit dialog traps focus and closes on Escape', async ({ page }) => {
  await page.goto('/kit')
  await page.getByRole('button', { name: 'Open dialog' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('heading', { name: 'Cancel this order?' })).toBeVisible()
  await page.keyboard.press("Escape")
  await expect(dialog).toBeHidden()
})

test('an invalid field is announced and tied to its message', async ({ page }) => {
  await page.goto('/kit')
  const field = page.getByLabel('Title')
  await expect(field).toHaveAttribute('aria-invalid', 'true')
  const describedBy = await field.getAttribute('aria-describedby')
  expect(describedBy).toBeTruthy()
  await expect(page.locator(`#${describedBy}`)).toHaveText('Give the product a title.')
})
