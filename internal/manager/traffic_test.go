package manager

import (
	"testing"
	"time"

	"amneziawg-web-ui/web-ui/api"
)

func TestTrafficSnapshotJoinsPeersWithClients(t *testing.T) {
	m, run := newTestManager(t)
	seen, _, _ := m.AddClient("s1", api.AddClientRequest{Name: "seen"})
	quiet, _, _ := m.AddClient("s1", api.AddClientRequest{Name: "quiet"})

	run.Stub("/usr/bin/awg show wg-test-absent",
		"interface: wg-test-absent\n\npeer: "+seen.ClientPublicKey+"\n  endpoint: 203.0.113.5:1\n  transfer: 1 KiB received, 2 KiB sent\n")
	run.Stub("ifconfig wg-test-absent", "RX bytes:10 (10.0 B)  TX bytes:20 (20.0 B)")

	snap := m.TrafficSnapshot()
	peers := snap.ClientTraffic["s1"]
	if peers[seen.ID].Received != "1 KiB" || peers[seen.ID].Endpoint != "203.0.113.5:1" {
		t.Errorf("seen client = %+v", peers[seen.ID])
	}
	if peers[quiet.ID].Received != "0 B" || peers[quiet.ID].LastHandshake != "Never" {
		t.Errorf("quiet client = %+v", peers[quiet.ID])
	}
	got := snap.ServerTraffic["s1"]
	if got.UptimeSeconds < 0 {
		t.Errorf("uptime = %v, want a running counter", got.UptimeSeconds)
	}
	got.UptimeSeconds = 0
	if want := (api.InterfaceTraffic{RX: "10.0 B", TX: "20.0 B", RXBytes: 10, TXBytes: 20}); got != want {
		t.Errorf("server traffic = %+v, want %+v", got, want)
	}
}

// The uptime counts from the moment the interface was seen coming up and
// survives the status cache expiring; a stop resets it.
func TestInterfaceUptimeCountsFromTheFirstSighting(t *testing.T) {
	m, _ := newTestManager(t)
	iface := m.cfg.Servers[0].Interface

	if got := m.interfaceUptime(iface); got != 0 {
		t.Errorf("uptime of an unseen interface = %v, want 0", got)
	}

	m.noteServerStatus(iface, "running")
	m.statusMu.Lock()
	seen := m.statuses[iface]
	seen.since = seen.since.Add(-time.Hour)
	seen.at = seen.at.Add(-time.Hour)
	m.statuses[iface] = seen
	m.statusMu.Unlock()

	m.noteServerStatus(iface, "running")
	if got := m.interfaceUptime(iface); got < 3600 {
		t.Errorf("uptime after a repeat sighting = %v, want the original hour kept", got)
	}

	m.noteServerStatus(iface, "stopped")
	if got := m.interfaceUptime(iface); got != 0 {
		t.Errorf("uptime of a stopped interface = %v, want 0", got)
	}
}

// A server whose interface is down contributes nothing rather than zeros, so
// the page keeps showing the last counters it had.
func TestTrafficSnapshotSkipsDownInterfaces(t *testing.T) {
	m, _ := newTestManager(t)
	snap := m.TrafficSnapshot()
	if len(snap.ClientTraffic) != 0 || len(snap.ServerTraffic) != 0 {
		t.Errorf("down interface reported traffic: %+v", snap)
	}
}

// The host gauges ride along in the same poll. What exactly they read is the
// sysinfo package's business; here it is enough that the real procfs gets
// through to the response.
func TestTrafficSnapshotCarriesSystemMetrics(t *testing.T) {
	m, _ := newTestManager(t)
	sys := m.TrafficSnapshot().System
	if sys.CPU.Cores == 0 || sys.CPU.TotalSeconds == 0 || sys.Memory.Total == 0 {
		t.Errorf("system metrics = %+v", sys)
	}
}
