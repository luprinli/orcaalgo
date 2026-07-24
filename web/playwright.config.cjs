const { defineConfig } = require('@playwright/test');
module.exports = defineConfig({
  testDir: './e2e',
  use: {
    baseURL: 'http://localhost:5173',
    headless: true,
  },
  projects: [
    {
      name: 'chromium',
      use: {
        browserName: 'chromium',
        launchOptions: {
          args: ['--disable-extensions'],
        },
      },
    },
  ],
});
