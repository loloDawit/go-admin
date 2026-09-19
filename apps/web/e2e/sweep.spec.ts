import { expect, test } from '@playwright/test'
import { settled } from './select'

const ROUTES = ['/', '/orders', '/products', '/customers', '/staff', '/roles', '/permissions', '/profile', '/settings', '/products/new', '/kit', '/403', '/nope']
const WIDTHS = [1930, 1440, 1024, 420]

// A sweep, not a spot check: every route, both themes, four widths, asserting
// the things that have actually gone wrong in this project.
for (const theme of ['light', 'dark'] as const) {
  test.describe(theme, () => {
    for (const width of WIDTHS) {
      test(`${width}px`, async ({ page }) => {
        await page.setViewportSize({ width, height: 900 })
        const problems: string[] = []
        const consoleErrors: string[] = []
        page.on('console', (m) => m.type() === 'error' && consoleErrors.push(m.text()))

        await page.goto('/')
        await page.evaluate((t) => localStorage.setItem('go-admin-theme', t), theme)

        for (const route of ROUTES) {
          await page.goto(route)
          await settled(page)

          const report = await page.evaluate(() => {
            const de = document.documentElement
            const out: string[] = []
            if (de.scrollWidth > de.clientWidth + 1) {
              out.push(`overflows by ${de.scrollWidth - de.clientWidth}px`)
            }
            // A border painting currentColor instead of the token is the bug
            // that went unnoticed site-wide.
            const token = getComputedStyle(de).getPropertyValue('--border').trim()
            for (const el of Array.from(document.querySelectorAll('[data-slot="card"], table, header'))) {
              const s = getComputedStyle(el as HTMLElement)
              if (parseFloat(s.borderTopWidth) > 0 && s.borderTopColor.includes('0.145')) {
                out.push(`near-black border on ${(el as HTMLElement).dataset.slot ?? el.tagName}`)
                break
              }
            }
            if (!token) out.push('no --border token')
            return out
          })
          for (const r of report) problems.push(`${route}: ${r}`)
        }

        const realErrors = consoleErrors.filter((e) => !/favicon|Download the React/i.test(e))
        expect(problems, problems.join('\n')).toEqual([])
        expect(realErrors, realErrors.join('\n')).toEqual([])
      })
    }
  })
}
