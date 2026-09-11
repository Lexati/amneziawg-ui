package store

import (
	"testing"

	"amneziawg-web-ui/web-ui/api"
)

// The stamp is the part every migration shares: a config that predates the
// field is brought up to the current revision and reported as changed, and
// one that is already current is left alone.
func TestMigrateStampsAnUnversionedConfig(t *testing.T) {
	cfg := &AppConfig{Servers: []api.Server{{ID: "s1"}}}
	if !Migrate(cfg) {
		t.Error("an unversioned config must be reported as changed")
	}
	if got := cfg.SchemaVersion; got != SchemaVersion {
		t.Errorf("schema version = %d, want %d", got, SchemaVersion)
	}
}

func TestMigrateLeavesACurrentConfigAlone(t *testing.T) {
	cfg := &AppConfig{SchemaVersion: SchemaVersion}
	if Migrate(cfg) {
		t.Error("a config that needs nothing must not be reported as changed")
	}
}

// Configs written before 3.1.5 can hold a client ServerID that decayed into a
// slice of an unrelated request path, because fiber recycles the buffer a
// route parameter points into. The server a client is stored under is what
// binds them, so the migration rewrites the copy from it.
func TestMigrationRepairsClientServerIDs(t *testing.T) {
	cfg := &AppConfig{SchemaVersion: 3, Servers: []api.Server{{
		ID: "s1",
		Clients: []api.Client{
			{ID: "c1", ServerID: "tatus5"},
			{ID: "c2", ServerID: "s1"},
		},
	}, {
		ID:      "s2",
		Clients: []api.Client{{ID: "c3", ServerID: "i-sett"}},
	}}}

	Migrate(cfg)

	for _, srv := range cfg.Servers {
		for _, c := range srv.Clients {
			if c.ServerID != srv.ID {
				t.Errorf("client %s: ServerID = %q, want %q", c.ID, c.ServerID, srv.ID)
			}
		}
	}
	if got := cfg.SchemaVersion; got != SchemaVersion {
		t.Errorf("schema version = %d, want %d", got, SchemaVersion)
	}
}
