import { defineConfig, devices } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  testMatch: "**/*.spec.ts",
  fullyParallel: false,
  workers: 1,
  timeout: 60000,
  expect: { timeout: 10000 },
  retries: process.env.CI ? 1 : 0,
  reporter: "list",
  use: { baseURL: "http://localhost:3100", trace: "retain-on-failure" },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], timezoneId: "America/Los_Angeles" },
    },
  ],
  webServer: [
    {
      command: "go run -tags=e2e ./cmd/e2e",
      cwd: "../backend",
      url: "http://localhost:18080/health",
      timeout: 120000,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: "npm run dev -- --port 3100",
      url: "http://localhost:3100",
      timeout: 120000,
      reuseExistingServer: !process.env.CI,
      env: {
        WAYPOINT_E2E: "1",
        NEXT_PUBLIC_API_BASE_URL: "http://localhost:18080",
        NEXT_PUBLIC_MAPS_BROWSER_API_KEY: "test-key",
        NEXT_TELEMETRY_DISABLED: "1",
      },
    },
  ],
});
