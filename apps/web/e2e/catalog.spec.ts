import { expect, test } from '@playwright/test'
import type { Page } from '@playwright/test'

// A 1x1 PNG, built here so the suite carries no binary fixture.
const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
  'base64',
)

function uniqueSku(): string {
  return `E2E-${Date.now()}`
}

async function createProduct(page: Page, sku: string, title: string, price: string) {
  await page.goto('/products/new')
  await page.getByRole('textbox', { name: 'SKU' }).fill(sku)
  await page.getByRole('textbox', { name: 'Title' }).fill(title)
  await page.getByRole('textbox', { name: 'Price' }).fill(price)
  await page.getByRole('button', { name: 'Create product' }).click()
  await expect(page.getByRole('heading', { name: title })).toBeVisible()
}

test('a product is created a draft, activated, and then searchable', async ({ page }) => {
  const sku = uniqueSku()
  const title = `Ash bread bin ${sku}`

  await createProduct(page, sku, title, '22.15')

  await expect(page.getByText('$22.15')).toBeVisible()
  await expect(page.getByText('Draft', { exact: true })).toBeVisible()
  await page.screenshot({ path: 'screenshots/catalog_detail-draft.png', fullPage: true })

  await page.getByRole('button', { name: 'Activate' }).click()
  await expect(page.getByText('Active', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Activate' })).toHaveCount(0)

  await page.goto('/products')
  await page.getByRole('searchbox', { name: 'Search' }).fill(title)
  await page.getByRole('button', { name: 'Search' }).click()
  await expect(page.getByRole('link', { name: title })).toBeVisible()
  await page.screenshot({ path: 'screenshots/catalog_search.png', fullPage: true })
})

test('an uploaded image is readable afterwards', async ({ page }) => {
  const sku = uniqueSku()
  await createProduct(page, sku, `Photographed mug ${sku}`, '9.00')

  await expect(page.getByText('No images yet')).toBeVisible()

  await page.setInputFiles('input[type=file]', { name: 'mug.png', mimeType: 'image/png', buffer: PNG })

  const image = page.locator('img').first()
  await expect(image).toBeVisible()
  // A 201 alone would not prove this: the presigned URL has to resolve from the
  // browser, which it did not when the store signed the container-internal host.
  await expect
    .poll(() => image.evaluate((node: HTMLImageElement) => node.naturalWidth))
    .toBeGreaterThan(0)
  await page.screenshot({ path: 'screenshots/catalog_detail-image.png', fullPage: true })

  await page.getByRole('button', { name: 'Remove' }).click()
  await expect(page.getByText('No images yet')).toBeVisible()
})

test('the price field refuses anything that is not money', async ({ page }) => {
  await page.goto('/products/new')
  await page.getByRole('button', { name: 'Create product' }).click()
  await expect(page.getByText('Give the product a SKU.')).toBeVisible()
  await expect(page.getByText('Give the product a title.')).toBeVisible()

  await page.getByRole('textbox', { name: 'SKU' }).fill(uniqueSku())
  await page.getByRole('textbox', { name: 'Title' }).fill('Malformed price')
  await page.getByRole('textbox', { name: 'Price' }).fill('22.155')
  await page.getByRole('button', { name: 'Create product' }).click()
  await expect(page.getByText('Enter an amount in USD')).toBeVisible()
  await page.screenshot({ path: 'screenshots/catalog_form-error.png', fullPage: true })
})

test('a duplicate SKU is reported on the SKU field', async ({ page }) => {
  const sku = uniqueSku()
  await createProduct(page, sku, `First ${sku}`, '5.00')

  await page.goto('/products/new')
  await page.getByRole('textbox', { name: 'SKU' }).fill(sku)
  await page.getByRole('textbox', { name: 'Title' }).fill('Second')
  await page.getByRole('button', { name: 'Create product' }).click()
  await expect(page.getByText('Another product already uses this SKU.')).toBeVisible()
})

test.describe('narrow', () => {
  test.use({ viewport: { width: 400, height: 900 } })

  test('the product screens hold at 400px', async ({ page }) => {
    const sku = uniqueSku()
    await createProduct(page, sku, `Narrow ${sku}`, '12.00')
    await page.screenshot({ path: 'screenshots/narrow_catalog_detail.png', fullPage: true })

    await page.goto('/products')
    await expect(page.getByRole('heading', { name: 'Products' })).toBeVisible()
    await page.screenshot({ path: 'screenshots/narrow_catalog_list.png', fullPage: true })
  })
})
