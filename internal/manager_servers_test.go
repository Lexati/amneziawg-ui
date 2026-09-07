package internal

import (
	"errors"
	"strings"
	"testing"
)

// TestCreateServerRejectsAPortAnotherServerAlreadyUses covers the check that
// runs before anything is written: two interfaces on one UDP port would only
// fail much later, when the second one cannot bind.
func TestCreateServerRejectsAPortAnotherServerAlreadyUses(t *testing.T) {
	m := newClientManager(t) // its one server, "srv", listens on 54844

	_, err := m.CreateServer(CreateServerRequest{Name: "second", Port: 54844, MTU: 1420})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if !strings.Contains(err.Error(), `"srv"`) {
		t.Fatalf("error should name the server holding the port, got %v", err)
	}
	if got := len(m.Config.Servers); got != 1 {
		t.Fatalf("nothing should have been created, got %d servers", got)
	}
}

// The panel's own port counts as taken too: publishing one number for both
// the web UI and a tunnel is a trap even though TCP and UDP would coexist.
func TestCreateServerRejectsTheWebUIsOwnPort(t *testing.T) {
	m := newClientManager(t)
	m.WebUIPort = "54845"

	_, err := m.CreateServer(CreateServerRequest{Name: "second", Port: 54845, MTU: 1420})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if !strings.Contains(err.Error(), "web UI") {
		t.Fatalf("error should say what holds the port, got %v", err)
	}
}

func TestPortInUse(t *testing.T) {
	m := newClientManager(t)
	m.WebUIPort = "54845"

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
