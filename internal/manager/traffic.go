package manager

import (
	"time"

	"amneziawg-web-ui/web-ui/api"
)

// TrafficSnapshot collects the interface and peer counters of every server,
// plus the host's own gauges, into the one response the page polls. Servers
// that are down contribute nothing; the page keeps their last counters as
// they were.
func (m *Manager) TrafficSnapshot() api.TrafficSnapshot {
	servers := m.copyServers()

	clientTraffic := map[string]map[string]api.ClientTraffic{}
	serverTraffic := map[string]api.InterfaceTraffic{}
	for i := range servers {
		srv := &servers[i]
		if t := m.peerTraffic(srv); len(t) > 0 {
			clientTraffic[srv.ID] = t
		}
		if c, ok := m.tools.InterfaceCounters(srv.Interface); ok {
			serverTraffic[srv.ID] = api.InterfaceTraffic{RX: c.RX, TX: c.TX, RXBytes: c.RXBytes, TXBytes: c.TXBytes}
		}
	}

	return api.TrafficSnapshot{
		Timestamp:     float64(time.Now().Unix()),
		ClientTraffic: clientTraffic,
		ServerTraffic: serverTraffic,
		System:        m.host.Metrics(),
	}
}

// peerTraffic joins what `awg show` reports with the clients the server
// owns, keyed by client ID. srv is a snapshot, so no lock is needed.
func (m *Manager) peerTraffic(srv *api.Server) map[string]api.ClientTraffic {
	peers := m.tools.ShowPeers(srv.Interface)
	if peers == nil {
		return nil
	}

	result := map[string]api.ClientTraffic{}
	for _, c := range srv.Clients {
		if p, ok := peers[c.ClientPublicKey]; ok {
			result[c.ID] = api.ClientTraffic{
				Received:      p.Received,
				Sent:          p.Sent,
				LastHandshake: p.LastHandshake,
				Endpoint:      p.Endpoint,
			}
		} else {
			result[c.ID] = api.ClientTraffic{Received: "0 B", Sent: "0 B", LastHandshake: "Never"}
		}
	}
	return result
}
