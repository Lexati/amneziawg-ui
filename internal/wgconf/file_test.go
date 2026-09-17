package wgconf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"amneziawg-web-ui/web-ui/api"
)

func newServerConf(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wg-s1.conf")
	iface := ServerInterface{PrivateKey: "PRIV", Address: "10.0.1.1/24", ListenPort: 54844, MTU: 1280,
		Obfuscation: &api.ObfuscationParams{S1: 50}}
	if err := WriteServerConf(path, iface); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestParkAndRestorePeerRoundTrip(t *testing.T) {
	path := newServerConf(t)
	alice := &api.Client{ID: "aaa", Name: "alice", ClientPublicKey: "APUB", PresharedKey: "APSK"}
	bob := &api.Client{ID: "bbb", Name: "bob", ClientPublicKey: "BPUB", PresharedKey: "BPSK"}
	for _, c := range []*api.Client{alice, bob} {
		if err := AppendPeer(path, c, "10.0.1.2/32"); err != nil {
			t.Fatal(err)
		}
	}

	if err := ParkPeer(path, alice); err != nil {
		t.Fatal(err)
	}
	conf := read(t, path)
	if strings.Contains(conf, "APUB") || !strings.Contains(conf, "BPUB") || !strings.Contains(conf, "S1 = 50") {
		t.Fatalf("after park:\n%s", conf)
	}
	parked := filepath.Join(SuspendedDir(path), "aaa.conf")
	if _, err := os.Stat(parked); err != nil {
		t.Fatalf("parked block missing: %v", err)
	}

	alice.Name = "alice-renamed"
	if err := RestorePeer(path, alice); err != nil {
		t.Fatal(err)
	}
	conf = read(t, path)
	if !strings.Contains(conf, "APUB") || !strings.Contains(conf, PeerMarker("alice-renamed", "aaa")) {
		t.Fatalf("after restore:\n%s", conf)
	}
	if strings.Count(conf, "[Peer]") != 2 {
		t.Fatalf("peer count after round trip:\n%s", conf)
	}
	if _, err := os.Stat(parked); !os.IsNotExist(err) {
		t.Errorf("parked copy left behind (err %v)", err)
	}
}

func TestRestoringAnUnparkedPeerFails(t *testing.T) {
	path := newServerConf(t)
	if err := RestorePeer(path, &api.Client{ID: "nope"}); err == nil {
		t.Error("restoring a peer that was never parked must fail")
	}
}

func TestRemovePeerOfAClientWithoutABlockIsANoOp(t *testing.T) {
	path := newServerConf(t)
	before := read(t, path)
	block, err := RemovePeer(path, &api.Client{ID: "ghost"})
	if err != nil || block != nil {
		t.Errorf("RemovePeer = %v, %v", block, err)
	}
	if read(t, path) != before {
		t.Error("the file was rewritten for nothing")
	}
}

func TestRewritePeerAllowedIPsReplacesOnlyThatPeer(t *testing.T) {
	path := newServerConf(t)
	alice := &api.Client{ID: "aaa", Name: "alice", ClientPublicKey: "APUB", PresharedKey: "APSK"}
	bob := &api.Client{ID: "bbb", Name: "bob", ClientPublicKey: "BPUB", PresharedKey: "BPSK"}
	for _, c := range []*api.Client{alice, bob} {
		if err := AppendPeer(path, c, "10.0.1.2/32"); err != nil {
			t.Fatal(err)
		}
	}

	found, err := RewritePeerAllowedIPs(path, alice, "10.0.1.2/32, 192.168.30.0/24")
	if err != nil || !found {
		t.Fatalf("found = %v, err = %v", found, err)
	}

	conf := read(t, path)
	if !strings.Contains(conf, "AllowedIPs = 10.0.1.2/32, 192.168.30.0/24") {
		t.Fatalf("alice's AllowedIPs not updated:\n%s", conf)
	}
	if !strings.Contains(conf, "AllowedIPs = 10.0.1.2/32\n") {
		t.Fatalf("bob's AllowedIPs was touched:\n%s", conf)
	}
	if !strings.Contains(conf, "APUB") || !strings.Contains(conf, "APSK") {
		t.Fatalf("alice's other fields lost:\n%s", conf)
	}
}

func TestRewritePeerAllowedIPsOfMissingClientIsNoOp(t *testing.T) {
	path := newServerConf(t)
	before := read(t, path)
	found, err := RewritePeerAllowedIPs(path, &api.Client{ID: "ghost"}, "10.0.0.9/32")
	if err != nil || found {
		t.Errorf("found = %v, err = %v", found, err)
	}
	if read(t, path) != before {
		t.Error("file rewritten for a client that isn't there")
	}
}

// A routing change made while a client is suspended must survive
// reactivation - not get overwritten by the stale parked copy.
func TestRewriteParkedPeerAllowedIPsSurvivesRestore(t *testing.T) {
	path := newServerConf(t)
	alice := &api.Client{ID: "aaa", Name: "alice", ClientPublicKey: "APUB", PresharedKey: "APSK"}
	if err := AppendPeer(path, alice, "10.0.1.2/32"); err != nil {
		t.Fatal(err)
	}
	if err := ParkPeer(path, alice); err != nil {
		t.Fatal(err)
	}

	found, err := RewriteParkedPeerAllowedIPs(path, alice, "10.0.1.2/32, 192.168.30.0/24")
	if err != nil || !found {
		t.Fatalf("found = %v, err = %v", found, err)
	}

	if err := RestorePeer(path, alice); err != nil {
		t.Fatal(err)
	}
	if conf := read(t, path); !strings.Contains(conf, "AllowedIPs = 10.0.1.2/32, 192.168.30.0/24") {
		t.Fatalf("restored peer lost the routing change:\n%s", conf)
	}
}

func TestRewriteParkedPeerAllowedIPsOfUnparkedClientIsNoOp(t *testing.T) {
	path := newServerConf(t)
	found, err := RewriteParkedPeerAllowedIPs(path, &api.Client{ID: "ghost"}, "10.0.0.9/32")
	if err != nil || found {
		t.Errorf("found = %v, err = %v", found, err)
	}
}
