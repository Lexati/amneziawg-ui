package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"amneziawg-web-ui/web-ui/api"
)

const suspendLayout = "2006-01-02 15:04"

// buildClientRow renders one peer line and registers its mutable labels so
// the traffic feed can update them in place.
func (c *serverCard) buildClientRow(client api.Client) fyne.CanvasObject {
	row := &clientRow{
		traffic:   smallText("RX — · TX —", colMuted),
		handshake: smallText("handshake: —", colMuted),
		endpoint:  smallText("endpoint: —", colMuted),
	}
	c.rows[client.ID] = row

	name := canvas.NewText(client.Name, colText)
	name.TextSize = 14
	name.TextStyle = fyne.TextStyle{Bold: true}

	address := smallText(client.ClientIP, colPrimary)
	address.TextStyle = fyne.TextStyle{Monospace: true}

	labels := container.NewHBox(name, container.NewCenter(address))
	if client.ApplyISettings {
		labels.Add(container.NewCenter(badge("I1-5", colAccent)))
	}
	if client.Status == "suspended" {
		labels.Add(container.NewCenter(badge("SUSPENDED", colWarning)))
	} else {
		labels.Add(container.NewCenter(badge("ACTIVE", colSuccess)))
	}
	if client.SuspendAt != nil {
		when := time.Unix(int64(*client.SuspendAt), 0).Local().Format(suspendLayout)
		labels.Add(container.NewCenter(badge("auto-suspend "+when, colError)))
	}

	counters := container.NewHBox(
		row.traffic, smallText("·", colBorder),
		row.handshake, smallText("·", colBorder),
		row.endpoint,
	)

	edit := button("Edit", theme.DocumentCreateIcon(), func() {
		c.ui.showClientDialog(c.server, &client)
	})
	qr := button("QR / config", theme.VisibilityIcon(), func() {
		c.ui.showClientConfig(c.server, client)
	})
	download := button("", theme.DownloadIcon(), func() {
		openURL(c.ui.backend.ClientConfigURL(c.server.ID, client.ID))
	})

	var toggle *pointerButton
	if client.Status == "suspended" {
		toggle = button("Activate", theme.MediaPlayIcon(), func() {
			c.ui.setClientSuspended(c.server.ID, client, false)
		})
		toggle.Importance = widget.SuccessImportance
	} else {
		toggle = button("Suspend", theme.MediaPauseIcon(), func() {
			c.ui.setClientSuspended(c.server.ID, client, true)
		})
		toggle.Importance = widget.WarningImportance
	}

	remove := button("", theme.DeleteIcon(), func() {
		c.ui.confirmDeleteClient(c.server.ID, client)
	})
	remove.Importance = widget.DangerImportance

	actions := container.NewHBox(edit, qr, download, toggle, remove)

	bg := canvas.NewRectangle(colSurfaceAlt)
	bg.CornerRadius = 8

	body := container.NewBorder(nil, nil, nil, container.NewCenter(actions),
		container.NewVBox(labels, counters))

	return container.NewStack(bg, container.NewPadded(body))
}

// ── Client actions ───────────────────────────────────────────────────────────

func (u *UI) setClientSuspended(serverID string, client api.Client, suspend bool) {
	question := fmt.Sprintf("Activate %q again?", client.Name)
	if suspend {
		question = fmt.Sprintf("Suspend %q? The client loses its connection until reactivated.", client.Name)
	}

	u.showConfirm("Change client state", question, func() {
		go func() {
			var err error
			if suspend {
				err = u.backend.SuspendClient(serverID, client.ID)
			} else {
				err = u.backend.ActivateClient(serverID, client.ID)
			}
			if err != nil {
				u.fail(err)
				return
			}
			if suspend {
				u.ok("Client %q suspended", client.Name)
			} else {
				u.ok("Client %q activated", client.Name)
			}
			u.reloadServers()
		}()
	})
}

func (u *UI) confirmDeleteClient(serverID string, client api.Client) {
	u.showConfirm("Delete client", fmt.Sprintf("Delete %q?", client.Name), func() {
		go func() {
			if err := u.backend.DeleteClient(serverID, client.ID); err != nil {
				u.fail(err)
				return
			}
			u.ok("Client %q deleted", client.Name)
			u.reloadServers()
		}()
	})
}

// showClientDialog opens the add/edit form. A nil client means "add new".
func (u *UI) showClientDialog(server api.Server, client *api.Client) {
	editing := client != nil

	name := newEntry()
	allowedIPs := entryWithText("0.0.0.0/0, ::/0")
	suspendAt := entryWithPlaceholder(suspendLayout)

	applyI := widget.NewCheck("Apply I-settings (custom signature packets I1-I5)", nil)
	iEntries := make([]*widget.Entry, 5)
	for i := range iEntries {
		iEntries[i] = widget.NewEntry()
	}

	u.mu.Lock()
	defaults := u.defaultI
	u.mu.Unlock()
	for i, entry := range iEntries {
		key := fmt.Sprintf("i%d", i+1)
		if value := defaults[key]; value != "" {
			entry.SetPlaceHolder("server default: " + truncate(value, 40))
		} else {
			entry.SetPlaceHolder("leave empty to skip")
		}
	}

	if editing {
		name.SetText(client.Name)
		name.Disable()
		if client.AllowedIPs != "" {
			allowedIPs.SetText(client.AllowedIPs)
		}
		applyI.SetChecked(client.ApplyISettings)
		for i, entry := range iEntries {
			entry.SetText(client.ISettings[fmt.Sprintf("i%d", i+1)])
		}
		if client.SuspendAt != nil {
			suspendAt.SetText(time.Unix(int64(*client.SuspendAt), 0).Local().Format(suspendLayout))
		}
	} else {
		name.SetPlaceHolder("New Client")
	}

	iBox := container.NewVBox()
	for i, entry := range iEntries {
		iBox.Add(labeled(fmt.Sprintf("I%d", i+1), entry))
	}
	iNote := widget.NewLabel("Each value describes one packet sent before the handshake, as a sequence of tags: " +
		"<b 0x...> static bytes, <t> a 4-byte timestamp, <r N> random bytes, <rc N> random letters, <rd N> random digits " +
		"(N up to 1000). Example: <b 0xd100000001><rc 8><t><r 50>. These are client-only parameters, they are never " +
		"written into the server config; empty values are skipped, and an empty I1 turns the whole set off. Keep each " +
		"packet under about 1472 bytes or it is fragmented on the way out, which is the signature this is meant to avoid. " +
		"A config that grows past the QR code limit can still be downloaded as a file.")
	iNote.Wrapping = fyne.TextWrapWord
	iNote.TextStyle = fyne.TextStyle{Italic: true}
	iBox.Add(iNote)
	if !applyI.Checked {
		iBox.Hide()
	}

	applyI.OnChanged = func(on bool) {
		if on {
			iBox.Show()
		} else {
			iBox.Hide()
		}
	}

	items := []*widget.FormItem{
		{Text: "Client name", Widget: name},
		{Text: "Allowed IPs", Widget: allowedIPs,
			HintText: "comma-separated ranges routed through the VPN; default 0.0.0.0/0, ::/0"},
	}

	if editing {
		clear := button("", theme.CancelIcon(), func() { suspendAt.SetText("") })
		items = append(items, &widget.FormItem{
			Text:     "Auto-suspend at",
			Widget:   container.NewBorder(nil, nil, nil, clear, suspendAt),
			HintText: "local time, format " + suspendLayout + "; empty disables auto-suspension",
		})
		created := time.Unix(int64(client.CreatedAt), 0).Local().Format("2006-01-02 15:04:05")
		items = append(items, &widget.FormItem{Text: "Created", Widget: widget.NewLabel(created)})
	}

	content := container.NewVBox(widget.NewForm(items...), separator(), applyI, iBox)

	title := "Add client to " + server.Name
	confirm := "Add client"
	if editing {
		title = "Edit client " + client.Name
		confirm = "Save"
	}

	u.showFormDialog(title, confirm, "Cancel", u.scrolled(content), u.dialogSize(760, 620), func() bool {
		settings := api.ISettings{}
		var problems []string
		for i, entry := range iEntries {
			value := strings.TrimSpace(entry.Text)
			if value == "" {
				continue
			}
			key := fmt.Sprintf("I%d", i+1)
			if problem := api.ValidateCPS(key, value); problem != "" {
				problems = append(problems, problem)
				continue
			}
			settings[strings.ToLower(key)] = value
		}
		// Only worth checking what will actually be written: with the box
		// unticked the values are kept but never reach a config.
		if applyI.Checked && len(problems) > 0 {
			u.fail(errors.New(strings.Join(problems, "\n")))
			return false
		}

		routes := strings.TrimSpace(allowedIPs.Text)
		if routes == "" {
			routes = "0.0.0.0/0, ::/0"
		}

		if editing {
			u.saveClient(server.ID, *client, routes, applyI.Checked, settings, strings.TrimSpace(suspendAt.Text))
			return true
		}
		u.addClient(server, strings.TrimSpace(name.Text), routes, applyI.Checked, settings)
		return true
	})
}

func (u *UI) addClient(server api.Server, name, allowedIPs string, applyI bool, settings api.ISettings) {
	if name == "" {
		name = "New Client"
	}

	go func() {
		err := u.backend.AddClient(server.ID, api.AddClientRequest{
			Name:           name,
			ApplyISettings: applyI,
			ISettings:      settings,
			AllowedIPs:     allowedIPs,
		})
		if err != nil {
			u.fail(err)
			return
		}
		u.ok("Client %q added", name)
		u.reloadServers()
	}()
}

// saveClient applies the three independent updates the backend exposes for an
// existing peer: routing, I-settings and the auto-suspend timestamp.
func (u *UI) saveClient(serverID string, client api.Client, allowedIPs string, applyI bool, settings api.ISettings, suspendAt string) {
	var when *time.Time
	if suspendAt != "" {
		parsed, err := time.ParseInLocation(suspendLayout, suspendAt, time.Local)
		if err != nil {
			u.fail(fmt.Errorf("auto-suspend time must look like %s", suspendLayout))
			return
		}
		when = &parsed
	}

	go func() {
		if err := u.backend.UpdateAllowedIPs(serverID, client.ID, allowedIPs); err != nil {
			u.fail(err)
			return
		}
		if err := u.backend.UpdateISettings(serverID, client.ID, applyI, settings); err != nil {
			u.fail(err)
			return
		}
		if err := u.backend.UpdateSuspendTime(serverID, client.ID, when); err != nil {
			u.fail(err)
			return
		}
		u.ok("Client %q updated", client.Name)
		u.reloadServers()
	}()
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}
