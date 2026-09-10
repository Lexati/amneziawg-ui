# Browser tests

End-to-end tests that drive the real WebAssembly UI in Chrome with Playwright
and assert against the backend's REST API.

The frontend renders into a single `<canvas>`, so there is nothing to select
by CSS: the specs click by coordinate (at a fixed 1500x950 viewport) and check
the result through `/api/...`. Screenshots of every step land in `shots/`.

## Running

From the repository root:

```sh
make e2e
```

That is the whole thing. `make e2e` depends on `make e2e-reset`, which
recreates the backend from scratch before every run: it removes the
`awgui-test` container and its `awgui-test-data` volume, rebuilds the image
from the working tree, starts it on port 51836 and waits for `/status` to
answer.

The reset is why the suite gets its own container, volume and port rather than
the stack `make run` leaves behind: the specs share one backend and build on
each other's state - `02` creates the server that `03` adds a client to - so
they run in file order with a single worker and expect to start against an
instance with no servers configured. Wiping state before a test run must never
discard the servers you are working on.

The instance stays up after the run, so a failed spec can be inspected in the
browser at <http://localhost:51836> (admin / changeme). `make e2e-down`
removes the container and the volume.

Override any of it with the Makefile variables `E2E_PORT`, `E2E_NAME`,
`E2E_VOLUME`, `E2E_IMAGE`, `E2E_URL`, `E2E_TIMEOUT`.

To run the specs against a backend you started yourself, skip the Makefile:

```sh
cd e2e
npm install
AWG_URL=http://localhost:54845 npx playwright test
```

`AWG_URL` defaults to `http://localhost:51836`. Whatever it points at needs
`WEB_UI_USER=admin` and `WEB_UI_PASSWORD=BXugPWxEEEhj3HNh/kV4ll0YhzYPkKCJWILlimJI/IY=`
- the base64 SHA-256 of `changeme`, which is what the specs log in with.

The config reuses the Chrome installed on the machine (`channel: 'chrome'`)
rather than Playwright's own download; either way the browser needs WebGL,
which the app requires.
