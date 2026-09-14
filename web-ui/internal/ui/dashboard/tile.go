package dashboard

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"amneziawg-web-ui/web-ui/internal/ui/style"
	"amneziawg-web-ui/web-ui/internal/ui/widgets"
)

// tile is one cell of the grid: an icon and a caption, the current reading
// in large type, the chart of its history, and under a rule a second caption
// with the figure that does not move much - the total sent, the core count,
// the capacity in use.
type tile struct {
	value  *canvas.Text
	chart  *sparkline
	foot   *canvas.Text
	object fyne.CanvasObject
}

// newTile builds the cell. capacity is how many samples the chart keeps and
// ceiling pins its top (zero to autoscale); see sparkline.
func newTile(icon fyne.Resource, caption, footCaption string, tint color.NRGBA, capacity int, ceiling float64) *tile {
	t := &tile{chart: newSparkline(tint, capacity, ceiling)}

	// Captions are set in small capitals; ToUpper is what does that for the
	// Cyrillic translation as well.
	head := canvas.NewText(strings.ToUpper(caption), style.Muted)
	head.TextSize = 11
	head.TextStyle = fyne.TextStyle{Bold: true}
	headRow := container.NewHBox(widgets.SmallIcon(icon), container.NewCenter(head))

	t.value = canvas.NewText("—", style.Text)
	t.value.TextSize = 20
	t.value.TextStyle = fyne.TextStyle{Bold: true}

	footHead := canvas.NewText(strings.ToUpper(footCaption), style.Muted)
	footHead.TextSize = 11
	footHead.TextStyle = fyne.TextStyle{Bold: true}

	t.foot = canvas.NewText("—", style.Text)
	t.foot.TextSize = 16
	t.foot.TextStyle = fyne.TextStyle{Bold: true}

	t.object = container.NewVBox(
		headRow,
		t.value,
		t.chart,
		widgets.Separator(),
		container.NewCenter(footHead),
		container.NewCenter(t.foot),
	)
	return t
}

// set pushes a sample onto the chart and shows both figures. An empty value
// leaves the chart alone and shows a dash, for a reading that needs a second
// sample before it means anything. Must run on the UI goroutine.
func (t *tile) set(sample float64, value, foot string) {
	if value == "" {
		value = "—"
	} else {
		t.chart.Push(sample)
	}
	t.value.Text = value
	t.value.Refresh()
	t.foot.Text = foot
	t.foot.Refresh()
}
