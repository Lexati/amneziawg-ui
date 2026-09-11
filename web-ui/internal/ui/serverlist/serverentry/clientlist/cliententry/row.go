// Package cliententry is one peer line inside a server card: the name, the
// address, the state badges, the live counters and the row of actions.
package cliententry

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"amneziawg-web-ui/web-ui/api"
	"amneziawg-web-ui/web-ui/internal/ui/browser"
	"amneziawg-web-ui/web-ui/internal/ui/dialogs"
	"amneziawg-web-ui/web-ui/internal/ui/env"
	"amneziawg-web-ui/web-ui/internal/ui/serverlist/serverentry/clientlist/cliententry/clientdialogs"
	"amneziawg-web-ui/web-ui/internal/ui/style"
	"amneziawg-web-ui/web-ui/internal/ui/widgets"
)

// Row is one peer line. It keeps the labels the traffic feed updates in
// place, so a counter tick never rebuilds the row.
type Row struct {
	env    *env.Env
	server api.Server
	client api.Client

	traffic   *canvas.Text
	handshake *canvas.Text
	endpoint  *canvas.Text

	object fyne.CanvasObject
}

// New renders the row for client, a peer of server.
func New(e *env.Env, server api.Server, client api.Client) *Row {
	r := &Row{
		env:       e,
		server:    server,
		client:    client,
		traffic:   widgets.SmallText("RX — · TX —", style.Muted),
		handshake: widgets.SmallText("handshake: —", style.Muted),
		endpoint:  widgets.SmallText("endpoint: —", style.Muted),
	}

	name := canvas.NewText(client.Name, style.Text)
	name.TextSize = 14
	name.TextStyle = fyne.TextStyle{Bold: true}

	address := widgets.SmallText(client.ClientIP, style.Primary)
	address.TextStyle = fyne.TextStyle{Monospace: true}

	labels := container.NewHBox(name, container.NewCenter(address))
	if client.ApplyISettings {
		labels.Add(container.NewCenter(widgets.Badge("I1-5", style.Accent)))
	}
	if client.Status == "suspended" {
		labels.Add(container.NewCenter(widgets.Badge("SUSPENDED", style.Warning)))
	} else {
		labels.Add(container.NewCenter(widgets.Badge("ACTIVE", style.Success)))
	}
	if client.SuspendAt != nil {
		when := time.Unix(int64(*client.SuspendAt), 0).Local().Format(clientdialogs.SuspendLayout)
		labels.Add(container.NewCenter(widgets.Badge("auto-suspend "+when, style.Error)))
	}

	counters := container.NewHBox(
		r.traffic, widgets.SmallText("·", style.Border),
		r.handshake, widgets.SmallText("·", style.Border),
		r.endpoint,
	)

	edit := widgets.NewButton("Edit", theme.DocumentCreateIcon(), func() {
		clientdialogs.ShowEditor(e, server, &client)
	})
	qr := widgets.NewButton("QR / config", theme.VisibilityIcon(), func() {
		clientdialogs.ShowConfig(e, server, client)
	})
	download := widgets.NewButton("", theme.DownloadIcon(), func() {
		browser.OpenURL(e.Backend.ClientConfigURL(server.ID, client.ID))
	})

	var toggle *widgets.Button
	if client.Status == "suspended" {
		toggle = widgets.NewButton("Activate", theme.MediaPlayIcon(), func() { r.setSuspended(false) })
		toggle.Importance = widget.SuccessImportance
	} else {
		toggle = widgets.NewButton("Suspend", theme.MediaPauseIcon(), func() { r.setSuspended(true) })
		toggle.Importance = widget.WarningImportance
	}

	remove := widgets.NewButton("", theme.DeleteIcon(), r.confirmDelete)
	remove.Importance = widget.DangerImportance

	actions := container.NewHBox(edit, qr, download, toggle, remove)

	bg := canvas.NewRectangle(style.SurfaceAlt)
	bg.CornerRadius = 8

	body := container.NewBorder(nil, nil, nil, container.NewCenter(actions),
		container.NewVBox(labels, counters))

	r.object = container.NewStack(bg, container.NewPadded(body))
	return r
}

// CanvasObject is the row as the list places it.
func (r *Row) CanvasObject() fyne.CanvasObject {
	return r.object
}

// Apply pushes one traffic snapshot into the labels. Must run on the UI
// goroutine.
func (r *Row) Apply(data api.ClientTraffic) {
	r.traffic.Text = fmt.Sprintf("RX %s · TX %s", data.Received, data.Sent)
	r.traffic.Refresh()

	r.handshake.Text = "handshake: " + data.LastHandshake
	r.handshake.Refresh()

	endpoint := data.Endpoint
	if endpoint == "" {
		endpoint = "endpoint: —"
	} else {
		endpoint = "endpoint: " + endpoint
	}
	r.endpoint.Text = endpoint
	r.endpoint.Refresh()
}

// ── Actions ──────────────────────────────────────────────────────────────────

func (r *Row) setSuspended(suspend bool) {
	e, client := r.env, r.client

	question := fmt.Sprintf("Activate %q again?", client.Name)
	if suspend {
		question = fmt.Sprintf("Suspend %q? The client loses its connection until reactivated.", client.Name)
	}

	dialogs.Confirm(e.Win, "Change client state", question, func() {
		go func() {
			var err error
			if suspend {
				err = e.Backend.SuspendClient(r.server.ID, client.ID)
			} else {
				err = e.Backend.ActivateClient(r.server.ID, client.ID)
			}
			if err != nil {
				e.Notify.Fail(err)
				return
			}
			if suspend {
				e.Notify.OK("Client %q suspended", client.Name)
			} else {
				e.Notify.OK("Client %q activated", client.Name)
			}
			e.Reload()
		}()
	})
}

func (r *Row) confirmDelete() {
	e, client := r.env, r.client

	dialogs.Confirm(e.Win, "Delete client", fmt.Sprintf("Delete %q?", client.Name), func() {
		go func() {
			if err := e.Backend.DeleteClient(r.server.ID, client.ID); err != nil {
				e.Notify.Fail(err)
				return
			}
			e.Notify.OK("Client %q deleted", client.Name)
			e.Reload()
		}()
	})
}
