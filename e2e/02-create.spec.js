const { test, expect } = require('@playwright/test');
const { startApp, click, type, scroll, api } = require('./helpers');

test('create a server through the UI', async ({ page, request }) => {
  const errors = await startApp(page);

  // Open the "Create New VPN Server" accordion. It comes up in the simple
  // mode: a name, a port, and everything else generated on submit.
  await click(page, 130, 87);
  await page.screenshot({ path: 'shots/02-form.png' });

  // Tick and untick "Advanced settings" first: the rows it appends to the form
  // and the obfuscation block it reveals have to survive the round trip, and
  // the page-error assertion at the end covers a panic on the way.
  await click(page, 35, 189);
  await page.screenshot({ path: 'shots/02-advanced.png' });
  await scroll(page, 750, 500, 3000);
  await page.screenshot({ path: 'shots/02-advanced-bottom.png' });
  await scroll(page, 750, 500, -3000);
  await click(page, 35, 189);
  await page.screenshot({ path: 'shots/02-simple.png' });

  // The name is the only thing the simple mode has no default for.
  await click(page, 380, 149);
  await type(page, 'E2E Server');
  await page.screenshot({ path: 'shots/03-named.png' });

  await click(page, 87, 267);

  await expect.poll(async () => (await api(request, '/api/servers')).length,
    { timeout: 60_000, intervals: [1000] }).toBe(1);

  const [server] = await api(request, '/api/servers');
  expect(server.name).toBe('E2E Server');

  // Everything below was generated rather than typed.
  expect(server.subnet).toBe('10.0.0.0/24');
  expect(server.mtu).toBe(1420);
  expect(server.dns).toEqual(['8.8.8.8', '1.1.1.1']);
  expect(server.endpoint).not.toBe('');
  expect(server.obfuscation_enabled).toBe(true);
  expect(server.obfuscation_params.RandomTrailers).toBe(true);
  expect(server.obfuscation_params.HeaderProtectionKey).not.toBe('');
  for (const key of ['S1', 'S2', 'S3', 'S4']) {
    expect(server.obfuscation_params[key],
      `${key} must be >= 12 for AWG 3.x header protection`).toBeGreaterThanOrEqual(12);
  }
  expect(server.obfuscation_params.S1 + 56).not.toBe(server.obfuscation_params.S2);

  await page.waitForTimeout(4000);
  await page.screenshot({ path: 'shots/05-created.png' });
  expect(errors, 'uncaught page errors').toEqual([]);
});
