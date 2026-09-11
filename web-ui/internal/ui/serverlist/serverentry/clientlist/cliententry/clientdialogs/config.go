package clientdialogs

import (
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	qrcode "github.com/skip2/go-qrcode"

	"amneziawg-web-ui/web-ui/api"
	"amneziawg-web-ui/web-ui/internal/ui/browser"
	"amneziawg-web-ui/web-ui/internal/ui/dialogs"
	"amneziawg-web-ui/web-ui/internal/ui/env"
	"amneziawg-web-ui/web-ui/internal/ui/style"
	"amneziawg-web-ui/web-ui/internal/ui/widgets"
)

// qrCapacity is the largest payload still worth rendering as a QR code; past
// that the symbol gets too dense for a phone camera, so the UI points at the
// download instead. It matches the limit the previous web UI used.
const qrCapacity = 2000

// configView is one of the representations a client config has.
type configView struct {
	label string
	text  string
	note  string
	// qr marks the views worth rendering as a QR code. The AmneziaVPN link
	// is not one of them: it is far too long to scan comfortably, and the
	// app takes it from the clipboard anyway.
	qr bool
}

// ShowConfig fetches the client's configs and opens the viewer.
func ShowConfig(e *env.Env, server api.Server, client api.Client) {
	go func() {
		configs, err := e.Backend.ClientConfigs(server.ID, client.ID)
		if err != nil {
			e.Notify.Fail(err)
			return
		}
		link := e.Backend.AmneziaLink(server.ID, client.ID)

		fyne.Do(func() { presentConfig(e, server, client, configs, link) })
	}()
}

func presentConfig(e *env.Env, server api.Server, client api.Client, configs api.ClientConfigs, link string) {
	views := []configView{}
	if link != "" {
		// Importing a plain .conf makes the official Amnezia app tag the
		// server as the legacy "amnezia-awg" container (AmneziaWG 2.0, no
		// header protection). The native link carries the container and
		// protocol_version fields that make it recognise AWG 3.x, so it is
		// the recommended - and default - view.
		views = append(views, configView{
			label: lang.L("AmneziaVPN link"),
			text:  link,
			note:  lang.L("Copy this link into the AmneziaVPN app - it is recognised as AmneziaWG 3.x"),
		})
	}
	views = append(views, configView{
		label: ".conf",
		text:  configs.CleanConfig,
		note:  lang.L("Scan with the AmneziaWG / AmneziaVPN app"),
		qr:    true,
	})

	qrImage := canvas.NewImageFromResource(nil)
	qrImage.FillMode = canvas.ImageFillContain
	qrImage.SetMinSize(fyne.NewSize(280, 280))

	qrNote := widgets.SmallText("", style.Muted)
	warning := widget.NewLabel("")
	warning.Wrapping = fyne.TextWrapWord
	warning.Importance = widget.WarningImportance
	warning.Hide()

	text, setText := widgets.MonospaceView("")
	text.SetMinRowsVisible(12)
	length := widgets.SmallText("", style.Muted)

	current := views[0]
	var qrPNG []byte

	copyButton := widgets.NewButton(lang.L("Copy"), theme.ContentCopyIcon(), func() {
		dialogs.Copy(e.Notify, current.text)
	})
	saveQR := widgets.NewButton(lang.L("Save QR image"), theme.DownloadIcon(), func() {
		if qrPNG == nil {
			return
		}
		browser.SaveBytes(safeFileName(client.Name)+"_qr.png", "image/png", qrPNG)
	})

	// The QR column disappears for views that have no QR code, so the text
	// takes the full width instead of leaving a hole. The hint stays with
	// the text, which is the part that is always on screen.
	left := container.NewVBox(
		container.NewCenter(qrImage),
		container.NewCenter(saveQR),
	)

	render := func(view configView) {
		current = view
		setText(view.text)
		length.Text = lang.N("{{.Count}} characters", len(view.text), map[string]any{"Count": len(view.text)})
		length.Refresh()

		qrPNG = nil
		qrNote.Text = view.note

		if !view.qr {
			left.Hide()
			warning.Hide()
			qrNote.Refresh()
			return
		}

		left.Show()
		qrImage.Show()
		saveQR.Show()
		png, err := encodeQR(view.text)
		switch {
		case err != nil:
			qrImage.Hide()
			saveQR.Hide()
			qrNote.Text = ""
			warning.SetText(err.Error())
			warning.Show()
		default:
			qrPNG = png
			qrImage.Resource = fyne.NewStaticResource("qr.png", png)
			qrImage.Show()
			qrImage.Refresh()
			warning.Hide()
		}
		qrNote.Refresh()
	}

	labels := make([]string, 0, len(views))
	byLabel := map[string]configView{}
	for _, view := range views {
		labels = append(labels, view.label)
		byLabel[view.label] = view
	}

	tabs := widget.NewRadioGroup(labels, func(selected string) {
		if view, ok := byLabel[selected]; ok {
			render(view)
		}
	})
	tabs.Horizontal = true
	tabs.SetSelected(views[0].label)

	download := widgets.NewButton(lang.L("Download .conf"), theme.DownloadIcon(), func() {
		browser.OpenURL(e.Backend.ClientConfigURL(server.ID, client.ID))
	})
	download.Importance = widget.HighImportance

	created := lang.L("unknown")
	if configs.CreatedAt > 0 {
		created = configs.CreatedAtReadable
	}
	suspend := lang.L("not set")
	if configs.SuspendAt != nil {
		suspend = configs.SuspendAtReadable
	}

	right := container.NewBorder(
		container.NewVBox(tabs, qrNote, warning),
		container.NewVBox(
			container.NewBorder(nil, nil, length, copyButton),
		), nil, nil, text)

	meta := container.NewHBox(
		widgets.SmallText(lang.L("Created: {{.When}}", map[string]any{"When": created}), style.Muted),
		widgets.SmallText("·", style.Border),
		widgets.SmallText(lang.L("Auto-suspend: {{.When}}", map[string]any{"When": suspend}), style.Muted),
	)

	body := container.NewBorder(meta, container.NewHBox(download), left, nil, right)

	dialogs.Show(e.Win, lang.L("Client configuration: {{.Name}}", map[string]any{"Name": client.Name}),
		lang.L("Close"), body, dialogs.Size(e.Win, 960, 720))

	render(current)
}

// encodeQR renders the payload as a PNG, refusing anything too dense to scan.
func encodeQR(text string) ([]byte, error) {
	if text == "" {
		return nil, errors.New(lang.L("this view is unavailable"))
	}
	if len(text) > qrCapacity {
		return nil, errors.New(lang.L("config is too large for a QR code: {{.Length}} characters (max {{.Max}}). "+
			"Use \"Download .conf\" instead", map[string]any{"Length": len(text), "Max": qrCapacity}))
	}

	png, err := qrcode.Encode(text, qrcode.Medium, 640)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", lang.L("could not generate QR code"), err)
	}
	return png, nil
}

func safeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "client"
	}
	return b.String()
}
