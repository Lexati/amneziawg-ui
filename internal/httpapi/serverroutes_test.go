package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"amneziawg-web-ui/web-ui/api"
)

func TestUpdateClientServerRoutesRewritesTheServerPeerBlock(t *testing.T) {
	app, m, _ := newTestAPI(t)
	client := addClient(t, m, "alice", false)

	status, body := call(t, app, http.MethodPut,
		"/api/servers/s1/clients/"+client.ID+"/server-routes",
		`{"server_routes":"192.168.30.0/24, 10.50.0.0/16"}`)
	if status != http.StatusOK {
		t.Fatalf("update server-routes: status = %d (%s)", status, body)
	}

	var result api.ClientResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if result.Client.ServerRoutes != "192.168.30.0/24, 10.50.0.0/16" {
		t.Errorf("ServerRoutes = %q", result.Client.ServerRoutes)
	}

	srv, ok := m.Server("s1")
	if !ok {
		t.Fatal("server vanished")
	}
	conf, err := os.ReadFile(srv.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "AllowedIPs = " + client.ClientIP + "/32, 192.168.30.0/24, 10.50.0.0/16"
	if !strings.Contains(string(conf), want) {
		t.Errorf("server .conf peer AllowedIPs not updated:\n%s\nwant to contain %q", conf, want)
	}
}

func TestUpdateClientServerRoutesRejectsBadCIDR(t *testing.T) {
	app, m, _ := newTestAPI(t)
	client := addClient(t, m, "alice", false)

	status, body := call(t, app, http.MethodPut,
		"/api/servers/s1/clients/"+client.ID+"/server-routes",
		`{"server_routes":"not-a-network"}`)
	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (%s)", status, body)
	}
}
