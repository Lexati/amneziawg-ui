// Package serverentry is one server's panel in the list: its title line and
// interface counters, the start/stop/add/config actions, and its clients.
package serverentry

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
		"ID " + srv.ID,
		fmt.Sprintf("Port %d", srv.Port),
		"Subnet " + srv.Subnet,
		fmt.Sprintf("MTU %d", srv.MTU),
	}
	if srv.ObfuscationEnabled {
		meta = append(meta, "Obfuscated (AWG 3.1)")
	}

	c.ifaceText = widgets.SmallText("Interface: RX — · TX —", style.Muted)

	remove := widgets.NewButton("", theme.DeleteIcon(), c.confirmDelete)
	remove.Importance = widget.DangerImportance

	head := container.NewBorder(nil, nil,
		container.NewVBox(
			title,
			widgets.SmallText(strings.Join(meta, "  ·  "), style.Muted),
			c.ifaceText,
		),
		container.NewHBox(
			container.NewCenter(widgets.Badge(strings.ToUpper(srv.Status), statusColor)),
			container.NewCenter(remove),
		),
	)

	start := widgets.NewButton("Start", theme.MediaPlayIcon(), func() { c.setRunning(true) })
	start.Importance = widget.SuccessImportance
	stop := widgets.NewButton("Stop", theme.MediaStopIcon(), func() { c.setRunning(false) })
	stop.Importance = widget.DangerImportance
	add := widgets.NewButton("Add client", theme.ContentAddIcon(), func() {
		clientdialogs.ShowEditor(e, srv, nil)
	})
	add.Importance = widget.HighImportance
	config := widgets.NewButton("Show config", theme.DocumentIcon(), func() {
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
	c.ifaceText.Text = fmt.Sprintf("Interface: RX %s · TX %s", rx, tx)
	c.ifaceText.Refresh()
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
		if start {
			e.Notify.OK("Server %q started", srv.Name)
		} else {
			e.Notify.OK("Server %q stopped", srv.Name)
		}
		e.Reload()
	}()
}

func (c *Card) confirmDelete() {
	e, srv := c.env, c.server

	dialogs.Confirm(e.Win, "Delete server",
		fmt.Sprintf("Delete %q and all of its clients?", srv.Name),
		func() {
			go func() {
				if err := e.Backend.DeleteServer(srv.ID); err != nil {
					e.Notify.Fail(err)
					return
				}
				e.Notify.OK("Server %q deleted", srv.Name)
				e.Reload()
			}()
		})
}
