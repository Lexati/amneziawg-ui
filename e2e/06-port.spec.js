const { test, expect } = require('@playwright/test');
const { startApp, click, type, api } = require('./helpers');

// The form knows which ports are already spoken for - every existing server's,
// plus the panel's own - and refuses one before it reaches the backend.
test('a port another server already uses is rejected in the form', async ({ page, request }) => {
  const errors = await startApp(page);

  const before = await api(request, '/api/servers');
  expect(before.length, 'the create spec must run first').toBeGreaterThan(0);
  const taken = before[0].port;

  let posts = 0;
  page.on('request', (req) => {
    if (req.method() === 'POST' && new URL(req.url()).pathname === '/api/servers') {
      posts++;
    }
  });

  await click(page, 130, 143);

  await click(page, 380, 205);
  await type(page, 'Duplicate Port');

  // Replace whatever free port the form offered with one that is not.
  await click(page, 1100, 205);
  await page.keyboard.press('Control+a');
  await page.keyboard.press('Backspace');
  await type(page, String(taken));
  await page.screenshot({ path: 'shots/08-port-taken.png' });

  await click(page, 87, 323);
  await page.waitForTimeout(3000);
  await page.screenshot({ path: 'shots/09-port-rejected.png' });

  expect(posts, 'the request must not leave the browser at all').toBe(0);
  expect((await api(request, '/api/servers')).length,
    'no server should have been created').toBe(before.length);
  expect(errors, 'uncaught page errors').toEqual([]);
});
