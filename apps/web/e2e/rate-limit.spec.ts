import { expect, test } from '@playwright/test'

// The gateway answers a rate-limited caller with {code: "rate_limited"}. Read
// as "unreachable" it told the person to check a connection that is fine.
test('a rate-limited bootstrap says so, rather than blaming the connection', async ({ page }) => {
  await page.route('**/api/v1/me', (route) =>
    route.fulfill({
      status: 429,
      contentType: 'application/json',
      headers: { 'Retry-After': '1' },
      body: JSON.stringify({ code: 'rate_limited', message: 'too many requests; try again shortly' }),
    }),
  )

  await page.goto('/orders')
  await expect(page.getByRole('heading', { name: 'Too many requests' })).toBeVisible()
  await expect(page.getByText(/check your connection/i)).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Try again' })).toBeVisible()
})
