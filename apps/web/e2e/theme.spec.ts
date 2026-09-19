import { expect, test } from '@playwright/test'

// The toggle's accessible name is "Theme": distinct from the sidebar trigger
// ("Toggle Sidebar") and the breadcrumb text, so a substring match is safe.
test.describe('theme', () => {
  test('system preference sets the root class and reacts live', async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' })
    await page.goto('/')
    await expect(page.locator('html')).toHaveClass(/dark/)

    await page.emulateMedia({ colorScheme: 'light' })
    await expect(page.locator('html')).not.toHaveClass(/dark/)
  })

  test('choosing dark sets the root class and survives a reload', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('button', { name: 'Theme' }).click()
    await page.getByRole('menuitemradio', { name: 'Dark', exact: true }).click()
    await expect(page.locator('html')).toHaveClass(/dark/)

    await page.reload()
    await expect(page.locator('html')).toHaveClass(/dark/)
  })

  test('an explicit light choice overrides a dark system preference', async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' })
    await page.goto('/')
    await expect(page.locator('html')).toHaveClass(/dark/)

    await page.getByRole('button', { name: 'Theme' }).click()
    await page.getByRole('menuitemradio', { name: 'Light', exact: true }).click()
    await expect(page.locator('html')).not.toHaveClass(/dark/)

    await page.reload()
    await expect(page.locator('html')).not.toHaveClass(/dark/)

    // Proves dark: utilities key off .dark, not prefers-color-scheme: shadcn's
    // input carries dark:bg-input/30, which a media-query dark variant would
    // still apply here since the OS preference is dark.
    await page.goto('/products')
    const background = await page
      .getByRole('searchbox', { name: 'Search' })
      .evaluate((el) => getComputedStyle(el).backgroundColor)
    expect(background).toBe('rgba(0, 0, 0, 0)')
  })

  test('dark mode resolves the background token to its dark value', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('button', { name: 'Theme' }).click()
    await page.getByRole('menuitemradio', { name: 'Dark', exact: true }).click()

    // Asserting a literal hex pins the test to one palette; what matters is
    // that the token resolves to a different, darker value.
    const luminance = await page.evaluate(() => {
      const probe = document.createElement('div')
      probe.style.backgroundColor = 'var(--background)'
      document.body.append(probe)
      const rgb = getComputedStyle(probe).backgroundColor
      probe.remove()
      const [r, g, b] = rgb.match(/[\d.]+/g)!.map(Number)
      return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255
    })
    expect(luminance).toBeLessThan(0.3)
  })
})
