const { test, expect } = require('@playwright/test');
const { startApp, click, api } = require('./helpers');

test('client config dialog shows the AmneziaVPN link and the .conf QR code', async ({ page, request }) => {
  const errors = await startApp(page);

  const [server] = await api(request, '/api/servers');
  const [client] = await api(request, `/api/servers/${server.id}/clients`);

  // The backend has to offer both views the dialog renders, and the download
  // has to carry the full config with the obfuscation parameters.
  const configs = await api(request, `/api/servers/${server.id}/clients/${client.id}/config-both`);
  expect(configs.clean_config).toContain('[Interface]');
  const link = await api(request, `/api/servers/${server.id}/clients/${client.id}/link`);
  expect(link.vpn_url).toMatch(/^vpn:\/\//);

  const download = await request.get(`/api/servers/${server.id}/clients/${client.id}/config`);
  expect(download.ok()).toBeTruthy();
  expect(await download.text()).toContain('Jc =');

  // "QR / config" on the client row. The dialog opens on the .conf tab,
  // which is the one that gets a QR code.
  await click(page, 1240, 340);
  await page.waitForTimeout(2500);
  await page.screenshot({ path: 'shots/08-qr-conf.png' });

  // Switch to the AmneziaVPN link tab.
  await click(page, 407, 205);
  await page.waitForTimeout(1500);
  await page.screenshot({ path: 'shots/09-qr-link.png' });

  // Close, then open the server configuration dialog.
  await click(page, 749, 800);
  await page.waitForTimeout(1000);
  await click(page, 346, 267);
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'shots/11-server-config.png' });

  expect(errors, 'uncaught page errors').toEqual([]);
});
