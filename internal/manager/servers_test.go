package manager

import (
	"errors"
	"os"
	"strings"
	"testing"

	"amneziawg-web-ui/web-ui/api"
)

// TestCreateServerRejectsAPortAnotherServerAlreadyUses covers the check that
// runs before anything is written: two interfaces on one UDP port would only
// fail much later, when the second one cannot bind.
func TestCreateServerRejectsAPortAnotherServerAlreadyUses(t *testing.T) {
	m, _ := newTestManager(t) // its one server, "srv", listens on 54844

	_, err := m.CreateServer(api.CreateServerRequest{Name: "second", Port: 54844, MTU: 1420})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if !strings.Contains(err.Error(), `"srv"`) {
		t.Fatalf("error should name the server holding the port, got %v", err)
	}
	if got := len(m.cfg.Servers); got != 1 {
		t.Fatalf("nothing should have been created, got %d servers", got)
	}
}

// The panel's own port counts as taken too: publishing one number for both
// the web UI and a tunnel is a trap even though TCP and UDP would coexist.
func TestCreateServerRejectsTheWebUIsOwnPort(t *testing.T) {
	m, _ := newTestManager(t)

	_, err := m.CreateServer(api.CreateServerRequest{Name: "second", Port: 54845, MTU: 1420})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if !strings.Contains(err.Error(), "web UI") {
		t.Fatalf("error should say what holds the port, got %v", err)
	}
}

func TestPortInUse(t *testing.T) {
	m, _ := newTestManager(t)

	if holder, ok := m.portInUse(54844); !ok || holder != `server "srv"` {
		t.Errorf(`portInUse(54844) = %q, %v; want server "srv", true`, holder, ok)
	}
	if holder, ok := m.portInUse(54845); !ok || holder != "the web UI" {
		t.Errorf(`portInUse(54845) = %q, %v; want the web UI, true`, holder, ok)
	}
	if holder, ok := m.portInUse(54846); ok {
		t.Errorf("portInUse(54846) = %q, %v; want the port to be free", holder, ok)
	}
}

// A created server gets a .conf whose [Interface] carries the same
// obfuscation lines its clients will, and is brought up when asked.
func TestCreateServerWritesTheConfAndAutoStarts(t *testing.T) {
	m, run := newTestManager(t)
	run.Stub("awg genkey", "SPRIV").Stub("echo 'SPRIV' | awg pubkey", "SPUB")
	run.Stub("/usr/bin/awg-quick up wg-", "")
	yes := true

	srv, err := m.CreateServer(api.CreateServerRequest{Name: "second", Port: 54846, Subnet: "10.0.2.0/24", AutoStart: &yes})
	if err != nil {
		t.Fatal(err)
	}
	if srv.ServerPublicKey != "SPUB" || srv.ServerIP != "10.0.2.1" || !srv.ObfuscationEnabled {
		t.Errorf("server = %+v", srv)
	}

	conf, err := os.ReadFile(srv.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"PrivateKey = SPRIV", "Address = 10.0.2.1/24", "ListenPort = 54846", "HeaderProtectionKey = "} {
		if !strings.Contains(string(conf), line) {
			t.Errorf("conf missing %q:\n%s", line, conf)
		}
	}

	if !run.Ran("/usr/bin/awg-quick up " + srv.Interface) {
		t.Errorf("the server was not brought up: %v", run.Commands)
	}
	if got := m.ServerStatus(srv.ID); got != "running" {
		t.Errorf("status after auto-start = %q", got)
	}
}

func TestDeleteServerStopsARunningInterfaceFirst(t *testing.T) {
	m, run := newTestManager(t)
	run.Stub("ip link show wg-test-absent", "wg-test-absent: state UNKNOWN")
	run.Stub("/usr/bin/awg-quick down wg-test-absent", "")
	confPath := m.cfg.Servers[0].ConfigPath

	if err := m.DeleteServer("s1"); err != nil {
		t.Fatal(err)
	}
	if !run.Ran("/usr/bin/awg-quick down wg-test-absent") {
		t.Errorf("the interface was left up: %v", run.Commands)
	}
	if _, err := os.Stat(confPath); !os.IsNotExist(err) {
		t.Errorf("the .conf was left behind (err %v)", err)
	}
	if _, ok := m.Server("s1"); ok {
		t.Error("the server is still listed")
	}
}
