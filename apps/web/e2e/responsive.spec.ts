import { expect, test } from '@playwright/test'
import { settled } from './select'

// A wide table must scroll inside its own container. Without min-w-0 the
// content region takes the table's intrinsic width and the whole page scrolls
// sideways, which at 1024px hid the Price and Added columns entirely.
for (const width of [1440, 1024, 768, 420]) {
  test(`the page does not scroll sideways at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 })
    await page.goto('/products')
    await settled(page)

    const { scrollWidth, clientWidth } = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth,
    }))
    expect(scrollWidth, `${width}px overflows by ${scrollWidth - clientWidth}px`).toBeLessThanOrEqual(clientWidth + 1)
  })
}
