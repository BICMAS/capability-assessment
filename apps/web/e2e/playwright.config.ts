// Opt-in browser scenario. Install @playwright/test, then run npm run test:e2e
// with the web app, gateway, and Go services already running.
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  timeout: 60_000,
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL ?? "http://127.0.0.1:3000",
    ...devices["Desktop Chrome"],
  },
});
