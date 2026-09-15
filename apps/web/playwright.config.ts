import { defineConfig, devices } from '@playwright/test'

// The identity screens now call the real gateway (localhost:8080), not a mock — every project
// except "setup" and "anonymous" carries the owner's session cookie so those routes render.
const authFile = 'playwright/.auth/owner.json'

export default defineConfig({
  testDir: './e2e',
  use: { baseURL: 'http://localhost:5173', ...devices['Desktop Chrome'] },
  webServer: {
    command: 'npm run dev -- --port 5173',
    url: 'http://localhost:5173',
    reuseExistingServer: true,
  },
  projects: [
    { name: 'setup', testMatch: /auth\.setup\.ts/ },
    {
      name: 'authenticated',
      testMatch: /(shell|interaction)\.spec\.ts/,
      dependencies: ['setup'],
      use: { storageState: authFile },
    },
    {
      name: 'anonymous',
      testMatch: /identity\.spec\.ts/,
    },
  ],
})
