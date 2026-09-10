package internal

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// The stamp is the part every migration shares: a config that predates the
// field is brought up to the current revision and written back, and one that
// is already current is left alone.
func TestRunMigrationsStampsAnUnversionedConfig(t *testing.T) {
	path := t.TempDir() + "/web_config.json"
	m := &Manager{configFile: path, Config: &AppConfig{Servers: []Server{{ID: "s1"}}}}
	m.runMigrations()

	if got := m.Config.SchemaVersion; got != awgConfigSchemaVersion {
		t.Errorf("schema version = %d, want %d", got, awgConfigSchemaVersion)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), fmt.Sprintf(`"schema_version": %d`, awgConfigSchemaVersion)) {
		t.Errorf("schema version not stamped:\n%s", saved)
	}
}

func TestRunMigrationsLeavesACurrentConfigAlone(t *testing.T) {
	path := t.TempDir() + "/web_config.json"
	m := &Manager{configFile: path, Config: &AppConfig{SchemaVersion: awgConfigSchemaVersion}}
	m.runMigrations()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a config that needs nothing must not be rewritten (err %v)", err)
	}
}

// Configs written before 3.1.5 can hold a client ServerID that decayed into a
// slice of an unrelated request path, because fiber recycles the buffer a
// route parameter points into. The server a client is stored under is what
// binds them, so the migration rewrites the copy from it.
func TestMigrationRepairsClientServerIDs(t *testing.T) {
	path := t.TempDir() + "/web_config.json"
	m := &Manager{configFile: path, Config: &AppConfig{SchemaVersion: 3, Servers: []Server{{
		ID: "s1",
		Clients: []Client{
			{ID: "c1", ServerID: "tatus5"},
			{ID: "c2", ServerID: "s1"},
		},
	}, {
		ID:      "s2",
		Clients: []Client{{ID: "c3", ServerID: "i-sett"}},
	}}}}

	m.runMigrations()

	for _, want := range []struct {
		server, client string
	}{{"s1", "c1"}, {"s1", "c2"}, {"s2", "c3"}} {
		client, ok := m.getClientInServer(want.server, want.client)
		if !ok {
			t.Fatalf("client %s vanished from %s", want.client, want.server)
		}
		if client.ServerID != want.server {
			t.Errorf("client %s: ServerID = %q, want %q", want.client, client.ServerID, want.server)
		}
	}
	if got := m.Config.SchemaVersion; got != awgConfigSchemaVersion {
		t.Errorf("schema version = %d, want %d", got, awgConfigSchemaVersion)
	}
}
