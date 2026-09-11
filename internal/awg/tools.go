package awg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Tools wraps the host commands behind names that say what they are for.
// Nothing here knows about servers or clients; it takes interface names and
// subnets and reports what the host said.
type Tools struct {
	run Runner

	// ScriptsDir holds setup_iptables.sh and cleanup_iptables.sh. A
	// deployment without them - a development tree - simply skips the
	// firewall step.
	ScriptsDir string

	// BinDir is where awg and awg-quick are installed.
	BinDir string
}

// New returns Tools issuing commands through run, with the container's
// default locations.
func New(run Runner) *Tools {
	return &Tools{run: run, ScriptsDir: "/app/scripts", BinDir: "/usr/bin"}
}

// Available reports whether both amneziawg-tools binaries are installed.
func (t *Tools) Available() bool {
	_, awgErr := os.Stat(filepath.Join(t.BinDir, "awg"))
	_, quickErr := os.Stat(filepath.Join(t.BinDir, "awg-quick"))
	return awgErr == nil && quickErr == nil
}

// QuickUp brings an interface up from its .conf.
func (t *Tools) QuickUp(iface string) error {
	if _, err := t.run.Run(fmt.Sprintf("%s up %s", filepath.Join(t.BinDir, "awg-quick"), iface)); err != nil {
		return fmt.Errorf("awg-quick up %s: %w", iface, err)
	}
	return nil
}

// QuickDown tears an interface down.
func (t *Tools) QuickDown(iface string) error {
	if _, err := t.run.Run(fmt.Sprintf("%s down %s", filepath.Join(t.BinDir, "awg-quick"), iface)); err != nil {
		return fmt.Errorf("awg-quick down %s: %w", iface, err)
	}
	return nil
}

// SyncConf pushes the interface's .conf onto the running interface without
// taking it down, so existing peers keep their sessions.
func (t *Tools) SyncConf(iface string) error {
	cmd := fmt.Sprintf("bash -c 'awg syncconf %s <(awg-quick strip %s)'", iface, iface)
	_, err := t.run.Run(cmd)
	return err
}

// InterfaceUp reports whether the kernel has the interface. It shells out,
// so callers must not hold a lock other requests wait on while asking.
func (t *Tools) InterfaceUp(iface string) bool {
	if iface == "" {
		return false
	}
	result, err := t.run.Run(fmt.Sprintf("ip link show %s", iface))
	if err != nil {
		return false
	}
	return strings.Contains(result, "state UNKNOWN") || strings.Contains(result, iface)
}

// SetupIPTables adds the forwarding and NAT rules for an interface, if the
// script for it is deployed.
func (t *Tools) SetupIPTables(iface, subnet string) {
	t.runScript("setup_iptables.sh", iface, subnet)
}

// CleanupIPTables removes what SetupIPTables added.
func (t *Tools) CleanupIPTables(iface, subnet string) {
	t.runScript("cleanup_iptables.sh", iface, subnet)
}

func (t *Tools) runScript(name, iface, subnet string) {
	script := filepath.Join(t.ScriptsDir, name)
	if _, err := os.Stat(script); err != nil {
		return
	}
	t.run.Run(fmt.Sprintf("%s %s %s", script, iface, subnet)) //nolint:errcheck
}

// CheckIPTables looks for the rules SetupIPTables should have left, keyed by
// the command that looked. The values are "Found" or "Not found".
func (t *Tools) CheckIPTables(iface, subnet string) map[string]string {
	checks := []string{
		fmt.Sprintf("iptables -L INPUT -n | grep %s", iface),
		fmt.Sprintf("iptables -L FORWARD -n | grep %s", iface),
		fmt.Sprintf("iptables -t nat -L POSTROUTING -n | grep %s", subnet),
	}
	results := map[string]string{}
	for _, cmd := range checks {
		out, err := t.run.Run(cmd)
		if err == nil && out != "" {
			results[cmd] = "Found"
		} else {
			results[cmd] = "Not found"
		}
	}
	return results
}

// RouteSourceIP is the address the host would use to reach the internet,
// from its routing table. It is the fallback when no external service could
// say what the public address is.
func (t *Tools) RouteSourceIP() (string, error) {
	return t.run.Run("ip route get 1 | awk '{print $7}' | head -1")
}

// PeerStats is what `awg show` reports about one peer.
type PeerStats struct {
	Received, Sent, LastHandshake, Endpoint string
}

// ShowPeers parses `awg show <iface>` into per-peer counters keyed by the
// peer's public key. A down interface yields nil.
func (t *Tools) ShowPeers(iface string) map[string]PeerStats {
	output, err := t.run.Run(fmt.Sprintf("%s show %s", filepath.Join(t.BinDir, "awg"), iface))
	if err != nil || output == "" {
		return nil
	}
	return ParseShow(output)
}

// ParseShow reads the output of `awg show`. It is separate from ShowPeers so
// the parser can be tested on captured output.
func ParseShow(output string) map[string]PeerStats {
	peers := map[string]PeerStats{}
	current := ""
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "peer:"):
			current = strings.TrimSpace(strings.TrimPrefix(line, "peer:"))
			peers[current] = PeerStats{Received: "0 B", Sent: "0 B", LastHandshake: "Never"}
		case current == "":
			continue
		case strings.HasPrefix(line, "transfer:"):
			parts := strings.SplitN(strings.TrimPrefix(line, "transfer:"), ",", 2)
			if len(parts) == 2 {
				// "1.20 MiB received, 3.40 MiB sent": the units stay, the
				// direction is already implied by the field.
				p := peers[current]
				p.Received = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[0]), " received"))
				p.Sent = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), " sent"))
				peers[current] = p
			}
		case strings.HasPrefix(line, "endpoint:"):
			p := peers[current]
			p.Endpoint = strings.TrimSpace(strings.TrimPrefix(line, "endpoint:"))
			peers[current] = p
		case strings.HasPrefix(line, "latest handshake:"):
			p := peers[current]
			p.LastHandshake = strings.TrimSpace(strings.TrimPrefix(line, "latest handshake:"))
			peers[current] = p
		}
	}
	return peers
}

// Compiled once: the traffic snapshot calls InterfaceCounters for every
// server on every poll, and compiling these two on each call costs more than
// the matching itself.
var (
	rxRe = regexp.MustCompile(`RX bytes:\d+\s+\(([^)]+)\)`)
	txRe = regexp.MustCompile(`TX bytes:\d+\s+\(([^)]+)\)`)
)

// InterfaceCounters reads the human-readable RX/TX totals of an interface
// from ifconfig. ok is false for an interface that is down.
func (t *Tools) InterfaceCounters(iface string) (rx, tx string, ok bool) {
	output, err := t.run.Run(fmt.Sprintf("ifconfig %s", iface))
	if err != nil || output == "" {
		return "", "", false
	}
	rx, tx = "0 B", "0 B"
	if m := rxRe.FindStringSubmatch(output); len(m) > 1 {
		rx = m[1]
	}
	if m := txRe.FindStringSubmatch(output); len(m) > 1 {
		tx = m[1]
	}
	return rx, tx, true
}
