// Package dashboard is the collapsible monitoring panel at the top of the
// page: five tiles with a five-minute chart each - VPN upload and download
// summed over every server, and the host's CPU, memory and disk.
//
// The backend sends one instantaneous reading per poll and remembers
// nothing; the history the charts draw lives here, in the page, and starts
// over with every reload.
package dashboard

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"amneziawg-web-ui/web-ui/api"
	"amneziawg-web-ui/web-ui/internal/ui/style"
	"amneziawg-web-ui/web-ui/internal/ui/widgets"
)

// Window is how far back every chart reaches.
const Window = 5 * time.Minute

// Panel is the dashboard. Open by default: it is what the operator looks at
// first, and it folds away when the server list is what matters.
type Panel struct {
	// reflow tells the page its content height changed, so it can re-clamp
	// the scroll offset; see toggleOpen.
	reflow func()

	upload, download, cpu, memory, storage *tile

	// The rates are differences between consecutive polls, so the previous
	// reading has to be kept: the byte totals and when they arrived, and
	// the CPU seconds.
	last struct {
		at         time.Time
		rx, tx     uint64
		busy, idle float64
		ok         bool
	}

	toggle *widgets.Button
	body   *fyne.Container
	panel  *fyne.Container
}

// New builds the panel for a page that polls every interval; the charts hold
// Window worth of samples at that rate. reflow is called whenever the panel
// changes height.
func New(interval time.Duration, reflow func()) *Panel {
	points := int(Window / interval)
	p := &Panel{reflow: reflow}

	p.upload = newTile(theme.UploadIcon(), lang.L("Outgoing"), lang.L("Sent"), style.Success, points, 0)
	p.download = newTile(theme.DownloadIcon(), lang.L("Incoming"), lang.L("Received"), style.Primary, points, 0)
	p.cpu = newTile(theme.ComputerIcon(), lang.L("CPU"), lang.L("Cores"), style.Warning, points, 1)
	p.memory = newTile(theme.GridIcon(), lang.L("RAM"), lang.L("Used"), style.Accent, points, 1)
	p.storage = newTile(theme.StorageIcon(), lang.L("Storage"), lang.L("Used"), style.Muted, points, 1)

	grid := container.NewGridWithColumns(5,
		p.upload.object, p.download.object, p.cpu.object, p.memory.object, p.storage.object)

	// The same hand-rolled disclosure as the create form, for the same
	// reason: collapsing has to tell the page to re-clamp its scroll offset.
	p.body = container.NewPadded(grid)

	p.toggle = widgets.NewButton(lang.L("Monitoring"), theme.MenuDropUpIcon(), p.toggleOpen)
	p.toggle.Alignment = widget.ButtonAlignLeading
	p.toggle.Importance = widget.LowImportance

	p.panel = container.NewVBox(p.toggle, p.body)
	return p
}

// CanvasObject is the panel as the page places it.
func (p *Panel) CanvasObject() fyne.CanvasObject {
	return container.NewPadded(widgets.Card(p.panel))
}

func (p *Panel) toggleOpen() {
	if p.body.Visible() {
		p.body.Hide()
		p.toggle.SetIcon(theme.MenuDropDownIcon())
	} else {
		p.body.Show()
		p.toggle.SetIcon(theme.MenuDropUpIcon())
	}
	p.reflow()
}

// Apply takes one poll: sums the interface counters, differences them and
// the CPU jiffies against the previous poll, and pushes a sample onto every
// chart. The first poll only seeds the rates, so those two tiles show a dash
// until the second one; the gauges are drawn at once. Must run on the UI
// goroutine.
func (p *Panel) Apply(snap api.TrafficSnapshot) {
	p.applyAt(snap, time.Now())
}

func (p *Panel) applyAt(snap api.TrafficSnapshot, now time.Time) {
	var rx, tx uint64
	for _, t := range snap.ServerTraffic {
		rx += t.RXBytes
		tx += t.TXBytes
	}
	sys := snap.System
	busy, idle := sys.CPU.BusySeconds, sys.CPU.TotalSeconds-sys.CPU.BusySeconds

	// A poll on the heels of the previous one - the page reloads the counters
	// after every action as well as on its timer - has nothing to say about a
	// rate: a few milliseconds of CPU time is noise, not a load. The gauges
	// are still current; the rates wait for the next interval.
	if p.last.ok && now.Sub(p.last.at) < time.Second {
		setUsage(p.memory, sys.Memory)
		setUsage(p.storage, sys.Storage)
		return
	}

	if p.last.ok {
		seconds := now.Sub(p.last.at).Seconds()
		if seconds > 0 {
			p.upload.set(rate(tx, p.last.tx, seconds), formatRate(rate(tx, p.last.tx, seconds)), formatBytes(float64(tx)))
			p.download.set(rate(rx, p.last.rx, seconds), formatRate(rate(rx, p.last.rx, seconds)), formatBytes(float64(rx)))
		}
		if load, ok := cpuLoad(busy, idle, p.last.busy, p.last.idle); ok {
			p.cpu.set(load, formatPercent(load), lang.N("{{.Count}} cores", sys.CPU.Cores, map[string]any{"Count": sys.CPU.Cores}))
		}
	} else {
		p.upload.set(0, "", formatBytes(float64(tx)))
		p.download.set(0, "", formatBytes(float64(rx)))
		p.cpu.set(0, "", lang.N("{{.Count}} cores", sys.CPU.Cores, map[string]any{"Count": sys.CPU.Cores}))
	}

	setUsage(p.memory, sys.Memory)
	setUsage(p.storage, sys.Storage)

	p.last.at, p.last.rx, p.last.tx = now, rx, tx
	p.last.busy, p.last.idle = busy, idle
	p.last.ok = true
}

// rate is the bytes per second between two readings of a counter. A counter
// that went backwards - a server was stopped and its interface, with its
// total, disappeared from the sum - reads as zero rather than negative.
func rate(now, before uint64, seconds float64) float64 {
	if now < before {
		return 0
	}
	return float64(now-before) / seconds
}

// cpuLoad is the share of CPU time spent working between two readings;
// false when no time has passed or the counters went backwards.
func cpuLoad(busy, idle, prevBusy, prevIdle float64) (float64, bool) {
	if busy < prevBusy || idle < prevIdle {
		return 0, false
	}
	total := (busy - prevBusy) + (idle - prevIdle)
	if total <= 0 {
		return 0, false
	}
	return (busy - prevBusy) / total, true
}

// setUsage fills a gauge tile from a capacity reading, dash when there is
// none.
func setUsage(t *tile, u api.UsageMetrics) {
	if u.Total == 0 {
		t.set(0, "", "—")
		return
	}
	ratio := float64(u.Used) / float64(u.Total)
	t.set(ratio, formatPercent(ratio), formatUsage(float64(u.Used), float64(u.Total)))
}
