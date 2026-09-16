import { defineConfig, devices } from '@playwright/test'

// The identity screens now call the real gateway (localhost:8080), not a mock — every project
// except "setup" and "anonymous" carries the owner's session cookie so those routes render.
const authFile = 'playwright/.auth/owner.json'

const devServer = process.env.BASE_URL ? undefined : 'http://localhost:5173'
const baseURL = process.env.BASE_URL ?? devServer

export default defineConfig({
  testDir: './e2e',
  use: { baseURL, ...devices['Desktop Chrome'] },
  // BASE_URL points the suite at the gateway-served build instead. A login
  // against the dev server does not prove the app works where it is deployed.
  webServer: devServer
    ? { command: 'npm run dev -- --port 5173', url: devServer, reuseExistingServer: true }
    : undefined,
  projects: [
    { name: 'setup', testMatch: /auth\.setup\.ts/ },
    {
      name: 'authenticated',
      testMatch: /(shell|interaction|catalog|order|urlstate|coverage)\.spec\.ts/,
      dependencies: ['setup'],
      use: { storageState: authFile },
    },
    {
      name: 'anonymous',
      testMatch: /identity\.spec\.ts/,
    },
  ],
})
