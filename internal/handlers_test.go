package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// newTestAPI wires the real handlers onto a Fiber app backed by a manager
// whose state lives in a temp dir.
func newTestAPI(t *testing.T) (*fiber.App, *Manager) {
	t.Helper()
	m := newClientManager(t)
	app := fiber.New(FiberConfig())
	NewHandlers(m).RegisterRoutes(app)
	return app, m
}

func call(t *testing.T, app *fiber.App, method, path, body string) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, data
}

// "No such server" and "the interface refused to come up" used to be the same
// answer: 404 with "server not found or failed to start".
func TestMissingServerIs404AndAFailedStartIs500(t *testing.T) {
	app, _ := newTestAPI(t)

	status, body := call(t, app, http.MethodPost, "/api/servers/nope/start", "")
	if status != http.StatusNotFound {
		t.Errorf("unknown server: status = %d, want 404 (%s)", status, body)
	}

	// s1 exists; awg-quick is not installed here, so bringing it up fails.
	status, body = call(t, app, http.MethodPost, "/api/servers/s1/start", "")
	if status != http.StatusInternalServerError {
		t.Errorf("failed start: status = %d, want 500 (%s)", status, body)
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil || errResp.Error == "" {
		t.Errorf("error payload = %s (%v)", body, err)
	}
}

// Creating a server used to answer 400 whatever went wrong. A port another
// server already listens on is a clash with the current state, not a
// malformed request, and an invalid MTU is still the request's own fault.
func TestCreateServerSeparatesAPortClashFromABadValue(t *testing.T) {
	app, _ := newTestAPI(t) // "srv" is already on 54844

	status, body := call(t, app, http.MethodPost, "/api/servers",
		`{"name":"second","port":54844}`)
	if status != http.StatusConflict {
		t.Errorf("duplicate port: status = %d, want 409 (%s)", status, body)
	}
	if !strings.Contains(string(body), "54844") {
		t.Errorf("error payload should name the port: %s", body)
	}

	status, body = call(t, app, http.MethodPost, "/api/servers",
		`{"name":"second","port":54845,"mtu":9000}`)
	if status != http.StatusBadRequest {
		t.Errorf("bad MTU: status = %d, want 400 (%s)", status, body)
	}
}

func TestMalformedBodyIsRejected(t *testing.T) {
	app, m := newTestAPI(t)
	client, _, err := m.AddClient("s1", "alice", false, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"/api/servers/s1/clients",
		"/api/servers/s1/clients/" + client.ID + "/allowed-ips",
	} {
		method := http.MethodPost
		if strings.HasSuffix(path, "allowed-ips") {
			method = http.MethodPut
		}
		status, body := call(t, app, method, path, "{not json")
		if status != http.StatusBadRequest {
			t.Errorf("%s %s: status = %d, want 400 (%s)", method, path, status, body)
		}
	}

	// A body that parses but leaves everything at its default is still fine.
	status, _ := call(t, app, http.MethodPut,
		"/api/servers/s1/clients/"+client.ID+"/allowed-ips", `{}`)
	if status != http.StatusOK {
		t.Errorf("empty object: status = %d, want 200", status)
	}
}

func TestActivatingAnActiveClientIsAConflict(t *testing.T) {
	app, m := newTestAPI(t)
	client, _, err := m.AddClient("s1", "alice", false, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	status, body := call(t, app, http.MethodPost,
		"/api/servers/s1/clients/"+client.ID+"/activate", "")
	if status != http.StatusConflict {
		t.Errorf("status = %d, want 409 (%s)", status, body)
	}
}

// The frontend parses these payloads into the types in web-ui/api; the
// backend must produce exactly those shapes.
func TestResponsesMatchTheDeclaredPayloads(t *testing.T) {
	app, m := newTestAPI(t)
	client, _, err := m.AddClient("s1", "alice", true, nil, "")
	if err != nil {
		t.Fatal(err)
	}

	status, body := call(t, app, http.MethodGet, "/api/servers/s1/info", "")
	if status != http.StatusOK {
		t.Fatalf("info: status = %d (%s)", status, body)
	}
	var info ServerInfo
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatal(err)
	}
	if info.ID != "s1" || info.ClientsCount != 1 || len(info.Clients) != 1 {
		t.Errorf("ServerInfo = %+v", info)
	}
	if info.DefaultISettings["i1"] != DefaultI1 {
		t.Errorf("default I-settings missing from ServerInfo")
	}
	if info.Status != "stopped" {
		t.Errorf("status = %q, want the observed stopped", info.Status)
	}

	status, body = call(t, app, http.MethodGet,
		"/api/servers/s1/clients/"+client.ID+"/config-both", "")
	if status != http.StatusOK {
		t.Fatalf("config-both: status = %d (%s)", status, body)
	}
	var configs ClientConfigs
	if err := json.Unmarshal(body, &configs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(configs.CleanConfig, "[Interface]") || configs.FullLength == 0 {
		t.Errorf("ClientConfigs = %+v", configs)
	}

	status, body = call(t, app, http.MethodDelete, "/api/servers/s1/clients/"+client.ID, "")
	if status != http.StatusOK {
		t.Fatalf("delete: status = %d (%s)", status, body)
	}
	var action ActionResult
	if err := json.Unmarshal(body, &action); err != nil {
		t.Fatal(err)
	}
	if action.Status != "deleted" || action.ClientID != client.ID {
		t.Errorf("ActionResult = %+v", action)
	}

	status, body = call(t, app, http.MethodGet, "/api/system/status", "")
	if status != http.StatusOK {
		t.Fatalf("system status: status = %d (%s)", status, body)
	}
	var sys SystemStatus
	if err := json.Unmarshal(body, &sys); err != nil {
		t.Fatal(err)
	}
	if sys.TotalServers != 1 || sys.TotalClients != 0 {
		t.Errorf("SystemStatus = %+v", sys)
	}
}

// Starting a server whose interface is already up is a conflict, not a
// failure: awg-quick would just error out with "File exists".
func TestStartingARunningServerIsAConflict(t *testing.T) {
	app, m := newTestAPI(t)
	m.noteServerStatus(m.Config.Servers[0].Interface, "running")

	status, body := call(t, app, http.MethodPost, "/api/servers/s1/start", "")
	if status != http.StatusConflict {
		t.Errorf("start: status = %d, want 409 (%s)", status, body)
	}

	m.noteServerStatus(m.Config.Servers[0].Interface, "stopped")
	status, body = call(t, app, http.MethodPost, "/api/servers/s1/stop", "")
	if status != http.StatusConflict {
		t.Errorf("stop: status = %d, want 409 (%s)", status, body)
	}
}

// A client's ServerID is taken straight off the route and kept in the config
// for as long as the client exists, which is exactly the shape that broke
// once: fiber hands route parameters back as strings pointing into the
// request buffer it recycles after the handler returns, so without
// FiberConfig's Immutable the stored id came back as a slice of an unrelated
// later request ("i-sett", off some .../i-settings path) while the file on
// disk, written before the recycling, still looked right.
//
// app.Test does not recycle contexts the way a real listener does, so the
// traffic below does not reproduce that corruption - only a running server
// did. This guards the invariant the fix restored, not the mechanism.
func TestStoredClientServerIDSurvivesLaterRequests(t *testing.T) {
	app, m := newTestAPI(t)

	status, body := call(t, app, http.MethodPost, "/api/servers/s1/clients", `{"name":"laptop"}`)
	if status != http.StatusOK {
		t.Fatalf("add client: %d %s", status, body)
	}

	var added ClientResult
	if err := json.Unmarshal(body, &added); err != nil {
		t.Fatal(err)
	}

	for range 20 {
		call(t, app, http.MethodPut, "/api/servers/s1/clients/"+added.Client.ID+"/i-settings",
			`{"apply_i_settings":false}`)
		call(t, app, http.MethodGet, "/api/servers/s1/info", "")
	}

	stored, ok := m.getClientInServer("s1", added.Client.ID)
	if !ok {
		t.Fatal("client vanished from its server")
	}
	if stored.ServerID != "s1" {
		t.Errorf("stored ServerID = %q, want \"s1\"", stored.ServerID)
	}
}
