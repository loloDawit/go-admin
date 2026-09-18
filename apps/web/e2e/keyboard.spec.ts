import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

async function selectCustomer(page: Page, name: string, stamp: string): Promise<void> {
  await page.getByRole('textbox', { name: 'Customer email' }).fill(`${stamp}@example.com`)
  await page.getByRole('textbox', { name: 'Customer email' }).press('Enter')
  await expect(page.getByText(`Ordering for ${name}`)).toBeVisible()
}

function unique(): string {
  return String(Date.now())
}

async function draftProduct(page: Page, title: string): Promise<void> {
  await page.goto('/products/new')
  await page.getByRole('textbox', { name: 'SKU' }).fill(`KEY-${unique()}`)
  await page.getByRole('textbox', { name: 'Title' }).fill(title)
  await page.getByRole('button', { name: 'Create product' }).click()
  await expect(page.getByRole('heading', { name: title })).toBeVisible()
}

async function addCustomer(page: Page, name: string, stamp: string): Promise<void> {
  await page.goto('/customers')
  await page.getByRole('button', { name: 'Add customer' }).first().click()
  await page.getByRole('textbox', { name: 'Name' }).fill(name)
  await page.getByRole('textbox', { name: 'Email' }).fill(`${stamp}@example.com`)
  await page.getByRole('dialog').getByRole('button', { name: 'Add customer' }).click()
  await expect(page.getByRole('link', { name })).toBeVisible()
}

test('escape closes the archive dialog and the product is not archived', async ({ page }) => {
  const title = `Escapable ${unique()}`
  await draftProduct(page, title)
  await page.getByRole('button', { name: 'Activate' }).click()
  await expect(page.getByText('Active', { exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Archive' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toBeHidden()

  await page.reload()
  await expect(page.getByText('Active', { exact: true })).toBeVisible()
})

test('escape closes the add-customer dialog and no customer is created', async ({ page }) => {
  const stamp = unique()
  await page.goto('/customers')
  await page.getByRole('button', { name: 'Add customer' }).first().click()
  await page.getByRole('textbox', { name: 'Name' }).fill(`Discarded ${stamp}`)
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toBeHidden()

  await page.reload()
  await expect(page.getByRole('link', { name: `Discarded ${stamp}` })).toHaveCount(0)
})

test('escape closes the cancel-order dialog and the order still moves', async ({ page }) => {
  const stamp = unique()
  const title = `Keyboard ${stamp}`
  const buyer = `Keyboard buyer ${stamp}`

  await draftProduct(page, title)
  await page.getByRole('button', { name: 'Activate' }).click()
  await expect(page.getByText('Active', { exact: true })).toBeVisible()

  await addCustomer(page, buyer, stamp)

  await page.goto('/orders/new')
  await selectCustomer(page, buyer, stamp)
  await page.getByRole('searchbox', { name: 'Search the catalog' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await page.getByRole('button', { name: 'Add' }).click()
  await page.getByRole('button', { name: 'Place order' }).click()
  await expect(page.getByRole('heading', { name: /^ORD-/ })).toBeVisible()

  await page.getByRole('button', { name: 'Cancel order' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toBeHidden()

  await expect(page.getByText('Awaiting payment')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Mark paid' })).toBeEnabled()
})

test('focus returns to the trigger when a dialog closes', async ({ page }) => {
  await page.goto('/customers')
  const trigger = page.getByRole('button', { name: 'Add customer' }).first()
  await trigger.click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.keyboard.press('Escape')

  await expect(trigger).toBeFocused()
})

test('a dialog traps tab focus inside itself', async ({ page }) => {
  await page.goto('/customers')
  await page.getByRole('button', { name: 'Add customer' }).first().click()
  await expect(page.getByRole('dialog')).toBeVisible()

  for (let i = 0; i < 12; i++) await page.keyboard.press('Tab')

  const inside = await page.evaluate(() => {
    const dialog = document.querySelector('dialog[open]')
    return (
      dialog !== null && document.activeElement !== null && dialog.contains(document.activeElement)
    )
  })
  expect(inside).toBe(true)
})
