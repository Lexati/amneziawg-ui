package dashboard

import (
	"image/color"
	"testing"
	"time"

	"amneziawg-web-ui/web-ui/api"
)

func snapshot(rx, tx uint64, busy, total float64) api.TrafficSnapshot {
	return api.TrafficSnapshot{
		ServerTraffic: map[string]api.InterfaceTraffic{
			"a": {RXBytes: rx / 2, TXBytes: tx / 2},
			"b": {RXBytes: rx - rx/2, TXBytes: tx - tx/2},
		},
		System: api.SystemMetrics{
			CPU:     api.CPUMetrics{Cores: 4, BusySeconds: busy, TotalSeconds: total},
			Memory:  api.UsageMetrics{Total: 8 << 30, Used: 2 << 30},
			Storage: api.UsageMetrics{Total: 100 << 30, Used: 25 << 30},
		},
	}
}

// The first poll can only seed the rates; the gauges are drawn from it at
// once. The second poll turns the differences into a speed and a load.
func TestRatesNeedTwoPolls(t *testing.T) {
	p := New(5*time.Second, func() {})
	start := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	p.applyAt(snapshot(1000, 2000, 10, 100), start)
	if p.upload.value.Text != "—" || p.cpu.value.Text != "—" {
		t.Errorf("rates after one poll = %q / %q, want dashes", p.upload.value.Text, p.cpu.value.Text)
	}
	if p.upload.foot.Text != "1.95 KB" || p.memory.value.Text != "25.0%" || p.storage.foot.Text != "25.00 / 100.00 GB" {
		t.Errorf("gauges after one poll: sent %q, memory %q, storage %q", p.upload.foot.Text, p.memory.value.Text, p.storage.foot.Text)
	}

	// 5 s later: 10240 more bytes out, 5120 in; 2 of 8 CPU seconds busy.
	p.applyAt(snapshot(1000+5120, 2000+10240, 12, 108), start.Add(5*time.Second))
	if p.upload.value.Text != "2.00 KB/s" || p.download.value.Text != "1.00 KB/s" {
		t.Errorf("rates = %q up / %q down", p.upload.value.Text, p.download.value.Text)
	}
	if p.cpu.value.Text != "25.0%" || p.cpu.foot.Text != "4 cores" {
		t.Errorf("cpu = %q, %q", p.cpu.value.Text, p.cpu.foot.Text)
	}
	if got := len(p.upload.chart.values); got != 1 {
		t.Errorf("chart has %d samples after two polls, want 1", got)
	}
}

// The page reloads the counters after every action as well as on its timer,
// so two polls can land milliseconds apart. The second one must not turn
// that sliver into a rate or a chart sample.
func TestPollsTooCloseTogetherOnlyUpdateGauges(t *testing.T) {
	p := New(5*time.Second, func() {})
	start := time.Now()
	p.applyAt(snapshot(1000, 2000, 10, 100), start)
	p.applyAt(snapshot(1000+5120, 2000+10240, 10.01, 100.01), start.Add(20*time.Millisecond))
	if p.upload.value.Text != "—" || p.cpu.value.Text != "—" || len(p.upload.chart.values) != 0 {
		t.Errorf("a 20ms poll produced %q / %q and %d samples", p.upload.value.Text, p.cpu.value.Text, len(p.upload.chart.values))
	}
	// The interval after that is measured from the first poll, not the burst.
	p.applyAt(snapshot(1000+5120, 2000+10240, 12, 108), start.Add(5*time.Second))
	if p.upload.value.Text != "2.00 KB/s" || p.cpu.value.Text != "25.0%" {
		t.Errorf("rates = %q / %q", p.upload.value.Text, p.cpu.value.Text)
	}
}

// A server that stops takes its interface counters out of the sum, so the
// total goes backwards; that is not negative traffic.
func TestCountersGoingBackwardsReadAsZero(t *testing.T) {
	p := New(5*time.Second, func() {})
	start := time.Now()
	p.applyAt(snapshot(5000, 9000, 10, 100), start)
	p.applyAt(snapshot(1000, 2000, 9, 99), start.Add(5*time.Second))
	if p.upload.value.Text != "0 B/s" || p.download.value.Text != "0 B/s" {
		t.Errorf("rates = %q / %q, want zero", p.upload.value.Text, p.download.value.Text)
	}
	if p.cpu.value.Text != "—" {
		t.Errorf("cpu = %q, want a dash for counters that went backwards", p.cpu.value.Text)
	}
}

// The chart keeps exactly the window: Window / interval samples, oldest out
// first.
func TestChartKeepsTheWindow(t *testing.T) {
	s := newSparkline(styleColor(), 3, 0)
	for i := 1; i <= 5; i++ {
		s.Push(float64(i))
	}
	if len(s.values) != 3 || s.values[0] != 3 || s.values[2] != 5 {
		t.Errorf("values = %v, want [3 4 5]", s.values)
	}
	// It also has to rasterise without panicking at any size, full or not.
	for _, w := range []int{0, 1, 2, 50, 300} {
		s.draw(w, 40)
	}
	s.draw(300, 1)
	newSparkline(styleColor(), 60, 1).draw(200, 40)
}

func styleColor() color.NRGBA { return color.NRGBA{R: 1, G: 2, B: 3, A: 255} }
