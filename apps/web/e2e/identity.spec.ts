import { expect, test, type Page } from '@playwright/test'

// Each run gets its own email/role names so a rerun never collides with a previous run's row
// (the create routes answer 409 on a repeat address, and a stale role would carry stale members).
const stamp = Date.now()

async function loginAsOwner(page: Page) {
  await page.goto('/login')
  await page.getByRole('textbox', { name: 'Email' }).fill('owner@example.com')
  await page.getByRole('textbox', { name: 'Password' }).fill('dev_only_owner_password')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL('/')
}

// No test here may grant edit_staff to a new account, or last_admin below actually succeeds.
async function createRole(page: Page, name: string) {
  await page.goto('/roles')
  await page.getByRole('button', { name: 'Add role' }).click()
  await page.getByRole('textbox', { name: 'Name' }).fill(name)
  await page.getByRole('button', { name: 'Create role' }).click()
  await expect(page.getByText(name)).toBeVisible()
}

async function createStaff(
  page: Page,
  { email, firstName, lastName, roleName }: { email: string; firstName: string; lastName: string; roleName: string },
) {
  await page.goto('/staff')
  await page.getByRole('button', { name: 'Add colleague' }).click()
  await page.getByRole('textbox', { name: 'Email' }).fill(email)
  await page.getByRole('textbox', { name: 'First name' }).fill(firstName)
  await page.getByRole('textbox', { name: 'Last name' }).fill(lastName)
  await page.getByLabel('Role').selectOption({ label: roleName })
  await page.getByRole('button', { name: 'Create account' }).click()
  await expect(page.getByRole('heading', { name: 'Account created' })).toBeVisible()
  const password = await page.getByLabel('Generated password').inputValue()
  await page.getByRole('checkbox', { name: 'I have saved this password' }).check()
  await page.getByRole('button', { name: 'Done' }).click()
  return password
}

async function changeForcedPassword(page: Page, email: string, temporaryPassword: string) {
  await page.getByRole('textbox', { name: 'Email' }).fill(email)
  await page.getByRole('textbox', { name: 'Password' }).fill(temporaryPassword)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page).toHaveURL(/\/change-password$/)
  await page.getByRole('textbox', { name: 'Current password' }).fill(temporaryPassword)
  await page.getByRole('textbox', { name: 'New password' }).fill(`${temporaryPassword}-changed`)
  await page.getByRole('button', { name: 'Change password' }).click()
  await expect(page).toHaveURL('/', { timeout: 15_000 })
}

test('create, edit and deactivate a colleague, with the generated password shown once', async ({
  page,
}) => {
  await loginAsOwner(page)
  await createRole(page, `no-access-${stamp}`)
  const email = `avery.${stamp}@northgate.example`
  const password = await createStaff(page, {
    email,
    firstName: 'Avery',
    lastName: `Stone-${stamp}`,
    roleName: `no-access-${stamp}`,
  })
  await expect(page.getByRole('link', { name: `Avery Stone-${stamp}`, exact: true })).toBeVisible()

  await page.getByRole('link', { name: `Avery Stone-${stamp}`, exact: true }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  await page.getByRole('textbox', { name: 'Last name' }).fill(`Stonewright-${stamp}`)
  await page.getByRole('button', { name: 'Save changes' }).click()
  await expect(page.getByRole('heading', { name: `Avery Stonewright-${stamp}` })).toBeVisible()

  await page.getByRole('button', { name: 'Deactivate' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Deactivate' }).click()
  await expect(page.getByText('Deactivated')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Deactivate' })).toHaveCount(0)

  // Sign out, then the new colleague signs in and is forced to change their password first.
  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page).toHaveURL(/\/login$/)
  await page.getByRole('textbox', { name: 'Email' }).fill(email)
  await page.getByRole('textbox', { name: 'Password' }).fill(password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  // Deactivated, so even the correct one-time password no longer signs them in.
  await expect(page.getByText('email or password is incorrect')).toBeVisible()
})

test('editing a role changes what it grants', async ({ page }) => {
  await loginAsOwner(page)
  const roleName = `growing-${stamp}`
  await createRole(page, roleName)
  let row = page.locator('tr').filter({ hasText: roleName })
  await expect(row.getByText('0 of 8')).toBeVisible()

  await row.getByRole('button', { name: 'Edit' }).click()
  await page.getByRole('checkbox', { name: 'View orders and history' }).check()
  await page.getByRole('button', { name: 'Save changes' }).click()
  row = page.locator('tr').filter({ hasText: roleName })
  await expect(row.getByText('1 of 8')).toBeVisible()
})

test('a permission error on a real route is a page state, not a crash', async ({ page }) => {
  await loginAsOwner(page)
  const roleName = `viewer-${stamp}`
  await createRole(page, roleName)
  const email = `dana.${stamp}@northgate.example`
  const password = await createStaff(page, {
    email,
    firstName: 'Dana',
    lastName: `Okafor-${stamp}`,
    roleName,
  })

  await page.getByRole('button', { name: 'Sign out' }).click()
  await changeForcedPassword(page, email, password)

  // Dana holds no permissions, so Access is not offered at all — courtesy hiding, not the only guard.
  await expect(page.getByRole('link', { name: 'Staff' })).toHaveCount(0)

  await page.goto('/staff')
  await expect(page.getByText('This list could not be loaded')).toBeVisible()
  await expect(page.getByText('you do not have permission to perform this action')).toBeVisible()
  // The rest of the shell is still usable: this is a page-level state, not a broken app.
  await expect(page.getByRole('link', { name: 'Dashboard' })).toBeVisible()
})

test('the last active manager of staff cannot be deactivated', async ({ page }) => {
  await loginAsOwner(page)
  await page.goto('/staff/1')
  await page.getByRole('button', { name: 'Deactivate' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Deactivate' }).click()
  await expect(page.getByText('Could not deactivate this account')).toBeVisible()
  await expect(page.getByText(/at least one active person/i)).toBeVisible()
  await expect(page.getByText('Active', { exact: true })).toBeVisible()
})

test('a role still held by staff cannot be deleted', async ({ page }) => {
  await loginAsOwner(page)
  const roleName = `held-${stamp}`
  await createRole(page, roleName)
  await createStaff(page, {
    email: `held.${stamp}@northgate.example`,
    firstName: 'Held',
    lastName: `Role-${stamp}`,
    roleName,
  })

  await page.goto('/roles')
  const row = page.locator('tr').filter({ hasText: roleName })
  await row.getByRole('button', { name: 'Delete' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await expect(page.getByText('Could not delete this role')).toBeVisible()
  await expect(page.getByText('People still hold this role. Move them to another role first.')).toBeVisible()
})

test('a session that dies mid-visit sends the next action to login, not a crash', async ({
  page,
  context,
}) => {
  await loginAsOwner(page)
  // A client-side nav, not page.goto: a fresh navigation re-mounts the app and races its own
  // bootstrap /me fetch against clearCookies below. Staying on the already-settled page avoids that.
  await page.getByRole('link', { name: 'Orders' }).click()
  await expect(page.getByRole('heading', { name: 'Orders' })).toBeVisible()
  await context.clearCookies()
  await page.getByRole('link', { name: 'Staff' }).click()
  await expect(page).toHaveURL(/\/login$/)
})
