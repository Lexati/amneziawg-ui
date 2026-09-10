package internal

import (
	"errors"
	"testing"
)

// The set a new client starts from has to satisfy the rules the app enforces
// on everyone else.
func TestDefaultISettingsAreValid(t *testing.T) {
	if err := validateISettings(defaultISettings()); err != nil {
		t.Fatalf("the built-in defaults do not validate: %v", err)
	}
}

// A malformed signature packet is a bad request, not a server error, and it is
// rejected before any client is created.
func TestAddClientRejectsAMalformedSignaturePacket(t *testing.T) {
	m := newClientManager(t)

	_, _, err := m.AddClient("s1", "alice", true, map[string]string{"i1": "<b 0xabc>"}, "")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	if got := m.clientCount(); got != 0 {
		t.Errorf("nothing should have been created, got %d clients", got)
	}
}
