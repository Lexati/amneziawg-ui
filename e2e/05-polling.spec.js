const { test, expect } = require('@playwright/test');
const { startApp } = require('./helpers');

// The counters on the cards come from a polling loop, not from an event feed:
// if that loop ever dies, the page keeps showing stale numbers with nothing
// to tell it apart from a quiet server. Make sure it keeps ticking.
test('the page keeps polling the traffic snapshot', async ({ page }) => {
  const polls = [];
  page.on('request', (request) => {
    if (request.url().endsWith('/api/traffic')) polls.push(request.url());
  });

  await startApp(page);
  await page.waitForTimeout(12_000);

  expect(polls.length, 'traffic polling stopped').toBeGreaterThanOrEqual(2);
});
