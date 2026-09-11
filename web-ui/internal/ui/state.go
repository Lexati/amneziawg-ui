package ui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"

	"amneziawg-web-ui/web-ui/api"
	"amneziawg-web-ui/web-ui/internal/ui/newserver"
	"amneziawg-web-ui/web-ui/internal/ui/style"
)

const (
	// The Socket.IO feed drives the counters; these loops only cover the
	// case where the socket is down, and keep the list honest if an event
	// is ever missed.
	trafficInterval = 10 * time.Second
	serversInterval = 60 * time.Second
)

// state is what the page has last heard from the backend, behind one lock:
// the server list the cards are rendered from, and the defaults the create
// form and the client editor start from. It is the create form's
// newserver.State.
type state struct {
	mu          sync.Mutex
	servers     []api.Server
	serversHash string
	defaultI    api.ISettings

	// webUIPort is the panel's own listener, reported by /api/system/status.
	// The create form treats it as taken like any server's port.
	webUIPort int

	// serverDefaultMTU is DEFAULT_MTU as the backend was started with it,
	// also from /api/system/status. The create form starts from it rather
	// than from a number of its own, so the operator's setting is what a new
	// server actually gets.
	serverDefaultMTU int

	socketLive atomic.Bool
}

func newState() *state {
	return &state{defaultI: api.ISettings{}}
}

// ServerMTU is the MTU a new server starts from: what the backend was
// configured with, or the built-in default while the status call is still in
// flight or reported something outside the range servers are allowed.
func (s *state) ServerMTU() int {
	s.mu.Lock()
	mtu := s.serverDefaultMTU
	s.mu.Unlock()

	if mtu < api.MinMTU || mtu > api.MaxMTU {
		return newserver.DefaultMTU
	}
	return mtu
}

// TakenPorts maps every port that is already spoken for to what holds it: an
// existing server, or the panel's own listener. Two interfaces cannot share a
// port, and the clash would otherwise only show up when the second one fails
// to come up.
func (s *state) TakenPorts() map[int]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	taken := make(map[int]string, len(s.servers)+1)
	if s.webUIPort > 0 {
		taken[s.webUIPort] = "the web UI"
	}
	for _, server := range s.servers {
		taken[server.Port] = fmt.Sprintf("server %q", server.Name)
	}
	return taken
}

// TakenSubnets is the set of subnets existing servers occupy.
func (s *state) TakenSubnets() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	taken := make(map[string]bool, len(s.servers))
	for _, server := range s.servers {
		taken[strings.TrimSpace(server.Subnet)] = true
	}
	return taken
}

// defaultISettings is the I1-I5 set the backend gives a new client, as last
// reported.
func (s *state) defaultISettings() api.ISettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.defaultI
}

// ── Data loading ─────────────────────────────────────────────────────────────

// Start begins loading once the toolkit is running, so the first fyne.Do
// calls always find a live event loop.
func (u *UI) Start() {
	go u.loadSystemStatus()
	go u.loadDefaultISettings()
	go u.reloadServers()
	go u.connectSocket()

	// Fallbacks for when the Socket.IO feed is not up: poll the same data
	// over REST, but stay quiet while events are flowing.
	go func() {
		for {
			time.Sleep(trafficInterval)
			if u.socketLive.Load() {
				continue
			}
			u.refreshTraffic()
		}
	}()
	go func() {
		for {
			time.Sleep(serversInterval)
			u.reloadServers()
		}
	}()
}

func (u *UI) loadSystemStatus() {
	status, err := u.env.Backend.SystemStatus()
	if err != nil {
		u.unreachable()
		return
	}
	if !u.socketLive.Load() {
		u.setTransport("polling", style.Warning)
	}

	port, err := strconv.Atoi(strings.TrimSpace(status.Environment.WebUIPort))
	u.mu.Lock()
	if err == nil {
		u.webUIPort = port
	}
	u.serverDefaultMTU = status.Environment.DefaultMTU
	u.mu.Unlock()

	u.setSummary(fmt.Sprintf("%d/%d servers running · %d clients",
		status.ActiveServers, status.TotalServers, status.TotalClients))
	u.setPublicIP(status.PublicIP)
}

func (u *UI) loadDefaultISettings() {
	settings, err := u.env.Backend.DefaultISettings()
	if err != nil {
		return
	}
	u.mu.Lock()
	u.defaultI = settings
	u.mu.Unlock()
}

func (u *UI) refreshPublicIP() {
	go func() {
		ip, err := u.env.Backend.RefreshIP()
		if err != nil {
			u.feedback.Fail(err)
			return
		}
		u.setPublicIP(ip)
		u.feedback.OK("Public IP refreshed")
		u.reloadServers()
	}()
}

// reloadServers pulls the server list and rebuilds the cards when something
// actually changed. Every action that alters structure (create, delete,
// start/stop, client changes) calls it, through env.Env.Reload.
func (u *UI) reloadServers() {
	servers, err := u.env.Backend.Servers()
	if err != nil {
		u.unreachable()
		return
	}
	u.loadSystemStatus()

	hash := fingerprint(servers)

	u.mu.Lock()
	unchanged := hash == u.serversHash
	u.servers = servers
	u.serversHash = hash
	u.mu.Unlock()

	if unchanged {
		return
	}

	fyne.Do(func() {
		u.list.Render(servers)
		u.clampScroll()
	})

	if !u.socketLive.Load() {
		u.refreshTraffic()
	}
}

// refreshTraffic fetches the counters over REST. Only used while the
// Socket.IO feed is down.
func (u *UI) refreshTraffic() {
	iface, err := u.env.Backend.InterfaceTraffic()
	if err != nil {
		u.unreachable()
		return
	}

	u.mu.Lock()
	servers := u.servers
	u.mu.Unlock()

	peers := map[string]map[string]api.ClientTraffic{}
	for _, srv := range servers {
		peers[srv.ID] = u.env.Backend.PeerTraffic(srv.ID)
	}

	u.applyTraffic(iface, peers)
}

// applyTraffic hands a snapshot to the list on the UI goroutine.
func (u *UI) applyTraffic(iface map[string]api.InterfaceTraffic, peers map[string]map[string]api.ClientTraffic) {
	fyne.Do(func() { u.list.ApplyTraffic(iface, peers) })
}

// fingerprint is a cheap "did anything change" marker for the server list.
func fingerprint(servers []api.Server) string {
	data, err := json.Marshal(servers)
	if err != nil {
		return ""
	}
	return string(data)
}
