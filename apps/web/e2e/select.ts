import { expect } from '@playwright/test'
import type { Page } from '@playwright/test'

// Radix renders a listbox, not <select>: selectOption and toHaveValue do not
// apply. The trigger carries the label and shows the chosen option's text.
export async function chooseOption(page: Page, label: string, option: string) {
  await page.getByRole('combobox', { name: label }).click()
  await page.getByRole('option', { name: option, exact: true }).click()
}

export async function expectChosen(page: Page, label: string, option: string) {
  await expect(page.getByRole('combobox', { name: label })).toHaveText(option)
}

// aria-busy is true in the first paint, because the resource hook starts in
// its loading state. Asserting the count alone therefore also passes before
// React has mounted anything, which is a gate that cannot fail; waiting for the
// main region first guarantees a paint has happened.
export async function settled(page: Page) {
  await expect(page.getByRole('main')).toBeVisible()
  await expect(page.locator('[aria-busy="true"]')).toHaveCount(0)
}
