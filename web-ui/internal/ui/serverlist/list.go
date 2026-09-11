// Package serverlist is the body of the page below the create form: one card
// per server, or a hint while there are none.
package serverlist

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"amneziawg-web-ui/web-ui/api"
	"amneziawg-web-ui/web-ui/internal/ui/env"
	"amneziawg-web-ui/web-ui/internal/ui/serverlist/serverentry"
)

// List owns the cards on screen, keyed by server ID so a traffic snapshot can
// find the card it belongs to.
type List struct {
	env *env.Env

	box   *fyne.Container
	empty *widget.Label
	cards map[string]*serverentry.Card

	object fyne.CanvasObject
}

// New builds an empty list; Render fills it.
func New(e *env.Env) *List {
	l := &List{env: e, box: container.NewVBox(), cards: map[string]*serverentry.Card{}}

	l.empty = widget.NewLabel(lang.L("No servers created yet. Create your first server above."))
	l.empty.Alignment = fyne.TextAlignCenter

	l.object = container.NewVBox(l.box, l.empty)
	return l
}

// CanvasObject is the list as the page places it.
func (l *List) CanvasObject() fyne.CanvasObject {
	return l.object
}

// Render rebuilds every card from a fresh server list. Must run on the UI
// goroutine.
func (l *List) Render(servers []api.Server) {
	l.cards = map[string]*serverentry.Card{}
	l.box.Objects = nil

	for _, srv := range servers {
		card := serverentry.New(l.env, srv)
		l.cards[srv.ID] = card
		l.box.Add(card.CanvasObject())
	}

	if len(servers) > 0 {
		l.empty.Hide()
	} else {
		l.empty.Show()
	}
	l.box.Refresh()
}

// ApplyTraffic pushes counters into the existing labels, so the page updates
// without rebuilding (and without losing scroll position or focus). Must run
// on the UI goroutine.
func (l *List) ApplyTraffic(iface map[string]api.InterfaceTraffic, peers map[string]map[string]api.ClientTraffic) {
	for id, card := range l.cards {
		card.ApplyInterfaceTraffic(iface[id])
		card.ApplyPeerTraffic(peers[id])
	}
}
