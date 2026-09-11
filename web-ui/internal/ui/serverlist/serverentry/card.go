// Package serverentry is one server's panel in the list: its title line and
// interface counters, the start/stop/add/config actions, and its clients.
package serverentry

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"amneziawg-web-ui/web-ui/api"
	"amneziawg-web-ui/web-ui/internal/ui/dialogs"
	"amneziawg-web-ui/web-ui/internal/ui/env"
	"amneziawg-web-ui/web-ui/internal/ui/serverlist/serverentry/clientlist"
	"amneziawg-web-ui/web-ui/internal/ui/serverlist/serverentry/clientlist/cliententry/clientdialogs"
	"amneziawg-web-ui/web-ui/internal/ui/serverlist/serverentry/serverdialogs"
	"amneziawg-web-ui/web-ui/internal/ui/style"
	"amneziawg-web-ui/web-ui/internal/ui/widgets"
)

// Card is one server's panel. It keeps the interface counter label the
// traffic feed updates in place, and the client rows below it.
type Card struct {
	env    *env.Env
	server api.Server

	ifaceText *canvas.Text
	clients   *clientlist.List

	object fyne.CanvasObject
}

// New renders the card for server as the list last fetched it.
func New(e *env.Env, srv api.Server) *Card {
	c := &Card{env: e, server: srv}

	title := canvas.NewText(srv.Name, style.Text)
	title.TextSize = 17
	title.TextStyle = fyne.TextStyle{Bold: true}

	running := srv.Status == "running"
	statusColor := style.Error
	if running {
		statusColor = style.Success
	}

	meta := []string{
		lang.L("ID {{.ID}}", map[string]any{"ID": srv.ID}),
		lang.L("Port {{.Port}}", map[string]any{"Port": srv.Port}),
		lang.L("Subnet {{.Subnet}}", map[string]any{"Subnet": srv.Subnet}),
		lang.L("MTU {{.MTU}}", map[string]any{"MTU": srv.MTU}),
	}
	if srv.ObfuscationEnabled {
		meta = append(meta, lang.L("Obfuscated (AWG 3.1)"))
	}

	c.ifaceText = widgets.SmallText(interfaceTraffic("—", "—"), style.Muted)

	remove := widgets.NewButton("", theme.DeleteIcon(), c.confirmDelete)
	remove.Importance = widget.DangerImportance

	head := container.NewBorder(nil, nil,
		container.NewVBox(
			title,
			widgets.SmallText(strings.Join(meta, "  ·  "), style.Muted),
			c.ifaceText,
		),
		container.NewHBox(
			container.NewCenter(widgets.Badge(strings.ToUpper(lang.L(srv.Status)), statusColor)),
			container.NewCenter(remove),
		),
	)

	start := widgets.NewButton(lang.L("Start"), theme.MediaPlayIcon(), func() { c.setRunning(true) })
	start.Importance = widget.SuccessImportance
	stop := widgets.NewButton(lang.L("Stop"), theme.MediaStopIcon(), func() { c.setRunning(false) })
	stop.Importance = widget.DangerImportance
	add := widgets.NewButton(lang.L("Add client"), theme.ContentAddIcon(), func() {
		clientdialogs.ShowEditor(e, srv, nil)
	})
	add.Importance = widget.HighImportance
	config := widgets.NewButton(lang.L("Show config"), theme.DocumentIcon(), func() {
		serverdialogs.ShowConfig(e, srv.ID)
	})

	if running {
		start.Disable()
	} else {
		stop.Disable()
	}

	actions := container.NewHBox(start, stop, add, config)

	c.clients = clientlist.New(e, srv)

	content := container.NewVBox(
		head,
		actions,
		widgets.Separator(),
		c.clients.CanvasObject(),
	)

	c.object = container.NewPadded(widgets.Card(content))
	return c
}

// CanvasObject is the card as the list places it.
func (c *Card) CanvasObject() fyne.CanvasObject {
	return c.object
}

// ApplyInterfaceTraffic updates the RX/TX line of the interface. Must run on
// the UI goroutine.
func (c *Card) ApplyInterfaceTraffic(traffic api.InterfaceTraffic) {
	rx, tx := "—", "—"
	if traffic != nil {
		rx, tx = traffic["rx"], traffic["tx"]
	}
	c.ifaceText.Text = interfaceTraffic(rx, tx)
	c.ifaceText.Refresh()
}

func interfaceTraffic(rx, tx string) string {
	return lang.L("Interface: RX {{.RX}} · TX {{.TX}}", map[string]any{"RX": rx, "TX": tx})
}

// ApplyPeerTraffic hands the per-client snapshot down to the rows. Must run
// on the UI goroutine.
func (c *Card) ApplyPeerTraffic(traffic map[string]api.ClientTraffic) {
	c.clients.ApplyTraffic(traffic)
}

// ── Actions ──────────────────────────────────────────────────────────────────

func (c *Card) setRunning(start bool) {
	e, srv := c.env, c.server

	go func() {
		var err error
		if start {
			err = e.Backend.StartServer(srv.ID)
		} else {
			err = e.Backend.StopServer(srv.ID)
		}
		if err != nil {
			e.Notify.Fail(err)
			return
		}
		name := map[string]any{"Name": srv.Name}
		if start {
			e.Notify.OK(lang.L("Server \"{{.Name}}\" started", name))
		} else {
			e.Notify.OK(lang.L("Server \"{{.Name}}\" stopped", name))
		}
		e.Reload()
	}()
}

func (c *Card) confirmDelete() {
	e, srv := c.env, c.server

	name := map[string]any{"Name": srv.Name}
	dialogs.Confirm(e.Win, lang.L("Delete server"),
		lang.L("Delete \"{{.Name}}\" and all of its clients?", name),
		func() {
			go func() {
				if err := e.Backend.DeleteServer(srv.ID); err != nil {
					e.Notify.Fail(err)
					return
				}
				e.Notify.OK(lang.L("Server \"{{.Name}}\" deleted", name))
				e.Reload()
			}()
		})
}
